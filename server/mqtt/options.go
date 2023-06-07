package mqtt

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
)

type ServerOption func(o *Server)

// Middleware with service md option.
func Middleware(m ...middleware.Middleware) ServerOption {
	return func(o *Server) {
		//		o.md = m
	}
}

// Logger with server logger.
// Deprecated: use global logger instead.
func Logger(logger log.Logger) ServerOption {
	return func(s *Server) {}
}
