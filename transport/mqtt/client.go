package mqtt

import (
	"context"
)

type CallOption interface {
}

type Client struct {
	Server
}

func (c *Client) Invoke(ctx context.Context, topic string, mothed string, in interface{}, out interface{}, opts ...CallOption) error {
	return nil
}
