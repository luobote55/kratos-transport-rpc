package mqtt

import (
	"crypto/tls"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"kratos-transport-rpc/broker"
	"kratos-transport-rpc/broker/mqtt"
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

// WithBrokerOptions MQ代理配置
func WithBrokerOptions(opts ...broker.Option) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, opts...)
	}
}

func WithAddress(addrs []string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithAddress(addrs...))
	}
}

func WithTLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		if c != nil {
			s.brokerOpts = append(s.brokerOpts, broker.WithEnableSecure(true))
		}
		s.brokerOpts = append(s.brokerOpts, broker.WithTLSConfig(c))
	}
}

func WithCleanSession(enable bool) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, mqtt.WithCleanSession(enable))
	}
}

func WithAuth(username string, password string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, mqtt.WithAuth(username, password))
	}
}

func WithClientId(clientId string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, mqtt.WithClientId(clientId))
	}
}

func WithFromTo(ft bool) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, mqtt.WithFromTo(ft))
	}
}

func WithCodec(c string) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithCodec(c))
	}
}

func WithLogger(logger log.Logger) ServerOption {
	return func(s *Server) {
		s.brokerOpts = append(s.brokerOpts, broker.WithLogger(logger))
	}
}
