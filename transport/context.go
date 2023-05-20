package transport

import (
	"context"
	"github.com/go-kratos/kratos/v2/middleware"
)

// Context is an broker Context.
type Context interface {
	context.Context
	Topic() string
	Bind(interface{}) error
	Middleware(middleware.Handler) middleware.Handler
	Result(int, interface{}) error
	SendResult()
	Req() interface{}
	SetOperation(ctx Context, bank string)
}
