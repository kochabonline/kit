package app

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/transport"
)

// 默认配置值
const (
	DefaultShutdownTimeout = 30 * time.Second
	DefaultCancelTimeout   = 30 * time.Second
)

// 默认关闭信号
var DefaultSignals = []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT}

var (
	ErrAlreadyStarted = errors.New("application already started")
	ErrCleanupPanic   = errors.New("cleanup function panicked")
)

// Option 定义 Application 的配置选项
type Option interface {
	apply(*Application)
}

// optionFunc 包装函数以实现 Option 接口
type optionFunc func(*Application)

func (f optionFunc) apply(app *Application) {
	f(app)
}

// Application 管理服务器和清理函数的生命周期
type Application struct {
	ctx             context.Context
	cancel          context.CancelFunc
	shutdownTimeout time.Duration
	signals         []os.Signal
	servers         []transport.Server
	cancelFuncs     []CancelFunc
	cancelTimeout   time.Duration
	mu              sync.RWMutex
	started         bool
}

// CancelFunc 具有可选超时的清理函数
type CancelFunc struct {
	Name    string
	Fn      func(context.Context) error
	Timeout time.Duration
}

// NewApplication 使用给定选项创建新的应用实例
func New(options ...Option) *Application {
	app := &Application{
		shutdownTimeout: DefaultShutdownTimeout,
		cancelTimeout:   DefaultCancelTimeout,
		signals:         make([]os.Signal, len(DefaultSignals)),
		servers:         make([]transport.Server, 0),
		cancelFuncs:     make([]CancelFunc, 0),
	}

	// 复制默认信号
	copy(app.signals, DefaultSignals)

	// 设置默认上下文
	app.ctx, app.cancel = context.WithCancel(context.Background())

	// 应用选项
	for _, opt := range options {
		opt.apply(app)
	}

	return app
}

// WithContext 设置应用的根上下文
func WithContext(ctx context.Context) Option {
	return optionFunc(func(app *Application) {
		if ctx != nil {
			app.ctx, app.cancel = context.WithCancel(ctx)
		}
	})
}

// WithShutdownTimeout 设置服务器关闭的超时时间
func WithShutdownTimeout(timeout time.Duration) Option {
	return optionFunc(func(app *Application) {
		if timeout > 0 {
			app.shutdownTimeout = timeout
		}
	})
}

// WithCancelTimeout 设置清理函数的默认超时时间
func WithCancelTimeout(timeout time.Duration) Option {
	return optionFunc(func(app *Application) {
		if timeout > 0 {
			app.cancelTimeout = timeout
		}
	})
}

// WithSignals 设置用于优雅关闭的自定义信号
func WithSignals(signals ...os.Signal) Option {
	return optionFunc(func(app *Application) {
		if len(signals) > 0 {
			app.signals = make([]os.Signal, len(signals))
			copy(app.signals, signals)
		}
	})
}

// WithServer 向应用添加服务器
func WithServer(server transport.Server) Option {
	return optionFunc(func(app *Application) {
		if server != nil {
			app.servers = append(app.servers, server)
		}
	})
}

// WithServers 向应用添加多个服务器
func WithServers(servers ...transport.Server) Option {
	return optionFunc(func(app *Application) {
		for _, server := range servers {
			if server != nil {
				app.servers = append(app.servers, server)
			}
		}
	})
}

// WithCancel 添加在关闭期间执行的清理函数
func WithCancel(name string, fn func(context.Context) error, timeout time.Duration) Option {
	return optionFunc(func(app *Application) {
		if fn == nil {
			log.Warn().Str("name", name).Msg("nil cancel function ignored")
			return
		}

		if timeout == 0 {
			timeout = app.cancelTimeout
		}

		cleanup := CancelFunc{
			Name:    name,
			Fn:      fn,
			Timeout: timeout,
		}

		app.cancelFuncs = append(app.cancelFuncs, cleanup)
	})
}

// AddServer 在运行时向应用添加服务器
func (app *Application) AddServer(server transport.Server) error {
	if server == nil {
		return errors.New("server cannot be nil")
	}

	app.mu.Lock()
	defer app.mu.Unlock()

	if app.started {
		log.Warn().Msg("attempted to add server after application started")
		return ErrAlreadyStarted
	}

	app.servers = append(app.servers, server)

	return nil
}

// RegisterCancel 在运行时向应用添加清理函数
func (app *Application) RegisterCancel(name string, fn func(context.Context) error, timeout time.Duration) error {
	if fn == nil {
		return errors.New("cancel function cannot be nil")
	}

	app.mu.Lock()
	defer app.mu.Unlock()

	if timeout == 0 {
		timeout = app.cancelTimeout
	}

	cancel := CancelFunc{
		Name:    name,
		Fn:      fn,
		Timeout: timeout,
	}

	app.cancelFuncs = append(app.cancelFuncs, cancel)

	return nil
}

// Start 启动所有服务器并阻塞直到关闭
func (app *Application) Start() error {
	app.mu.Lock()
	if app.started {
		app.mu.Unlock()
		return ErrAlreadyStarted
	}
	app.started = true
	servers := make([]transport.Server, len(app.servers))
	copy(servers, app.servers)
	signals := make([]os.Signal, len(app.signals))
	copy(signals, app.signals)
	app.mu.Unlock()

	if len(servers) == 0 {
		log.Info().Msg("no servers configured, starting signal handler only")
	}

	// 设置信号处理
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, signals...)
	defer signal.Stop(sigCh)

	// 创建错误组来管理协程
	eg, egCtx := errgroup.WithContext(app.ctx)

	// 启动服务器
	startedCh := make(chan struct{})
	if len(servers) > 0 {
		app.startServers(eg, egCtx, servers, startedCh)
	} else {
		close(startedCh)
	}

	// 处理关闭信号
	eg.Go(func() error {
		select {
		case sig := <-sigCh:
			log.Info().Str("signal", sig.String()).Msg("received shutdown signal")
			app.cancel()
			return nil
		case <-egCtx.Done():
			// context.Canceled 是正常的关闭，不应该作为错误返回
			if egCtx.Err() == context.Canceled {
				return nil
			}
			return egCtx.Err()
		}
	})

	// 等待关闭
	err := eg.Wait()

	// 如果有错误且不是正常的取消错误，则返回错误
	if err != nil && err != context.Canceled {
		return err
	}

	app.runCleanupTasks()

	return nil
}

// Stop 优雅地停止应用
func (app *Application) Stop() {
	app.cancel()
}

// startServers 启动所有配置的服务器
func (app *Application) startServers(eg *errgroup.Group, ctx context.Context, servers []transport.Server, startedCh chan struct{}) {
	startWg := sync.WaitGroup{}
	startWg.Add(len(servers))

	// 在单独的协程中启动每个服务器
	for _, server := range servers {
		// 服务器运行器
		eg.Go(func() error {
			defer startWg.Done()

			if err := server.Run(); err != nil {
				// http.ErrServerClosed 是正常关闭时的预期错误，不应该记录为错误
				if err != http.ErrServerClosed {
					return err
				}
			}
			return nil
		})

		// 服务器关闭处理器
		eg.Go(func() error {
			<-ctx.Done()

			shutdownCtx, cancel := context.WithTimeout(context.Background(), app.shutdownTimeout)
			defer cancel()

			if err := server.Shutdown(shutdownCtx); err != nil {
				return err
			}

			return nil
		})
	}

	// 等待所有服务器启动
	go func() {
		startWg.Wait()
		close(startedCh)
	}()
}

// runCleanupTasks 执行所有清理函数
func (app *Application) runCleanupTasks() {
	app.mu.RLock()
	cancelFuncs := make([]CancelFunc, len(app.cancelFuncs))
	copy(cancelFuncs, app.cancelFuncs)
	app.mu.RUnlock()

	if len(cancelFuncs) == 0 {
		return
	}

	// 并发执行清理函数
	eg := &errgroup.Group{}
	for _, cancel := range cancelFuncs {
		cancel := cancel
		eg.Go(func() error {
			return app.runCleanupTask(cancel)
		})
	}

	if err := eg.Wait(); err != nil {
		log.Error().Err(err).Msg("some cleanup functions failed")
	}
}

// runCleanupTask 执行单个带超时的清理函数
func (app *Application) runCleanupTask(cleanup CancelFunc) error {
	ctx, cancel := context.WithTimeout(context.Background(), cleanup.Timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).Str("cleanup", cleanup.Name).Msg("cleanup function panicked")
				done <- ErrCleanupPanic
			}
		}()
		done <- cleanup.Fn(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Error().Err(err).Str("cleanup", cleanup.Name).Msg("cleanup function failed")
		}
		return err
	case <-ctx.Done():
		log.Warn().Str("cleanup", cleanup.Name).Msg("cleanup function timed out")
		return ctx.Err()
	}
}

// Info 返回应用状态信息
func (app *Application) Info() ApplicationInfo {
	app.mu.RLock()
	defer app.mu.RUnlock()

	return ApplicationInfo{
		Started:      app.started,
		ServerCount:  len(app.servers),
		CleanupCount: len(app.cancelFuncs),
	}
}

// ApplicationInfo 提供应用状态信息
type ApplicationInfo struct {
	Started      bool `json:"started"`
	ServerCount  int  `json:"server_count"`
	CleanupCount int  `json:"cleanup_count"`
}

// 高级选项

// WithServerHealthCheck 添加具有健康检查功能的服务器
func WithServerHealthCheck(server transport.Server, healthCheck func() error) Option {
	return optionFunc(func(app *Application) {
		if server != nil {
			app.servers = append(app.servers, server)

			// 添加健康检查作为清理函数
			if healthCheck != nil {
				app.cancelFuncs = append(app.cancelFuncs, CancelFunc{
					Name: "health-check",
					Fn: func(ctx context.Context) error {
						return healthCheck()
					},
					Timeout: 5 * time.Second,
				})
			}
		}
	})
}

// WithGracefulShutdown 启用具有自定义预关闭钩子的优雅关闭
func WithGracefulShutdown(preShutdownHook func() error) Option {
	return optionFunc(func(app *Application) {
		if preShutdownHook != nil {
			app.cancelFuncs = append(app.cancelFuncs, CancelFunc{
				Name: "pre-shutdown-hook",
				Fn: func(ctx context.Context) error {
					return preShutdownHook()
				},
				Timeout: app.cancelTimeout,
			})
		}
	})
}

// WithStartupValidation 添加在服务器启动前运行的验证
func WithStartupValidation(validator func() error) Option {
	return optionFunc(func(app *Application) {
		if validator != nil {
			// 添加验证作为特殊的清理函数，优先运行
			validation := CancelFunc{
				Name: "startup-validation",
				Fn: func(ctx context.Context) error {
					return validator()
				},
				Timeout: 30 * time.Second,
			}

			// 插入到开头
			app.cancelFuncs = append([]CancelFunc{validation}, app.cancelFuncs...)
		}
	})
}
