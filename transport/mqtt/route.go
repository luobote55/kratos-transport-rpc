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

func (r *Router) REQ(id string, handler func(context.Context, interface{}) (interface{}, error)) {
	r.route[id] = bind{
		Handler: handler,
		//		Binder:  binder,
	}
}

func (r *Router) Route() {
	//	r.srv.
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
