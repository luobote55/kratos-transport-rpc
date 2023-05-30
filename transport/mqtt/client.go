package mqtt

import (
	"context"
)

type Client struct {
	Server
}

func (c Client) Invoke(ctx context.Context, s string, pattern string, in interface{}, p interface{}) error {
	return nil
}
