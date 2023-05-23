package mqtt

import (
	"github.com/luobote55/kratos-transport-rpc/broker"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-kratos/kratos/v2/log"
)

type mqttBroker struct {
	opts      broker.Options
	client    MQTT.Client
	addrs     []string
	logFilter logFilter
}

func NewBroker(opts ...broker.Option) broker.Broker {
	return newBroker(opts...)
}

func newClient(addrs []string, opts broker.Options, b *mqttBroker) MQTT.Client {
	cOpts := MQTT.NewClientOptions()

	// 是否清除会话，如果true，mqtt服务端将会清除掉
	cOpts.SetCleanSession(true)
	// 设置自动重连接
	cOpts.SetAutoReconnect(true)
	// 设置连接之后恢复订阅
	cOpts.SetResumeSubs(true)
	// 设置保活时间
	//cOpts.SetKeepAlive(10)
	// 设置最大重连时间间隔
	cOpts.SetMaxReconnectInterval(30)
	// 默认设置Client ID
	cOpts.SetClientID(generateClientId())

	// 连接成功回调
	cOpts.OnConnect = b.onConnect
	if opts.OnConnect != nil {
		cOpts.OnConnect = func(MQTT.Client) {
			opts.OnConnect()
		}
	}
	// 连接丢失回调
	cOpts.OnConnectionLost = b.onConnectionLost
	if opts.Disconncet != nil {
		cOpts.OnConnectionLost = func(client MQTT.Client, _ error) {
			opts.Disconncet()
		}
	}

	// 加入服务器地址列表
	for _, addr := range addrs {
		cOpts.AddBroker(addr)
	}

	if opts.TLSConfig != nil {
		cOpts.SetTLSConfig(opts.TLSConfig)
	}
	if auth, ok := opts.Context.Value(authKey{}).(*AuthRecord); ok && auth != nil {
		cOpts.SetUsername(auth.Username)
		cOpts.SetPassword(auth.Password)
	}
	if clientId, ok := opts.Context.Value(clientIdKey{}).(string); ok && clientId != "" {
		cOpts.SetClientID(clientId)
	}
	if enabled, ok := opts.Context.Value(cleanSessionKey{}).(bool); ok {
		cOpts.SetCleanSession(enabled)
	}
	if enabled, ok := opts.Context.Value(autoReconnectKey{}).(bool); ok {
		cOpts.SetAutoReconnect(enabled)
	}
	if enabled, ok := opts.Context.Value(resumeSubsKey{}).(bool); ok {
		cOpts.SetResumeSubs(enabled)
	}
	if enabled, ok := opts.Context.Value(orderMattersKey{}).(bool); ok {
		cOpts.SetOrderMatters(enabled)
	}

	return MQTT.NewClient(cOpts)
}

func newBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	b := &mqttBroker{
		opts:   options,
		client: nil,
		addrs:  options.Addrs,
		logFilter: logFilter{
			cache: cache{
				RWMutex:   sync.RWMutex{},
				data:      map[string]interface{}{},
				timestamp: map[string]int64{},
				timeout:   0,
			},
		},
	}
	if clientId, ok := options.Context.Value(clientIdKey{}).(string); ok && clientId != "" {
		b.opts.ClientId = clientId
	}
	if ft, ok := options.Context.Value(fromToKey{}).(bool); ok {
		b.opts.FromTo = ft
	}

	b.client = newClient(options.Addrs, options, b)

	return b
}

func (m *mqttBroker) Logger() *log.Helper {
	return m.Options().Log
}

func (m *mqttBroker) Name() string {
	return "MQTT"
}

func (m *mqttBroker) Options() broker.Options {
	return m.opts
}

func (m *mqttBroker) Address() string {
	return strings.Join(m.addrs, ",")
}

func (m *mqttBroker) Init(opts ...broker.Option) error {
	if m.client.IsConnected() {
		return errors.New("cannot init while connected")
	}

	for _, o := range opts {
		o(&m.opts)
	}

	m.addrs = setAddrs(m.opts.Addrs)
	//	m.client = newClient(m.addrs, m.opts, m)
	return nil
}

func (m *mqttBroker) Connect() error {
	if m.client.IsConnected() {
		return nil
	}

	t := m.client.Connect()

	if rs, err := checkClientToken(t); !rs {
		return err
	}

	return nil
}

func (m *mqttBroker) ConnectRetry() error {
	if m.client.IsConnected() {
		return nil
	}

	t := m.client.Connect()
	if rs, _ := checkClientTokenTimeout(t, 5*time.Second); rs {
		return nil
	}

	go func() {
		for {
			t := m.client.Connect()
			if rs, _ := checkClientTokenTimeout(t, 5*time.Second); !rs {
				time.Sleep(1 * time.Second)
				continue
			}
			break
		}
	}()

	return nil
}

func (m *mqttBroker) Disconnect() error {
	if !m.client.IsConnected() {
		return nil
	}
	m.client.Disconnect(0)
	return nil
}

func (m *mqttBroker) Publish(topic string, msg broker.Any, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(m.opts.Codec, msg)
	if err != nil {
		return err
	}

	return m.publish(topic, buf, opts...)
}

func checkProtoMessage(msg broker.Any) (*anypb.Any, error) {
	msgAny, ok := msg.(protoreflect.ProtoMessage)
	if !ok {
		if _, ok = msg.(string); !ok {
			return nil, errors.New("checkProtoMessage protoreflect.ProtoMessage, failed")
		}
		msgAny = &broker.CommonReply{
			Status: http.StatusBadRequest,
			Msg:    msg.(string),
		}
	}
	// 使用 Any 类型存储数据消息
	return anypb.New(msgAny)
}

func UnmarshalAny(reqAny *anypb.Any, msg broker.Any) error {
	msgAny, ok := msg.(protoreflect.ProtoMessage)
	if !ok {
		return errors.New("UnmarshalAny protoreflect.ProtoMessage, failed")
	}
	return reqAny.UnmarshalTo(msgAny)
}

func (m *mqttBroker) PublishReq(topic string, msg broker.Any, opts ...broker.PublishOption) error {
	msgAny, err := checkProtoMessage(msg)
	if err != nil {
		return err
	}
	options := broker.NewPublishOptions(opts...)
	var qos byte = 2
	var retained = false
	if value, ok := options.Context.Value(qosPublishKey{}).(byte); ok {
		qos = value
	}
	if value, ok := options.Context.Value(retainedPublishKey{}).(bool); ok {
		retained = value
	}
	header := broker.Headers{
		Headers: map[string]string{broker.MessageAct: "request"},
	}
	if value, ok := options.Context.Value(broker.MessageId).(string); ok {
		header.Headers[broker.MessageId] = value
	}
	if value, ok := options.Context.Value(broker.Identifier).(string); ok {
		header.Headers[broker.Identifier] = value
	}
	if m.opts.FromTo {
		if value, ok := options.Context.Value(broker.MeggageTo).(string); ok {
			header.Headers[broker.MeggageTo] = value
		}
		header.Headers[broker.MessageFrom] = m.opts.ClientId
	}
	req := &broker.Message{
		Headers: &header,
		Data:    msgAny,
	}
	buf, err := broker.Marshal(m.opts.Codec, &req)
	if err != nil {
		return err
	}
	ret := m.client.Publish(topic+"/req", qos, retained, buf)
	return ret.Error()
}

func (m *mqttBroker) PublishCommErrorResp(topic string, msg string, opts ...broker.PublishOption) error {
	msgAny, err := checkProtoMessage(msg)
	if err != nil {
		return err
	}
	options := broker.NewPublishOptions(opts...)
	var qos byte = 2
	var retained = false
	if value, ok := options.Context.Value(qosPublishKey{}).(byte); ok {
		qos = value
	}
	if value, ok := options.Context.Value(retainedPublishKey{}).(bool); ok {
		retained = value
	}
	header := broker.Headers{
		Headers: map[string]string{broker.MessageAct: "response"},
	}
	if value, ok := options.Context.Value(broker.MessageId).(string); ok {
		header.Headers[broker.MessageId] = value
	}
	header.Headers[broker.Identifier] = broker.Failed
	resp := &broker.Message{
		Headers: &header,
		Data:    msgAny,
	}
	buf, err := broker.Marshal(m.opts.Codec, resp)
	if err != nil {
		return err
	}
	ret := m.client.Publish(topic+"/resp", qos, retained, buf)
	return ret.Error()
}

func (m *mqttBroker) PublishResp(topic string, msg broker.Any, opts ...broker.PublishOption) error {
	msgAny, err := checkProtoMessage(msg)
	if err != nil {
		return err
	}
	options := broker.NewPublishOptions(opts...)
	var qos byte = 2
	var retained = false
	if value, ok := options.Context.Value(qosPublishKey{}).(byte); ok {
		qos = value
	}
	if value, ok := options.Context.Value(retainedPublishKey{}).(bool); ok {
		retained = value
	}
	header := broker.Headers{
		Headers: map[string]string{broker.MessageAct: "response"},
	}
	if value, ok := options.Context.Value(broker.MessageId).(string); ok {
		header.Headers[broker.MessageId] = value
	}
	if value, ok := options.Context.Value(broker.Identifier).(string); ok {
		header.Headers[broker.Identifier] = value
	}
	resp := &broker.Message{
		Headers: &header,
		Data:    msgAny,
	}
	buf, err := broker.Marshal(m.opts.Codec, resp)
	if err != nil {
		return err
	}
	ret := m.client.Publish(topic+"/resp", qos, retained, buf)
	return ret.Error()
}

func (m *mqttBroker) PublishUpload(topic string, msg broker.Any, opts ...broker.PublishOption) error {
	msgAny, err := checkProtoMessage(msg)
	if err != nil {
		return err
	}
	options := broker.NewPublishOptions(opts...)
	var qos byte = 2
	var retained = false
	if value, ok := options.Context.Value(qosPublishKey{}).(byte); ok {
		qos = value
	}
	if value, ok := options.Context.Value(retainedPublishKey{}).(bool); ok {
		retained = value
	}
	header := broker.Headers{
		Headers: map[string]string{broker.MessageAct: "upload"},
	}
	if value, ok := options.Context.Value(broker.MessageId).(string); ok {
		header.Headers[broker.MessageId] = value
	}
	if value, ok := options.Context.Value(broker.Identifier).(string); ok {
		header.Headers[broker.Identifier] = value
	}
	resp := &broker.Message{
		Headers: &header,
		Data:    msgAny,
	}
	buf, err := broker.Marshal(m.opts.Codec, resp)
	if err != nil {
		return err
	}
	ret := m.client.Publish(topic+"/upload", qos, retained, buf)
	return ret.Error()
}

func (m *mqttBroker) publish(topic string, buf []byte, opts ...broker.PublishOption) error {
	if !m.client.IsConnected() {
		return errors.New("not connected")
	}

	options := broker.NewPublishOptions(opts...)
	var qos byte = 2
	var retained = false

	if value, ok := options.Context.Value(qosPublishKey{}).(byte); ok {
		qos = value
	}
	if value, ok := options.Context.Value(retainedPublishKey{}).(bool); ok {
		retained = value
	}

	ret := m.client.Publish(topic, qos, retained, buf)
	return ret.Error()
}

func (m *mqttBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	if !m.client.IsConnected() {
		return nil, errors.New("not connected")
	}

	options := broker.NewSubscribeOptions(opts...)
	var qos byte = 2
	if value, ok := options.Context.Value(qosSubscribeKey{}).(byte); ok {
		qos = value
	}
	t := m.client.Subscribe(topic, qos, func(c MQTT.Client, mq MQTT.Message) {
		msg := broker.Message{}
		p := &publication{topic: mq.Topic(), msg: &msg}
		if err := broker.Unmarshal(m.opts.Codec, mq.Payload(), &msg); err != nil {
			p.err = err
			log.Error(err)
			return
		}

		if _, err := handler(m.opts.Context, p); err != nil {
			p.err = err
			log.Error(err)
		}
	})

	if rs, err := checkClientToken(t); !rs {
		return nil, err
	}

	return &subscriber{
		opts:   options,
		client: m.client,
		topic:  topic,
	}, nil
}

func FindHeader(pd []byte) (n int) {
	if pd[0] != '{' {
		return 0
	}
	count := 0
	for {
		if n >= len(pd) {
			return 0
		}
		if pd[n] == '{' {
			count++
		} else if pd[n] == '}' {
			count--
			if count == 0 {
				return n + 1
			}
		}
		n++
	}
}
func (m *mqttBroker) filterMessageMetadata(header map[string]string) error {
	if !m.opts.FromTo {
		return nil
	}
	if to, ok := header[broker.MeggageTo]; !ok {
		if from, ok1 := header[broker.MessageFrom]; !ok1 {
			m.logFilter.Error("Massage-From&Massage-To", "massage without metadata MessageFrom&MessageTo")
			return errors.New("massage without metadata MessageFrom&MessageTo")
		} else if from == "" {
			m.logFilter.Error(to, "massage with error metadata MessageFrom")
			return errors.WithMessage(ErrMessageFrom, "massage with error metadata MessageFrom")
		} else {
			m.logFilter.Error(from, "massage without metadata MessageTo")
			return errors.New("massage without metadata MessageTo")
		}
	} else {
		if _, ok1 := header[broker.MessageFrom]; !ok1 {
			m.logFilter.Error(to, "massage without metadata MessageFrom")
			return errors.New("massage without metadata MessageFrom")
		} else if to != m.opts.ClientId {
			//		m.logFilter.Error(to, "massage with error metadata MessageTo")
			return errors.WithMessage(ErrMessageTo, "massage with error metadata MessageTo")
		} else if to == "" {
			m.logFilter.Error(to, "massage with error metadata MessageTo")
			return errors.WithMessage(ErrMessageTo, "massage with error metadata MessageTo")
		}
	}
	return nil
}

func (m *mqttBroker) SubscribeReq(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	if !m.client.IsConnected() {
		return nil, errors.New("not connected")
	}

	options := broker.NewSubscribeOptions(opts...)
	var qos byte = 2
	t := m.client.Subscribe(topic+"/req", qos, func(c MQTT.Client, mq MQTT.Message) {
		pd := mq.Payload()
		msg := broker.Message{}
		if err := broker.Unmarshal(m.opts.Codec, pd, &msg); err != nil {
			m.logFilter.Error("SubscribeReq Unmarshal", "SubscribeReq Unmarshal failed:"+topic)
			return
		}
		id, ok := msg.Headers.Headers[broker.Identifier]
		if !ok && id == "" {
			m.logFilter.Error(broker.Identifier+"Resp", "SubscribeReq massage with none metadata Identifier")
			return
		}
		if err := m.filterMessageMetadata(msg.Headers.Headers); err != nil {
			if errors.Cause(err) != ErrMessageTo {
				m.PublishCommErrorResp(topic, err.Error(), WithPublishQos(2))
				return
			}
			return
		}
		p := &publication{
			topic: mq.Topic(),
			msg:   &msg,
			data:  binder(id),
			err:   nil,
		}
		if err := msg.Data.UnmarshalTo(p.data.(protoreflect.ProtoMessage)); err != nil {
			log.Error(err)
			m.PublishCommErrorResp(topic,
				err.Error(),
				WithPublishQos(2),
				broker.PublishContextWithValue(broker.MessageId, msg.Headers.Headers[broker.MessageId]))
			return
		}
		resp, err := handler(m.opts.Context, p)
		if err != nil {
			log.Error(err)
			m.PublishCommErrorResp(topic,
				err.Error(),
				WithPublishQos(2),
				broker.PublishContextWithValue(broker.MessageId, msg.Headers.Headers[broker.MessageId]))
			return
		}
		m.PublishResp(topic,
			resp,
			WithPublishQos(2),
			broker.PublishContextWithValue(broker.MessageId, msg.Headers.Headers[broker.MessageId]),
			broker.PublishContextWithValue(broker.Identifier, msg.Headers.Headers[broker.Identifier]))
	})
	if rs, err := checkClientToken(t); !rs {
		return nil, err
	}
	return &subscriber{
		opts:   options,
		client: m.client,
		topic:  topic,
	}, nil
}

func (m *mqttBroker) SubscribeResp(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	if !m.client.IsConnected() {
		return nil, errors.New("not connected")
	}

	options := broker.NewSubscribeOptions(opts...)
	var qos byte = 2
	if value, ok := options.Context.Value(qosSubscribeKey{}).(byte); ok {
		qos = value
	}

	t := m.client.Subscribe(topic+"/resp", qos, func(c MQTT.Client, mq MQTT.Message) {
		pd := mq.Payload()
		msg := broker.Message{}
		if err := broker.Unmarshal(m.opts.Codec, pd, &msg); err != nil {
			log.Error(err)
			return
		}
		id, ok := msg.Headers.Headers[broker.Identifier]
		if !ok && id == "" {
			m.logFilter.Error(broker.Identifier+"Resp", "SubscribeResp massage with none metadata Identifier")
			return
		}
		p := &publication{
			topic: mq.Topic(),
			msg:   &msg,
			data:  binder(id),
			err:   nil,
		}
		if err := msg.Data.UnmarshalTo(p.data.(protoreflect.ProtoMessage)); err != nil {
			p.err = err
			log.Error(err)
			return
		}
		if _, err := handler(m.opts.Context, p); err != nil {
			p.err = err
			log.Error(err)
		}
	})

	if rs, err := checkClientToken(t); !rs {
		return nil, err
	}

	return &subscriber{
		opts:   options,
		client: m.client,
		topic:  topic,
	}, nil
}

func (m *mqttBroker) SubscribeUpload(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	if !m.client.IsConnected() {
		return nil, errors.New("not connected")
	}

	options := broker.NewSubscribeOptions(opts...)
	var qos byte = 2
	if value, ok := options.Context.Value(qosSubscribeKey{}).(byte); ok {
		qos = value
	}

	t := m.client.Subscribe(topic+"/upload", qos, func(c MQTT.Client, mq MQTT.Message) {
		pd := mq.Payload()
		msg := broker.Message{}
		id, ok := msg.Headers.Headers[broker.Identifier]
		if !ok && id == "" {
			m.logFilter.Error(broker.Identifier, "massage with none metadata Identifier")
			return
		}
		p := &publication{
			topic: mq.Topic(),
			msg:   &msg,
			data:  binder(id),
			err:   nil,
		}
		if err := broker.Unmarshal(m.opts.Codec, pd, &msg); err != nil {
			p.err = err
			log.Error(err)
			return
		}
		if err := UnmarshalAny(msg.Data, p.data); err != nil {
			p.err = err
			log.Error(err)
			return
		}
		if _, err := handler(m.opts.Context, p); err != nil {
			p.err = err
			log.Error(err)
		}
	})

	if rs, err := checkClientToken(t); !rs {
		return nil, err
	}

	return &subscriber{
		opts:   options,
		client: m.client,
		topic:  topic,
	}, nil
}

func (m *mqttBroker) onConnect(_ MQTT.Client) {
	log.Debug("on connect")
}

func (m *mqttBroker) onConnectionLost(client MQTT.Client, _ error) {
	log.Errorf("%s on connect lost, try to reconnect", m.addrs[0])
	m.Options()
	m.loopConnect(client)
}

func (m *mqttBroker) loopConnect(client MQTT.Client) {
	for {
		token := client.Connect()
		if rs, err := checkClientToken(token); !rs {
			log.Errorf("connect error: %s", err.Error())
		} else {
			break
		}
		time.Sleep(1 * time.Second)
	}
}
