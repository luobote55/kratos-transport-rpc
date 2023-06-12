package mqtt

import (
	"crypto/tls"
	"github.com/go-kratos/kratos/v2/encoding"
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

func WithAddress(addrs []string) ServerOption {
	return func(s *Server) {
		s.Opts.Addrs = addrs
	}
}

func WithTLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		s.Opts.TLSConfig = c
	}
}

func WithAuth(username string, password string) ServerOption {
	return func(s *Server) {
		//		broker.WithAuth(username, password)
	}
}

func WithClientId(clientId string) ServerOption {
	return func(s *Server) {
		s.Opts.ClientId = clientId
	}
}

func WithCodec(c string) ServerOption {
	return func(s *Server) {
		s.codec = c
		s.Opts.Codec = encoding.GetCodec(c)
	}
}

func WithLogger(logger log.Logger) ServerOption {
	return func(s *Server) {
		s.Opts.Log = log.NewHelper(logger)
	}
}
