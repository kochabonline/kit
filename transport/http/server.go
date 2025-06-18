package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"

	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/transport"
	"github.com/kochabonline/kit/transport/http/metrics/prometheus"
)

var _ transport.Server = (*Server)(nil)

const (
	// DefaultAddr is the default address for the server.
	defaultName = "http"
	defaultAddr = ":8080"
)

// Meta is the metadata of the server.
type Meta struct {
	Name string
}

type Server struct {
	Meta
	server  *http.Server
	log     *log.Logger
	options Options
}

type Option func(*Server)

func WithLogger(log *log.Logger) Option {
	return func(s *Server) {
		s.log = log
	}
}

func WithMetricsOptions(metrics MetricsOption) Option {
	return func(s *Server) {
		if err := metrics.init(); err != nil {
			s.log.Error().Err(err).Send()
			return
		}
		s.options.Metrics = metrics
	}
}

func WithSwagOptions(swag SwagOption) Option {
	return func(s *Server) {
		if err := swag.init(); err != nil {
			s.log.Error().Err(err).Send()
			return
		}
		s.options.Swag = swag
	}
}

func WithHealthOptions(health HealthOption) Option {
	return func(s *Server) {
		if err := health.init(); err != nil {
			s.log.Error().Err(err).Send()
			return
		}
		s.options.Health = health
	}
}

func NewServer(addr string, handler http.Handler, opts ...Option) *Server {
	s := &Server{
		server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
		log: log.DefaultLogger,
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
	if s.Name == "" {
		s.Name = defaultName
	}

	if ok := transport.ValidateAddress(s.server.Addr); !ok {
		s.log.Warn().Msgf("invalid address %s, using default address: %s", s.server.Addr, defaultAddr)
		s.server.Addr = defaultAddr
	}
	s.log.Info().Msgf("%s server listening on %s", s.Name, s.server.Addr)

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
		prom := prometheus.NewPrometheus(prometheus.Config{
			Path:               s.options.Metrics.Path,
			EnabledGoCollector: true,
		})

		r.GET(prom.Config.Path, gin.WrapH(promhttp.HandlerFor(prom.Registry, promhttp.HandlerOpts{
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
