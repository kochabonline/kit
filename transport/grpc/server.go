package grpc

import (
	"context"
	"net"

	"google.golang.org/grpc"

	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/log/zerolog"
	"github.com/kochabonline/kit/transport"
)

var _ transport.Server = (*Server)(nil)

const (
	// DefaultAddr is the default address for the server.
	defaultAddr = ":50051"
)

type Server struct {
	server *grpc.Server
	addr   string
	log    *log.Helper
}

type Option func(*Server)

func NewServer(addr string, opts ...Option) *Server {
	s := &Server{
		server: grpc.NewServer(),
		log:    log.NewHelper(zerolog.New()),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Server) Run() error {
	if ok := transport.ValidateAddress(s.addr); !ok {
		s.log.Warnf("invalid address %s, using default address: %s", s.addr, defaultAddr)
		s.addr = defaultAddr
	}

	listen, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.log.Infof("grpc server listening on %s", s.addr)

	return s.server.Serve(listen)
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.server.GracefulStop()
	return nil
}
