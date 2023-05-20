package mqtt

import (
	"context"
	"github.com/google/uuid"
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

	subscribers          sync.Map
	subscriberOpts       sync.Map // SubscribeOptionMap
	subscriberReqOpts    sync.Map // SubscribeOptionMap
	subscriberRespOpts   sync.Map // SubscribeOptionMap
	subscriberUploadOpts sync.Map // SubscribeOptionMap
	responseMap          sync.Map

	//	sync.RWMutex
	started bool

	baseCtx context.Context
	err     error
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		Broker:               nil,
		brokerOpts:           []broker.Option{},
		subscribers:          sync.Map{},
		subscriberReqOpts:    sync.Map{},
		subscriberRespOpts:   sync.Map{},
		subscriberUploadOpts: sync.Map{},
		started:              false,
		baseCtx:              context.Background(),
		err:                  nil,
	}

	srv.init(opts...)
	//	srv.subscriberReqOpts.Store("topic", &SubscribeOption{})

	srv.brokerOpts = append(srv.brokerOpts, broker.WithOnConnect(func() {
		err := srv.doRegisterSubscriberMap()
		if err != nil {
			return
		}
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

	//s.err = s.Init()
	//if s.err != nil {
	//	log.Errorf("[mqtt] init broker failed: [%s]", s.err.Error())
	//	return s.err
	//}
	//
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

func (s *Server) PublishUpload(ctx context.Context, topic string, msg any, binder broker.Binder) (any, error) {
	err := s.Broker.PublishUpload(topic, msg, broker.WithPublishContext(ctx))
	return nil, err
}

func (s *Server) PublishReq(ctx context.Context, topic string, msg any, binder broker.Binder) (any, error) {
	_, ok := s.subscriberOpts.Load(topic)
	if !ok {
		handler := func(ctx1 context.Context, event broker.Event) (broker.RespEvent, error) {
			resp := event.Message()
			value, ok := resp.Headers.Headers["Message-Id"]
			if !ok {
				return nil, nil
			}
			ch, ok := s.responseMap.LoadAndDelete(value)
			if !ok {
				return nil, nil
			}
			rr := resp.GetBody()
			ch.(chan any) <- rr
			return nil, nil
		}
		s.RegisterSubscriberResp(ctx, topic, handler, binder)
	}
	msgId, _ := uuid.NewRandom()
	ch := make(chan any, 1)
	s.responseMap.Store(msgId.String(), ch)
	err := s.Broker.PublishReq(topic,
		msg,
		broker.WithPublishContext(ctx),
		broker.PublishContextWithValue(broker.MessageId, msgId.String()))
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-ch:
		return resp, nil
	}
	return nil, nil
}

func (s *Server) RegisterSubscriber(ctx context.Context, topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) error {
	opts = append(opts, broker.WithSubscribeContext(ctx))

	if s.started {
		return s.doRegisterSubscriber(topic, handler, binder, opts...)
	} else {
		s.subscriberOpts.Store(topic, &SubscribeOption{handler: handler, binder: binder, opts: opts})
	}
	return nil
}

func (s *Server) RegisterSubscriberReq(ctx context.Context, topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) error {
	opts = append(opts, broker.WithSubscribeContext(ctx))
	s.subscriberReqOpts.Store(topic, &SubscribeOption{handler: handler, binder: binder, opts: opts})

	if s.started {
		return s.doRegisterSubscriberReq(topic, handler, binder, opts...)
	} else {
	}
	return nil
}

func (s *Server) RegisterSubscriberResp(ctx context.Context, topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) error {
	opts = append(opts, broker.WithSubscribeContext(ctx))
	_, ok := s.subscriberRespOpts.LoadOrStore(topic, &SubscribeOption{handler: handler, binder: binder, opts: opts})
	if ok {
		return nil
	}

	if s.started {
		return s.doRegisterSubscriberResp(topic, handler, binder, opts...)
	} else {
	}
	return nil
}

func (s *Server) RegisterSubscriberUpload(ctx context.Context, topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) error {
	opts = append(opts, broker.WithSubscribeContext(ctx))
	s.subscriberUploadOpts.Store(topic, &SubscribeOption{handler: handler, binder: binder, opts: opts})

	if s.started {
		return s.doRegisterSubscriberUpload(topic, handler, binder, opts...)
	} else {
	}
	return nil
}

func (s *Server) doRegisterSubscriber(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) error {
	sub, err := s.Subscribe(topic, handler, binder, opts...)
	if err != nil {
		return err
	}
	s.subscribers.Store(topic, sub)
	return nil
}

func (s *Server) doRegisterSubscriberReq(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) error {
	sub, err := s.SubscribeReq(topic, handler, binder, opts...)
	if err != nil {
		return err
	}
	s.subscribers.Store(topic, sub)
	return nil
}

func (s *Server) doRegisterSubscriberResp(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) error {
	sub, err := s.SubscribeResp(topic, handler, binder, opts...)
	if err != nil {
		return err
	}
	s.subscribers.Store(topic, sub)
	return nil
}

func (s *Server) doRegisterSubscriberUpload(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) error {
	sub, err := s.Subscribe(topic, handler, binder, opts...)
	if err != nil {
		return err
	}

	s.subscribers.Store(topic, sub)

	return nil
}

func (s *Server) doRegisterSubscriberMap() error {
	s.subscriberOpts.Range(func(key, value any) bool {
		topic := key.(string)
		opt := value.(*SubscribeOption)
		_ = s.doRegisterSubscriber(topic, opt.handler, opt.binder, opt.opts...)
		return true
	})
	s.subscriberReqOpts.Range(func(key, value any) bool {
		topic := key.(string)
		opt := value.(*SubscribeOption)
		_ = s.doRegisterSubscriberReq(topic, opt.handler, opt.binder, opt.opts...)
		return true
	})
	s.subscriberRespOpts.Range(func(key, value any) bool {
		topic := key.(string)
		opt := value.(*SubscribeOption)
		_ = s.doRegisterSubscriberResp(topic, opt.handler, opt.binder, opt.opts...)
		return true
	})
	s.subscriberUploadOpts.Range(func(key, value any) bool {
		topic := key.(string)
		opt := value.(*SubscribeOption)
		_ = s.doRegisterSubscriberUpload(topic, opt.handler, opt.binder, opt.opts...)
		return true
	})
	return nil
}
