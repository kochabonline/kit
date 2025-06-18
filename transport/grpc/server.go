package grpc

import (
	"context"
	"net"

	"google.golang.org/grpc"

	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/transport"
)

var _ transport.Server = (*Server)(nil)

const (
	// DefaultAddr is the default address for the server.
	defaultAddr = ":50051"
)

type Server struct {
	addr   string
	server *grpc.Server
	log    *log.Logger
}

type Option func(*Server)

func WithLogger(log *log.Logger) Option {
	return func(s *Server) {
		s.log = log
	}
}

func NewServer(addr string, opts ...Option) *Server {
	s := &Server{
		addr:   addr,
		server: grpc.NewServer(),
		log:    log.DefaultLogger,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Server) Run() error {
	if s.server == nil {
		return grpc.ErrServerStopped
	}

	if ok := transport.ValidateAddress(s.addr); !ok {
		s.log.Warn().Msgf("invalid address %s, using default address: %s", s.addr, defaultAddr)
		s.addr = defaultAddr
	}

	listen, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.log.Info().Msgf("grpc server listening on %s", s.addr)

	return s.server.Serve(listen)
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return grpc.ErrServerStopped
	}

	s.server.GracefulStop()
	return nil
}
