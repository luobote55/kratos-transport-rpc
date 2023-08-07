package mqtt

import (
	"errors"
	"github.com/luobote55/kratos-transport-rpc/broker"
	"runtime"
	"strings"
	"sync"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-kratos/kratos/v2/log"
)

type mqttBroker struct {
	addrs     []string
	opts      broker.Options
	client    MQTT.Client
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

func (m *mqttBroker) PublishRaw(topic string, buf []byte, opts ...broker.PublishOption) error {
	return m.publish(topic, buf, opts...)
}

func (m *mqttBroker) Publish(topic string, msg broker.Any, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(m.opts.Codec, msg)
	if err != nil {
		return err
	}

	return m.publish(topic, buf, opts...)
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
		go func() {
			defer func() {
				if rerr := recover(); rerr != nil {
					buf := make([]byte, 64<<10) //nolint:gomnd
					n := runtime.Stack(buf, false)
					log.Errorf("panic %v: \n%s\n", rerr, buf[:n])
				}
			}()
			var msg broker.Message
			p := &publication{topic: mq.Topic(), msg: &msg, raw: mq.Payload()}
			if err := handler(m.opts.Context, p); err != nil {
				log.Error(err)
			}
		}()
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

func (m *mqttBroker) onConnect(_ MQTT.Client) {
	log.Debug("on connect")
}

func (m *mqttBroker) onConnectionLost(client MQTT.Client, _ error) {
	log.Errorf("%s on connect lost, try to reconnect", m.addrs[0])
	m.Options()
//	m.loopConnect(client)
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
