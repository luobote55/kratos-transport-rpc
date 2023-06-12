package mqtt

import (
	"context"
	"fmt"
	"testing"
	"time"
)

const (
	EmqxBroker        = "tcp://broker.emqx.io:1883"
	EmqxCnBroker      = "tcp://broker-cn.emqx.io:1883"
	EclipseBroker     = "tcp://mqtt.eclipseprojects.io:1883"
	MosquittoBroker   = "tcp://test.mosquitto.org:1883"
	HiveMQBroker      = "tcp://broker.hivemq.com:1883"
	LocalEmxqBroker   = "tcp://127.0.0.1:1883"
	LocalRabbitBroker = "tcp://user:bitnami@127.0.0.1:1883"

	TestTopic = "topic/bobo/helloword"
)

type HelloRequest struct {
	Name string
}

type HelloReply struct {
	Name string
}

const (
	Greeter_SayHello_Mqt_FullMethodName = "SayHelloUri"
	Greeter_SayHello_Mqt_Topic          = "HelloTopic"
)

func TestServer(t *testing.T) {
	s := NewServer(
		WithAddress([]string{EmqxCnBroker}),
		WithCodec("json"),
	)

	r := s.Route("")
	REQ(r, Greeter_SayHello_Mqt_FullMethodName, HelloRequest{}, func(ctx context.Context, in interface{}) (interface{}, error) {
		req := in.(*HelloRequest)
		fmt.Println(req.Name)
		return &HelloReply{
			Name: req.Name + req.Name + req.Name,
		}, nil
	})
	r.Route(Greeter_SayHello_Mqt_Topic)

	go func() {
		s.Start(context.Background())
	}()
	time.Sleep(time.Second * time.Duration(2))
	out := new(HelloReply)
	err := NewClient().Invoke(context.Background(), Greeter_SayHello_Mqt_FullMethodName, Greeter_SayHello_Mqt_Topic, &HelloRequest{
		Name: "HelloName",
	}, out)
	if err != nil {
		t.Error(err)
	}
	t.Log(out)
}
