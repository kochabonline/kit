package app

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/transport"
)

var (
	closeFuncs      = make([]func(), 0)
	closeFuncsMutex sync.Mutex
)

type App struct {
	ctx             context.Context
	servers         []transport.Server
	sigs            []os.Signal
	closeFuncs      []func()
	shutdownTimeout time.Duration
	log             log.Helper
}

type Option func(*App)

func WithServer(s ...transport.Server) Option {
	return func(a *App) {
		a.servers = append(a.servers, s...)
	}
}

func WithContext(ctx context.Context) Option {
	return func(a *App) {
		a.ctx = ctx
	}
}

func WithSignal(sig ...os.Signal) Option {
	return func(a *App) {
		a.sigs = append(a.sigs, sig...)
	}
}

func WithCloseFuncs(f ...func()) Option {
	return func(a *App) {
		a.closeFuncs = append(a.closeFuncs, f...)
	}
}

func WithShutdownTimeout(timeout time.Duration) Option {
	return func(a *App) {
		a.shutdownTimeout = timeout
	}
}

func WithLogger(log log.Helper) Option {
	return func(a *App) {
		a.log = log
	}
}

func NewApp(servers []transport.Server, opts ...Option) *App {
	app := &App{
		ctx:             context.Background(),
		servers:         servers,
		sigs:            []os.Signal{os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT},
		shutdownTimeout: 30 * time.Second,
		closeFuncs:      make([]func(), 0),
		log:             log.DefaultLogger,
	}

	for _, opt := range opts {
		opt(app)
	}

	return app
}

func (a *App) Run() error {
	eg, ctx := errgroup.WithContext(a.ctx)
	wg := sync.WaitGroup{}

	for _, server := range a.servers {
		server := server
		eg.Go(func() error {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
			defer cancel()

			return server.Shutdown(shutdownCtx)
		})

		wg.Add(1)
		eg.Go(func() error {
			wg.Done()
			return server.Run()
		})
	}

	// Wait for all servers to start
	wg.Wait()

	// Handle signals
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, a.sigs...)

	eg.Go(func() error {
		select {
		case signal := <-ch:
			a.log.Infof("received signal %s, shutting down", signal)
			a.close()
			return context.Canceled
		case <-ctx.Done():
			return nil
		}
	})

	// Wait for all goroutines to finish
	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}

// Close closes the app.
func (a *App) close() {
	wg := sync.WaitGroup{}
	closeFuncs := append(closeFuncs, a.closeFuncs...)
	wg.Add(len(closeFuncs))
	for _, close := range closeFuncs {
		go func(fn func()) {
			defer wg.Done()
			fn()
		}(close)
	}
	wg.Wait()
}

// AddCloseFuncs adds close functions to the global close functions.
func AddCloseFuncs(f ...func()) {
	closeFuncsMutex.Lock()
	defer closeFuncsMutex.Unlock()
	closeFuncs = append(closeFuncs, f...)
}
