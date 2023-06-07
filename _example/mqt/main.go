package main

import (
	"context"
	v1 "github.com/luobote55/kratos-transport-rpc/api/v1"
	"github.com/luobote55/kratos-transport-rpc/server/mqtt"
	"log"
)

type Service struct {
	v1.UnimplementedGreeterMqtServer
}

func (s *Service) SayHello(ctx context.Context, request *v1.HelloRequest) (*v1.HelloReply, error) {
	//TODO implement me
	panic("implement me")
}

func (s *Service) mustEmbedUnimplementedGreeterMqtServer() {
	//TODO implement me
	panic("implement me")
}

func main() {
	s := mqtt.NewServer()
	go func() {
		s.Start(context.Background())
	}()

	service := &Service{}
	v1.RegisterGreeterMqtServer(s, "/hello", service)

	c := v1.NewGreeterMqtClient(mqtt.NewClient())
	resp, err := c.SayHello(context.Background(), "/hello", &v1.HelloRequest{})
	if err != nil {
		log.Fatal(err)
	}
	log.Print(resp)
}
