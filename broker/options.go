package broker

import (
	"context"
	"crypto/tls"
	"github.com/go-kratos/kratos/v2/log"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-kratos/kratos/v2/encoding"

	"github.com/tx7do/kratos-transport/tracing"
)

var (
	DefaultCodec encoding.Codec = nil
)

///////////////////////////////////////////////////////////////////////////////

type Options struct {
	Addrs    []string
	ClientId string // 检查是不是发给自己的
	FromTo   bool

	Codec encoding.Codec

	ErrorHandler Handler

	Secure    bool
	TLSConfig *tls.Config

	Context context.Context

	Tracings []tracing.Option

	OnConnect  func()
	Disconncet func()

	Log *log.Helper

	errorEnc EncodeErrorFunc
}

// EncodeErrorFunc is encode error func.
type EncodeErrorFunc func(error) RespEvent

type Option func(*Options)

func (o *Options) Apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

func (o *Options) Logger() *log.Helper {
	return o.Log
}

func NewOptions() Options {
	opt := Options{
		Addrs:        []string{},
		Codec:        DefaultCodec,
		ErrorHandler: nil,
		Secure:       false,
		TLSConfig:    nil,
		Context:      context.Background(),
		Tracings:     []tracing.Option{},
		OnConnect:    nil,
		Disconncet:   nil,
	}

	return opt
}

func NewOptionsAndApply(opts ...Option) Options {
	opt := NewOptions()
	opt.Apply(opts...)
	return opt
}

func WithOptionContext(ctx context.Context) Option {
	return func(o *Options) {
		if o.Context == nil {
			o.Context = ctx
		}
	}
}

func OptionContextWithValue(k, v interface{}) Option {
	return func(o *Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

// WithAddress set broker address
func WithAddress(addressList ...string) Option {
	return func(o *Options) {
		o.Addrs = addressList
	}
}

// WithClientId set broker clientId
func WithClientId(clientId string) Option {
	return func(o *Options) {
		o.ClientId = clientId
	}
}

// WithCodec set codec, support: json, proto.
func WithCodec(name string) Option {
	return func(o *Options) {
		o.Codec = encoding.GetCodec(name)
	}
}

func WithErrorHandler(handler Handler) Option {
	return func(o *Options) {
		o.ErrorHandler = handler
	}
}

func WithEnableSecure(enable bool) Option {
	return func(o *Options) {
		o.Secure = enable
	}
}

func WithTLSConfig(config *tls.Config) Option {
	return func(o *Options) {
		o.TLSConfig = config
		if o.TLSConfig != nil {
			o.Secure = true
		}
	}
}

func WithTracerProvider(provider trace.TracerProvider, tracerName string) Option {
	return func(opt *Options) {
		opt.Tracings = append(opt.Tracings, tracing.WithTracerProvider(provider))
	}
}

func WithPropagator(propagators propagation.TextMapPropagator) Option {
	return func(opt *Options) {
		opt.Tracings = append(opt.Tracings, tracing.WithPropagator(propagators))
	}
}

func WithGlobalTracerProvider() Option {
	return func(opt *Options) {
		opt.Tracings = append(opt.Tracings, tracing.WithGlobalTracerProvider())
	}
}

func WithGlobalPropagator() Option {
	return func(opt *Options) {
		opt.Tracings = append(opt.Tracings, tracing.WithGlobalPropagator())
	}
}

func WithOnConnect(f func()) Option {
	return func(opt *Options) {
		opt.OnConnect = f
	}
}

func WithDisconnect(f func()) Option {
	return func(opt *Options) {
		opt.Disconncet = f
	}
}

func WithLogger(logger log.Logger) Option {
	return func(o *Options) {
		o.Log = log.NewHelper(logger)
	}
}

// /////////////////////////////////////////////////////////////////////////////
const (
	MessageAct    = "Message-Act"
	MessageId     = "Message-Id"
	MessageFrom   = "Massage-From"
	MeggageTo     = "Massage-To"
	Identifier    = "Identifier"
	Failed        = "Failed"
	MaggageToken  = "Massage-Token"
	ContextCodec  = "Context-Codec"
	TimeStampFrom = "TimeStampFrom"
	TimeStampTo   = "TimeStampTo"

	Host             = "Host"
	Connection       = "Connection"
	ContentType      = "Content-Type"
	TransferEncoding = "Transfer-Encoding"
	AcceptEncoding   = "Accept-Encoding"
	Authorization    = "Authorization"
	ContentLength    = "Content-Length"
)

type PublishOptions struct {
	Context context.Context
}

type PublishOption func(*PublishOptions)

func (o *PublishOptions) Apply(opts ...PublishOption) {
	for _, opt := range opts {
		opt(o)
	}
}

func NewPublishOptions(opts ...PublishOption) PublishOptions {
	opt := PublishOptions{
		Context: context.Background(),
	}

	opt.Apply(opts...)

	return opt
}

func PublishContextWithValue(k, v interface{}) PublishOption {
	return func(o *PublishOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

func WithPublishContext(ctx context.Context) PublishOption {
	return func(o *PublishOptions) {
		o.Context = ctx
	}
}

///////////////////////////////////////////////////////////////////////////////

type SubscribeOptions struct {
	AutoAck bool
	Act     int // 0 :  ,1 : req, 2 : resp, 3 : upload
	Queue   string
	Context context.Context
}

type SubscribeOption func(*SubscribeOptions)

func (o *SubscribeOptions) Apply(opts ...SubscribeOption) {
	for _, opt := range opts {
		opt(o)
	}
}

func NewSubscribeOptions(opts ...SubscribeOption) SubscribeOptions {
	opt := SubscribeOptions{
		AutoAck: true,
		Queue:   "",
		Context: context.Background(),
	}

	opt.Apply(opts...)

	return opt
}

func SubscribeContextWithValue(k, v interface{}) SubscribeOption {
	return func(o *SubscribeOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

func DisableAutoAck() SubscribeOption {
	return func(o *SubscribeOptions) {
		o.AutoAck = false
	}
}

func WithQueueName(name string) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.Queue = name
	}
}

func WithSubscribeContext(ctx context.Context) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.Context = ctx
	}
}

func WithSubscribAct(Act int) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.Act = Act
	}
}
