package mqtt

import "github.com/luobote55/kratos-transport-rpc/broker"

type publication struct {
	topic string
	msg   *broker.Message
	data  broker.Any
	err   error
}

func (m *publication) Data() broker.Any {
	return m.data
}

func (m *publication) Ack() error {
	return nil
}

func (m *publication) Error() error {
	return m.err
}

func (m *publication) Topic() string {
	return m.topic
}

func (m *publication) Message() *broker.Message {
	return m.msg
}
