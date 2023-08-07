package main

import (
	"context"
	"github.com/luobote55/kratos-transport-rpc/server/mqtt"
	"log"
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

type Service struct {
	UnimplementedGreeterMqtServer
}

func (s *Service) SayHello(ctx context.Context, request *HelloRequest) (*HelloReply, error) {
	resp := HelloReply{
		Message:  "asdfasdfasdf",
		TimeFrom: request.TimeFrom,
		TimeRecv: time.Now().Unix(),
		TimeTo:   0,
	}
	return &resp, nil
}

func (s *Service) mustEmbedUnimplementedGreeterMqtServer() {
	//TODO implement me
	panic("implement me")
}

func main() {
	s := mqtt.NewServer(
		mqtt.WithAddress([]string{EmqxCnBroker}),
		mqtt.WithCodec("json"),
	)
	go func() {
		s.Start(context.Background())
	}()

	time.Sleep(time.Second * time.Duration(2))
	service := &Service{}
	RegisterGreeterMqtServer(s, "/hello", service)

	c := NewGreeterMqtClient(mqtt.NewClient())
	resp, err := c.SayHello(context.Background(), "/hello", &HelloRequest{})
	if err != nil {
		log.Fatal(err)
	}
	log.Print(resp)
}
