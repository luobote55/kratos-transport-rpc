package mqtt

import (
	"context"
	"github.com/luobote55/kratos-transport-rpc/broker"
	"github.com/luobote55/kratos-transport-rpc/broker/mqtt"
	"net/url"
	"strings"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
)

type SubscriberMap map[string]broker.Subscriber

type SubscribeOption struct {
	handler broker.Handler
	binder  broker.Binder
	opts    []broker.SubscribeOption
}
type SubscribeOptionMap map[string]*SubscribeOption

type Server struct {
	broker.Broker
	brokerOpts []broker.Option

	subscribers    sync.Map
	subscriberOpts sync.Map // SubscribeOptionMap

	//	sync.RWMutex
	started bool

	baseCtx context.Context
	err     error
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		Broker:      nil,
		brokerOpts:  []broker.Option{},
		subscribers: sync.Map{},
		started:     false,
		baseCtx:     context.Background(),
		err:         nil,
	}

	srv.init(opts...)
	//	srv.subscriberReqOpts.Store("topic", &SubscribeOption{})

	srv.brokerOpts = append(srv.brokerOpts, broker.WithOnConnect(func() {
		err := srv.doRegisterSubscriberMap()
		if err != nil {
			return
		}
		srv.started = true
	}))
	srv.Broker = mqtt.NewBroker(srv.brokerOpts...)

	return srv
}

func (s *Server) InitServer(opts ...ServerOption) {
	s.brokerOpts = []broker.Option{}
	s.init(opts...)
	//	srv.subscriberReqOpts.Store("topic", &SubscribeOption{})

	s.brokerOpts = append(s.brokerOpts, broker.WithOnConnect(func() {
		err := s.doRegisterSubscriberMap()
		if err != nil {
			return
		}
	}))
	s.Broker = mqtt.NewBroker(s.brokerOpts...)
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}
}

func (s *Server) Name() string {
	return "mqtt"
}

func (s *Server) Endpoint() (*url.URL, error) {
	if s.err != nil {
		return nil, s.err
	}

	addr := s.Address()
	if !strings.HasPrefix(addr, "tcp://") {
		addr = "tcp://" + addr
	}

	return url.Parse(addr)
}

func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}

	if s.started {
		return nil
	}

	s.err = s.ConnectRetry()
	if s.err != nil {
		return s.err
	}

	log.Infof("[mqtt] server listening on: %s", s.Address())

	s.baseCtx = ctx
	s.started = true

	return nil
}

func (s *Server) Stop(_ context.Context) error {
	log.Info("[mqtt] server stopping")
	s.started = false
	return s.Disconnect()
}

func (s *Server) RegisterSubscriber(ctx context.Context, topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) error {
	opts = append(opts, broker.WithSubscribeContext(ctx))
	s.subscriberOpts.Store(topic, &SubscribeOption{handler: handler, binder: binder, opts: opts})

	if s.started {
		return s.doRegisterSubscriber(topic, handler, binder, opts...)
	}
	return nil
}

func (s *Server) doRegisterSubscriber(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) error {
	_, err := s.Subscribe(topic, handler, binder, opts...)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) doRegisterSubscriberMap() error {
	s.subscriberOpts.Range(func(key, value any) bool {
		topic := key.(string)
		opt := value.(*SubscribeOption)
		_ = s.doRegisterSubscriber(topic, opt.handler, opt.binder, opt.opts...)
		return true
	})
	return nil
}
