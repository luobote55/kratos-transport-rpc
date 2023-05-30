package mqtt

import (
	"context"
	"github.com/go-kratos/kratos/v2/middleware"
	"time"
)

var _ Context = (*wrapper)(nil)

// Context is an broker Context.
type Context interface {
	context.Context
	Topic() string
	Bind(interface{}) error
	Middleware(middleware.Handler) middleware.Handler
	Result(int, interface{}) error
	SendResult()
	Req() interface{}
	SetOperation(ctx context.Context, bank string)
}

type wrapper struct {
}

func (w wrapper) Deadline() (deadline time.Time, ok bool) {
	//TODO implement me
	panic("implement me")
}

func (w wrapper) Done() <-chan struct{} {
	//TODO implement me
	panic("implement me")
}

func (w wrapper) Err() error {
	//TODO implement me
	panic("implement me")
}

func (w wrapper) Value(key any) any {
	//TODO implement me
	panic("implement me")
}

func (w wrapper) Topic() string {
	//TODO implement me
	panic("implement me")
}

func (w wrapper) Bind(i interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (w wrapper) Middleware(handler middleware.Handler) middleware.Handler {
	//TODO implement me
	panic("implement me")
}

func (w wrapper) Result(i int, i2 interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (w wrapper) SendResult() {
	//TODO implement me
	panic("implement me")
}

func (w wrapper) Req() interface{} {
	//TODO implement me
	panic("implement me")
}

func (w wrapper) SetOperation(ctx context.Context, bank string) {
	//TODO implement me
	panic("implement me")
}
