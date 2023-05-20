package mqtt

import (
	"github.com/go-kratos/kratos/v2/transport"
	"kratos-transport-rpc/broker"
	transport2 "kratos-transport-rpc/transport"
)

const (
	KindMQTT transport.Kind = "mqtt"
)

// Transport is an HTTP transport.
type Transport struct {
	endpoint     string
	operation    string
	reqHeader    headerCarrier
	replyHeader  headerCarrier
	pathTemplate string
}

// Kind returns the transport kind.
func (tr *Transport) Kind() transport.Kind {
	return KindMQTT
}

// Endpoint returns the transport endpoint.
func (tr *Transport) Endpoint() string {
	return tr.endpoint
}

// Operation returns the transport operation.
func (tr *Transport) Operation() string {
	return tr.operation
}

// RequestHeader returns the request header.
func (tr *Transport) RequestHeader() transport.Header {
	return tr.reqHeader
}

// ReplyHeader returns the reply header.
func (tr *Transport) ReplyHeader() transport.Header {
	return tr.replyHeader
}

// PathTemplate returns the http path template.
func (tr *Transport) PathTemplate() string {
	return tr.pathTemplate
}

// SetOperation sets the transport operation.
func SetOperation(ctx transport2.Context, op string) {
	if tr, ok := transport.FromServerContext(ctx); ok {
		if tr, ok := tr.(*Transport); ok {
			tr.operation = op
		}
	}
}

type headerCarrier broker.Headers

func (hc headerCarrier) Add(key string, value string) {
	//TODO implement me
	panic("implement me")
}

func (hc headerCarrier) Values(key string) []string {
	//TODO implement me
	panic("implement me")
}

// Get returns the value associated with the passed key.
func (hc headerCarrier) Get(key string) string {
	return broker.Headers(hc).Headers[key]
}

// Set stores the key-value pair.
func (hc headerCarrier) Set(key string, value string) {
	broker.Headers(hc).Headers[key] = value
}

// Keys lists the keys stored in this carrier.
func (hc headerCarrier) Keys() []string {
	keys := make([]string, 0, len(hc.Headers))
	for k := range broker.Headers(hc).Headers {
		keys = append(keys, k)
	}
	return keys
}
