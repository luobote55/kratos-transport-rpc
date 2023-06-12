package mqtt

import "github.com/luobote55/kratos-transport-rpc/broker"

type publication struct {
	topic string
	msg   *broker.Message
	err   error
	raw   []byte
}

func (m *publication) Raw() []byte {
	return m.raw
}

func (m *publication) Data() broker.Any {
	//TODO implement me
	panic("implement me")
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
