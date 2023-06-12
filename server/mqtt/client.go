package mqtt

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/google/uuid"
	"github.com/luobote55/kratos-transport-rpc/broker"
	"github.com/luobote55/kratos-transport-rpc/server/mqtt/bufio"
	"github.com/luobote55/kratos-transport-rpc/server/mqtt/bytes"
	"github.com/pkg/errors"
	"io"
	"strings"
	"time"
)

func nop()              {}
func alwaysFalse() bool { return false }

type CallOption interface {
}

type Client struct {
	Timeout time.Duration
}

type RequestRead struct {
	in    interface{}
	Codec encoding.Codec
}

func (r *RequestRead) Read(p []byte) (int, error) {
	buf, err := broker.Marshal(r.Codec, r.in)
	if err != nil {
		return 0, err
	}
	copy(buf, p)
	return len(buf), nil
}

func Req(ctx context.Context, topic string, mothed string, in interface{}, out interface{}, opts ...CallOption) error {
	return NewClient().Invoke(ctx, topic, mothed, in, out, opts...)
}

func NewClient() *Client {
	return &Client{}
}

type cancelTimerBody struct {
	stop          func() // stops the time.Timer waiting to cancel the request
	rc            io.ReadCloser
	reqDidTimeout func() bool
}

var ErrTimeout error = errors.New("timeout awaiting response headers")
var ErrTimeoutExceeded error = errors.New("Timeout exceeded while awaiting header")

func (b *cancelTimerBody) Read(p []byte) (n int, err error) {
	n, err = b.rc.Read(p)
	if err == nil {
		return n, nil
	}
	if err == io.EOF {
		return n, err
	}
	if b.reqDidTimeout() {
		return n, ErrTimeout
	}
	return n, err
}

func (b *cancelTimerBody) Close() error {
	err := b.rc.Close()
	b.stop()
	return err
}

func (c *Client) deadline() time.Time {
	if c.Timeout > 0 {
		return time.Now().Add(c.Timeout)
	}
	return time.Time{}
}

type RoundTripper interface {
	RoundTrip(*Request) (*Response, error)
}

func (c *Client) Invoke(ctx context.Context, uri string, topic string, in interface{}, out interface{}, opts ...CallOption) error {
	wr := bytes.NewWriterSize(64 * 1024)
	writeFirstLine(wr, []byte(uri))

	writeHeader(wr, broker.ContentType, fmt.Sprintf("application/%s", srv.codec))
	//	writeHeader(wr, broker.AcceptEncoding, "gzip")
	msgId, _ := uuid.NewRandom()
	writeHeader(wr, broker.MessageId, msgId.String())
	buf, err := broker.Marshal(encoding.GetCodec(srv.codec), in)
	if err != nil {
		return err
	}
	writeHeaderInt(wr, broker.ContentLength, len(buf))
	wr.Write([]byte("\r\n"))
	wr.Write(buf)

	ctx, _ = context.WithTimeout(ctx, time.Second*200)

	return c.Send(ctx, wr, topic, func() broker.Any { return out })
}

func (c *Client) Send(ctx context.Context, wr *bytes.Writer, topic string, binder broker.Binder) (err error) {
	req := binder()
	ch := make(chan struct{}, 1)
	srv.server.RegisterSubscriber(context.Background(), topic+"/resp", func(ctx context.Context, msg broker.Event) error {
		request, err := ReadRequest(bufio.NewReader(strings.NewReader(string(msg.Raw()))))
		if err != nil {
			return err
		}
		var codec string = "json"
		contentType, ok := request.Header[broker.ContentType]
		if ok && contentType[0] == "appcation/proto" {
			codec = "proto"
		}
		if len(request.Body) == 0 {
			return errors.New("body没有body")
		}
		encoding.GetCodec(codec).Unmarshal(request.Body, req)

		ch <- struct{}{}
		return nil
	}, func() broker.Any { return nil })

	err = srv.server.PublishRaw(topic+"/req", wr.Buffer())
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
		return nil
	}
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
