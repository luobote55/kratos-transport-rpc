package broker

import (
	"context"
)

type Any interface{}

type Handler func(context.Context, Event) (RespEvent, error)

type Binder func() Any

//type Headers map[string]string
//
//func (h Headers) Get(key string) string {
//	return h[key]
//}
//
//func (h Headers) Set(key, value string) {
//	h[key] = value
//}

type Message struct {
	Headers Headers `protobuf:"varint,1,opt,name=headers,proto3" json:"headers,omitempty"`
	Body    Any
}

func (m Message) Topic() string {
	panic("implement me")
}

func (m Message) GetBody() Any {
	return m.Body
}

func (m Message) GetHeaders() Headers {
	return m.Headers
}

func (m Message) GetHeader(key string) string {
	return m.Headers.Headers[key]
}

type RespEvent interface {
	GetBody() Any
}

type Event interface {
	Topic() string
	Message() *Message
	Ack() error
	Error() error
}

type Subscriber interface {
	Options() SubscribeOptions
	Topic() string
	Unsubscribe() error
}

type Broker interface {
	Name() string
	Options() Options
	Address() string

	Init(...Option) error

	Connect() error
	ConnectRetry() error
	Disconnect() error

	Publish(topic string, msg Any, opts ...PublishOption) error
	PublishReq(topic string, msg Any, opts ...PublishOption) error
	PublishUpload(topic string, msg Any, opts ...PublishOption) error

	Subscribe(topic string, handler Handler, binder Binder, opts ...SubscribeOption) (Subscriber, error)
	SubscribeReq(topic string, handler Handler, binder Binder, opts ...SubscribeOption) (Subscriber, error)
	SubscribeResp(topic string, handler Handler, binder Binder, opts ...SubscribeOption) (Subscriber, error)
	SubscribeUpload(topic string, handler Handler, binder Binder, opts ...SubscribeOption) (Subscriber, error)
}
