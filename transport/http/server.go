package http

import (
	"context"
	"net/http"

	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/log/zerolog"
	"github.com/kochabonline/kit/transport"
)

var _ transport.Server = (*Server)(nil)

const (
	// DefaultAddr is the default address for the server.
	defaultAddr = ":8080"
)

type Server struct {
	server *http.Server
	log    *log.Helper
}

type Option func(*Server)

func WithLogger(logger *log.Helper) Option {
	return func(s *Server) {
		s.log = logger
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

	return s
}

func (s *Server) Run() error {
	if ok := transport.ValidateAddress(s.server.Addr); !ok {
		s.log.Warnf("invalid address %s, using default address: %s", s.server.Addr, defaultAddr)
		s.server.Addr = defaultAddr
	}
	s.log.Infof("http server listening on %s", s.server.Addr)

	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
