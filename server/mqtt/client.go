package mqtt

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/luobote55/kratos-transport-rpc/broker"
	"github.com/pkg/errors"
	"io"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func nop()              {}
func alwaysFalse() bool { return false }

type CallOption interface {
}

type Client struct {
	Timeout time.Duration
}

func (c *Client) RoundTrip(request *Request) (*Response, error) {
	//TODO implement me
	panic("implement me")
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

func send(ireq *Request, rt RoundTripper, deadline time.Time) (resp *Response, didTimeout func() bool, err error) {
	req := ireq // req is either the original request, or a modified fork

	if rt == nil {
		req.closeBody()
		return nil, alwaysFalse, errors.New("http: no Client.Transport or DefaultTransport")
	}

	if req.Method == "" {
		req.closeBody()
		return nil, alwaysFalse, errors.New("http: nil Request.URL")
	}

	if req.RequestURI != "" {
		req.closeBody()
		return nil, alwaysFalse, errors.New("http: Request.RequestURI can't be set in client requests")
	}

	// forkReq forks req into a shallow clone of ireq the first
	// time it's called.
	forkReq := func() {
		if ireq == req {
			req = new(Request)
			*req = *ireq // shallow clone
		}
	}

	// Most the callers of send (Get, Post, et al) don't need
	// Headers, leaving it uninitialized. We guarantee to the
	// Transport that this has been initialized, though.
	if req.Header == nil {
		forkReq()
		req.Header = make(Header)
	}

	if !deadline.IsZero() {
		forkReq()
	}
	stopTimer, didTimeout := setRequestCancel(req, rt, deadline)

	resp, err = rt.RoundTrip(req)
	if err != nil {
		stopTimer()
		if resp != nil {
			log.Printf("RoundTripper returned a response & error; ignoring response")
		}
		if tlsErr, ok := err.(tls.RecordHeaderError); ok {
			// If we get a bad TLS record header, check to see if the
			// response looks like HTTP and give a more helpful error.
			// See golang.org/issue/11111.
			if string(tlsErr.RecordHeader[:]) == "HTTP/" {
				err = errors.New("http: server gave HTTP response to HTTPS client")
			}
		}
		return nil, didTimeout, err
	}
	if resp == nil {
		return nil, didTimeout, fmt.Errorf("http: RoundTripper implementation (%T) returned a nil *Response with a nil error", rt)
	}
	if resp.Body == nil {
		// The documentation on the Body field says “The http Client and Transport
		// guarantee that Body is always non-nil, even on responses without a body
		// or responses with a zero-length body.” Unfortunately, we didn't document
		// that same constraint for arbitrary RoundTripper implementations, and
		// RoundTripper implementations in the wild (mostly in tests) assume that
		// they can use a nil Body to mean an empty one (similar to Request.Body).
		// (See https://golang.org/issue/38095.)
		//
		// If the ContentLength allows the Body to be empty, fill in an empty one
		// here to ensure that it is non-nil.
		if resp.ContentLength > 0 && req.Method != "HEAD" {
			return nil, didTimeout, fmt.Errorf("http: RoundTripper implementation (%T) returned a *Response with content length %d but a nil Body", rt, resp.ContentLength)
		}
		resp.Body = io.NopCloser(strings.NewReader(""))
	}
	if !deadline.IsZero() {
		resp.Body = &cancelTimerBody{
			stop:          stopTimer,
			rc:            resp.Body,
			reqDidTimeout: didTimeout,
		}
	}
	return resp, nil, nil
}

func timeBeforeContextDeadline(t time.Time, ctx context.Context) bool {
	d, ok := ctx.Deadline()
	if !ok {
		return true
	}
	return t.Before(d)
}

func setRequestCancel(req *Request, rt RoundTripper, deadline time.Time) (stopTimer func(), didTimeout func() bool) {
	if deadline.IsZero() {
		return nop, alwaysFalse
	}
	oldCtx := req.Context()
	if req.Cancel == nil {

		var cancelCtx func()
		req.ctx, cancelCtx = context.WithDeadline(oldCtx, deadline)
		return cancelCtx, func() bool { return time.Now().After(deadline) }
	}
	initialReqCancel := req.Cancel // the user's original Request.Cancel, if any

	var cancelCtx func()
	if timeBeforeContextDeadline(deadline, oldCtx) {
		req.ctx, cancelCtx = context.WithDeadline(oldCtx, deadline)
	}

	cancel := make(chan struct{})
	req.Cancel = cancel

	doCancel := func() {
		// The second way in the func comment above:
		close(cancel)
		// The first way, used only for RoundTripper
		// implementations written before Go 1.5 or Go 1.6.
		type canceler interface{ CancelRequest(*Request) }
		if v, ok := rt.(canceler); ok {
			v.CancelRequest(req)
		}
	}

	stopTimerCh := make(chan struct{})
	var once sync.Once
	stopTimer = func() {
		once.Do(func() {
			close(stopTimerCh)
			if cancelCtx != nil {
				cancelCtx()
			}
		})
	}

	timer := time.NewTimer(time.Until(deadline))
	var timedOut atomic.Bool

	go func() {
		select {
		case <-initialReqCancel:
			doCancel()
			timer.Stop()
		case <-timer.C:
			timedOut.Store(true)
			doCancel()
		case <-stopTimerCh:
			timer.Stop()
		}
	}()

	return stopTimer, timedOut.Load
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

func (c *Client) Invoke(ctx context.Context, topic string, mothed string, in interface{}, out interface{}, opts ...CallOption) error {
	req, err := NewRequestWithContext(ctx, topic, mothed, &RequestRead{
		in:    in,
		Codec: nil,
	})
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	out = resp
	return err
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (c *Client) Do(req *Request) (retres *Response, err error) {
	var (
		deadline      = c.deadline()
		reqs          []*Request
		resp          *Response
		reqBodyClosed = false // have we closed the current req.Body?

		// Redirect behavior:
		redirectMethod string
		includeBody    bool
	)

	defer func() {
		if !reqBodyClosed {
			req.closeBody()
		}
		if err != nil {
		}
	}()

	for {
		// For all but the first request, create the next
		// request hop and replace req.
		if len(reqs) > 0 {
			loc := resp.Header.Get("Location")
			if loc == "" {
				// While most 3xx responses include a Location, it is not
				// required and 3xx responses without a Location have been
				// observed in the wild. See issues #17773 and #49281.
				return resp, nil
			}
			ireq := reqs[0]
			req = &Request{
				Method:   redirectMethod,
				Response: resp,
				Header:   make(Header),
				Cancel:   ireq.Cancel,
				ctx:      ireq.ctx,
			}
			if includeBody && ireq.GetBody != nil {
				req.Body, err = ireq.GetBody()
				if err != nil {
					resp.closeBody()
					return nil, err
				}
				req.ContentLength = ireq.ContentLength
			}

			const maxBodySlurpSize = 2 << 10
			if resp.ContentLength == -1 || resp.ContentLength <= maxBodySlurpSize {
				io.CopyN(io.Discard, resp.Body, maxBodySlurpSize)
			}
			resp.Body.Close()
			if err != nil {
				return resp, err
			}
		}

		reqs = append(reqs, req)
		var err error
		var didTimeout func() bool
		if resp, didTimeout, err = send(req, c, deadline); err != nil {
			// c.send() always closes req.Body
			reqBodyClosed = true
			if !deadline.IsZero() && didTimeout() {
				return nil, ErrTimeoutExceeded
			}
			return nil, err
		}

		reqBodyClosed = true
		req.closeBody()
	}
}
