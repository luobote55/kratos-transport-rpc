package mqtt

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/luobote55/kratos-transport-rpc/broker"
	"github.com/luobote55/kratos-transport-rpc/server/mqtt/bufio"
	"github.com/luobote55/kratos-transport-rpc/server/mqtt/bytes"
	"github.com/luobote55/kratos-transport-rpc/transport/mqtt"
	"github.com/pkg/errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

const SupportPackageIsVersion1 = true

var (
	ErrBodyNotAllowed  = errors.New("http: request method or response status code does not allow body")
	ErrHijacked        = errors.New("http: connection has been hijacked")
	ErrContentLength   = errors.New("http: wrote more than the declared Content-Length")
	ErrWriteAfterFlush = errors.New("unused")
	ErrAbortHandler    = errors.New("net/http: abort Handler")
)

type Handler func(context.Context, interface{}) (interface{}, error)

type ResponseWriter interface {
	Header() broker.Headers
	Write([]byte) (int, error)
	WriteHeader(statusCode int)
}

type Flusher interface {
	Flush()
}

type CloseNotifier interface {
	CloseNotify() <-chan bool
}

type contextKey struct {
	name string
}

var (
	ServerContextKey    = &contextKey{"mqtt-server"}
	LocalAddrContextKey = &contextKey{"local-addr"}
)

const maxInt64 = 1<<63 - 1

var aLongTimeAgo = time.Unix(1, 0)

var (
	bufioReaderPool   sync.Pool
	bufioWriter2kPool sync.Pool
	bufioWriter4kPool sync.Pool
)

var copyBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 32*1024)
		return &b
	},
}

func bufioWriterPool(size int) *sync.Pool {
	switch size {
	case 2 << 10:
		return &bufioWriter2kPool
	case 4 << 10:
		return &bufioWriter4kPool
	}
	return nil
}

func newBufioReader(r io.Reader) *bufio.Reader {
	if v := bufioReaderPool.Get(); v != nil {
		br := v.(*bufio.Reader)
		br.Reset(r)
		return br
	}
	// Note: if this reader size is ever changed, update
	// TestHandlerBodyClose's assumptions.
	return bufio.NewReader(r)
}

func putBufioReader(br *bufio.Reader) {
	br.Reset(nil)
	bufioReaderPool.Put(br)
}

func newBufioWriterSize(w io.Writer, size int) *bufio.Writer {
	pool := bufioWriterPool(size)
	if pool != nil {
		if v := pool.Get(); v != nil {
			bw := v.(*bufio.Writer)
			bw.Reset(w)
			return bw
		}
	}
	return bufio.NewWriterSize(w, size)
}

func putBufioWriter(bw *bufio.Writer) {
	bw.Reset(nil)
	if pool := bufioWriterPool(bw.Available()); pool != nil {
		pool.Put(bw)
	}
}

const TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

// appendTime is a non-allocating version of []byte(t.UTC().Format(TimeFormat))
func appendTime(b []byte, t time.Time) []byte {
	const days = "SunMonTueWedThuFriSat"
	const months = "JanFebMarAprMayJunJulAugSepOctNovDec"

	t = t.UTC()
	yy, mm, dd := t.Date()
	hh, mn, ss := t.Clock()
	day := days[3*t.Weekday():]
	mon := months[3*(mm-1):]

	return append(b,
		day[0], day[1], day[2], ',', ' ',
		byte('0'+dd/10), byte('0'+dd%10), ' ',
		mon[0], mon[1], mon[2], ' ',
		byte('0'+yy/1000), byte('0'+(yy/100)%10), byte('0'+(yy/10)%10), byte('0'+yy%10), ' ',
		byte('0'+hh/10), byte('0'+hh%10), ':',
		byte('0'+mn/10), byte('0'+mn%10), ':',
		byte('0'+ss/10), byte('0'+ss%10), ' ',
		'G', 'M', 'T')
}

// DefaultMaxHeaderBytes is the maximum permitted size of the headers
// in an HTTP request.
// This can be overridden by setting Server.MaxHeaderBytes.
const DefaultMaxHeaderBytes = 1 << 20 // 1 MB

func (srv *Server) maxHeaderBytes() int {
	if srv.MaxHeaderBytes > 0 {
		return srv.MaxHeaderBytes
	}
	return DefaultMaxHeaderBytes
}

func (srv *Server) initialReadLimitSize() int64 {
	return int64(srv.maxHeaderBytes()) + 4096 // bufio slop
}

func writeFirstLine(bw *bytes.Writer, method []byte) {
	bw.Write([]byte(fmt.Sprintf("REQ %s MQT/1.0\r\n", method)))
}

func writeStatusLine(bw *bytes.Writer, code int, scratch []byte) {
	bw.Write([]byte(fmt.Sprintf("MQT/1.0 %d %s \r\n", code, scratch)))
}

func writeHeader(wr *bytes.Writer, head, context string) {
	wr.Write([]byte(fmt.Sprintf("%s: %s\r\n", head, context)))
}

func writeHeaderInt(wr *bytes.Writer, head string, context int) {
	wr.Write([]byte(fmt.Sprintf("%s: %d\r\n", head, context)))
}

var srv Server

type Server struct {
	baseCtx context.Context
	err     error
	Opts    broker.Options
	codec   string

	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	server         *mqtt.Server
	MaxHeaderBytes int

	topicPre     string
	md           []middleware.Middleware
	af           []middleware.Middleware
	CreateDevice func(deviceId string)

	log *log.Helper
}

func NewServer(opts ...ServerOption) *Server {
	srv = Server{
		baseCtx:        context.Background(),
		err:            nil,
		ReadTimeout:    2 * time.Second,
		WriteTimeout:   2 * time.Second,
		IdleTimeout:    2 * time.Second,
		server:         mqtt.NewServer(),
		MaxHeaderBytes: 0,
		topicPre:       "",
		md:             nil,
		af:             nil,
		CreateDevice:   nil,
		log:            nil,
	}

	srv.init(opts...)

	srv.server.InitServer(
		mqtt.WithAddress(srv.Opts.Addrs),
		mqtt.WithCodec("json"))
	//	srv.subscriberReqOpts.Store("topic", &SubscribeOption{})

	return &srv
}

// /sys/[ tenantid ]/[ sn ]    + /service/[ deviceType ]
func (s *Server) RegRoute(topic string, r *Router) {
	s.server.RegisterSubscriber(context.Background(), topic+"/req", func(ctx context.Context, msg broker.Event) error {

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
		bbind, ok := r.route[request.RequestURI]
		if !ok {
			return errors.New("没有注册这个接口," + request.RequestURI)
		}
		req := bbind.Binder()
		encoding.GetCodec(codec).Unmarshal(request.Body, req)

		h := func(c context.Context, in interface{}) (interface{}, error) {
			return bbind.Handler(c, in)
		}
		if len(s.md) > 0 {
			h = middleware.Chain(s.md...)(h)
		}
		resp, err := h(ctx, req)
		if err != nil {
			return err
		}

		wr := bytes.NewWriterSize(64 * 1024)
		writeStatusLine(wr, 200, []byte("scratch"))
		writeHeader(wr, broker.ContentType, fmt.Sprintf("application/%s", request.Header[broker.ContentType]))
		writeHeader(wr, broker.MessageId, request.Header[broker.MessageId][0])
		//	writeHeader(wr, broker.AcceptEncoding, "gzip")
		buf, err := broker.Marshal(encoding.GetCodec(srv.codec), resp)
		if err != nil {
			return err
		}
		writeHeaderInt(wr, broker.ContentLength, len(buf))
		wr.Write([]byte("\r\n"))
		wr.Write(buf)

		srv.server.PublishRaw(topic+"/resp", wr.Buffer())
		return nil
	}, func() broker.Any { return nil })
}

func (s *Server) InitServer(opts ...ServerOption) {
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}
}

func (s *Server) Name() string {
	return "mqtt"
}

func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}

	s.baseCtx = ctx

	s.server.Start(ctx)
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	log.Info("[mqtt] server stopping")
	return s.server.Stop(ctx)
}

func (s *Server) Route(prefix string) *Router {
	return newRouter(prefix, s)
}

const (
	runHooks  = true
	skipHooks = false
)
