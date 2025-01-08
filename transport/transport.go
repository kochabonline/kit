package transport

import (
	"context"
	"net"
	"strconv"
)

type Server interface {
	Run() error
	Shutdown(context.Context) error
}

func ValidateAddress(addr string) bool {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}

	p, err := strconv.Atoi(port)
	if err != nil {
		return false
	}
	if p < 1 || p > 65535 {
		return false
	}

	return true
}
