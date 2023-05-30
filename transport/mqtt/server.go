package mqtt

import (
	"context"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/google/uuid"
	"github.com/luobote55/kratos-transport-rpc/broker"
	"github.com/luobote55/kratos-transport-rpc/broker/mqtt"
	"github.com/pkg/errors"
	"net/url"
	"strings"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

const SupportPackageIsVersion1 = true

type Handler func(ctx context.Context, req interface{}) (interface{}, error)

// /sys/[ tenantid ]/[ sn ]    + /service/[ deviceType ]
func (s *Server) RegRouteUpload(topic string, r *Router) {
	s.RegisterSubscriberUpload(context.Background(), s.topicPre+topic, func(ctx context.Context, msg broker.Event) (broker.Any, error) {
		id := msg.Message().Headers.Headers[broker.Identifier]
		return r.route[id].Handler(ctx, msg.Data())
	}, func(id string) broker.Any {
		value, ok := r.route[id]
		if !ok {
			return nil
		}
		return value.Binder(id)
	})
}

func (s *Server) RegRouteReq(topic string, r *Router) {
	s.RegisterSubscriberReq(context.Background(), s.topicPre+topic, func(ctx context.Context, msg broker.Event) (broker.Any, error) {
		id := msg.Message().Headers.Headers[broker.Identifier]
		return r.route[id].Handler(ctx, msg.Data())
	}, func(id string) broker.Any {
		value, ok := r.route[id]
		if !ok {
			return nil
		}
		return value.Binder(id)
	})
}

func (s *Server) RegReq(prefix, topic string, handler Handler, binder broker.Binder) {
	s.RegisterSubscriberReq(context.Background(), prefix+topic, func(ctx context.Context, msg broker.Event) (broker.Any, error) {
		resp, err := handler(ctx, msg.Data())
		if err != nil {
			return nil, err
		}
		return resp, nil
	}, binder)
	return
}

func (s *Server) RegResp(prefix, topic string, handler Handler, binder broker.Binder) {
	s.RegisterSubscriber(context.Background(), prefix+topic, func(ctx context.Context, msg broker.Event) (broker.Any, error) {
		_, err := handler(ctx, msg.Data())
		return nil, err
	}, binder)
	return
}

func (s *Server) RegUpload(prefix, topic string, handler Handler, binder broker.Binder) {
	s.RegisterSubscriber(context.Background(), prefix+topic, func(ctx context.Context, msg broker.Event) (broker.Any, error) {
		_, err := handler(ctx, msg.Data())
		return nil, err
	}, binder)
	return
}

func (d *Server) Service(createDevice func(deviceId string)) {
	d.CreateDevice = createDevice
}

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

	topicPre     string
	md           []middleware.Middleware
	af           []middleware.Middleware
	CreateDevice func(deviceId string)

	log log.Logger
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

func (s *Server) PublishUpload(ctx context.Context, topic string, msg broker.Any, binder broker.Binder) (broker.Any, error) {
	err := s.Broker.PublishUpload(topic, msg, broker.WithPublishContext(ctx))
	return nil, err
}

func (s *Server) PublishReq(ctx context.Context, topic string, msg broker.Any, binder broker.Binder) (broker.Any, error) {
	_, ok := s.subscriberOpts.Load(topic)
	if !ok {
		handler := func(ctx1 context.Context, event broker.Event) (broker.Any, error) {
			resp := event.Message()
			value, ok := resp.Headers.Headers[broker.MessageId]
			if !ok {
				return nil, nil
			}
			ch, ok := s.responseMap.LoadAndDelete(value)
			if !ok {
				return nil, nil
			}
			data := event.Data()
			value, ok = resp.Headers.Headers[broker.Identifier]
			if !ok {
				return nil, nil
			}
			if value == broker.Failed {
				ch.(chan broker.Any) <- data.(*broker.CommonReply).Msg
				return nil, nil
			}
			ch.(chan broker.Any) <- data
			return nil, nil
		}
		s.RegisterSubscriberResp(ctx, topic, handler, func(id string) broker.Any {
			if id == broker.Failed {
				return &broker.CommonReply{}
			}
			return binder(id)
		})
	}
	msgId, _ := uuid.NewRandom()
	ch := make(chan broker.Any, 1)
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
		s.responseMap.Delete(msgId.String())
		return nil, ctx.Err()
	case resp := <-ch:
		if errStr, ok := resp.(string); ok {
			return nil, errors.New(errStr)
		}
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
	sub, err := s.SubscribeUpload(topic, handler, binder, opts...)
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

func (s *Server) Route(prefix string) *Router {
	return newRouter(prefix, s)
}
