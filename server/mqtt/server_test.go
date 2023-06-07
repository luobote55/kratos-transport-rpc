package mqtt

import (
	"context"
	"fmt"
	"testing"
)

type HelloRequest struct {
	Name string
}

type HelloReply struct {
	Name string
}

const (
	Greeter_SayHello_Mqt_FullMethodName = "SayHello"
	Greeter_SayHello_Mqt_Topic          = "Hello"
)

func TestServer(t *testing.T) {
	s := NewServer()
	r := s.Route("")
	r.REQ(Greeter_SayHello_Mqt_FullMethodName, func(ctx context.Context, in interface{}) (interface{}, error) {
		req := in.(*HelloRequest)
		fmt.Println(req.Name)
		return &HelloReply{
			Name: req.Name,
		}, nil
	})
	r.Route(Greeter_SayHello_Mqt_Topic)

	go func() {
		s.Start(context.Background())
	}()

	out := new(HelloReply)
	err := NewClient().Invoke(context.Background(), Greeter_SayHello_Mqt_FullMethodName, Greeter_SayHello_Mqt_Topic, &HelloRequest{
		Name: "HelloName",
	}, out)
	if err != nil {
		t.Error(err)
	}
	t.Log(out)
}
