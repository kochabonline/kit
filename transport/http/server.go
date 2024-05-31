package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"

	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/log/zerolog"
	"github.com/kochabonline/kit/transport"
	"github.com/kochabonline/kit/transport/http/metrics/prometheus"
)

var _ transport.Server = (*Server)(nil)

const (
	// DefaultAddr is the default address for the server.
	defaultAddr = ":8080"
)

type Server struct {
	server  *http.Server
	log     *log.Helper
	options Options
}

type Option func(*Server)

func WithLogger(logger *log.Helper) Option {
	return func(s *Server) {
		s.log = logger
	}
}

func WithMetricsOptions(metrics MetricsOptions) Option {
	return func(s *Server) {
		_ = metrics.init()
		s.options.Metrics = metrics
	}
}

func WithSwagOptions(swag SwagOptions) Option {
	return func(s *Server) {
		_ = swag.init()
		s.options.Swag = swag
	}
}

func WithHealthOptions(health HealthOptions) Option {
	return func(s *Server) {
		_ = health.init()
		s.options.Health = health
	}
}

func NewServer(addr string, handler http.Handler, opts ...Option) *Server {
	s := &Server{
		server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
		log: log.NewHelper(zerolog.New()),
	}

	for _, opt := range opts {
		opt(s)
	}

	// addon handlers
	if r, ok := s.server.Handler.(*gin.Engine); ok {
		handleMetrics(s, r)
		handleSwag(s, r)
		handleHealth(s, r)
	}

	return s
}

func (s *Server) Run() error {
	if s.server == nil {
		return http.ErrServerClosed
	}

	if ok := transport.ValidateAddress(s.server.Addr); !ok {
		s.log.Warnf("invalid address %s, using default address: %s", s.server.Addr, defaultAddr)
		s.server.Addr = defaultAddr
	}
	s.log.Infof("http server listening on %s", s.server.Addr)

	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return http.ErrServerClosed
	}

	return s.server.Shutdown(ctx)
}

func handleMetrics(s *Server, r *gin.Engine) {
	if s.options.Metrics.Enabled {
		r.GET(s.options.Metrics.Path, gin.WrapH(promhttp.HandlerFor(prometheus.Registry, promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		})))
	}
}

func handleSwag(s *Server, r *gin.Engine) {
	if s.options.Swag.Enabled {
		r.GET(s.options.Swag.Path, ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
}

func handleHealth(s *Server, r *gin.Engine) {
	if s.options.Health.Enabled {
		r.GET(s.options.Health.Path, func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}
}
