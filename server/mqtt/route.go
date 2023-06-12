package mqtt

import (
	"context"
	"github.com/luobote55/kratos-transport-rpc/broker"
)

type bind struct {
	Handler
	broker.Binder
}

type Router struct {
	route map[string]bind
	srv   *Server
}

func REQ[T any](r *Router, method string, resp T, handler func(context.Context, interface{}) (interface{}, error)) {
	r.REQ(method, func(ctx context.Context, in interface{}) (interface{}, error) {
		resp, err := handler(ctx, in)
		return resp, err
	}, func() broker.Any {
		return new(T)
	})
}

func (r *Router) REQ(method string, handler func(context.Context, interface{}) (resp interface{}, err error), binder broker.Binder) {
	r.route[method] = bind{
		Handler: handler,
		Binder:  binder,
	}
}

func (r *Router) Route(topic string) {
	r.srv.RegRoute(topic, r)
}

func (r *Router) RouteUpload() {

}

func newRouter(prefix string, srv *Server) *Router {
	r := &Router{
		route: map[string]bind{},
		srv:   srv,
	}
	return r
}
