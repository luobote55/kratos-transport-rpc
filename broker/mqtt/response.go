package mqtt

import (
	"bufio"
	"context"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// A response represents the server side of an HTTP response.
type response struct {
	mc         MQTT.Client
	req        MQTT.Message       // request for this response
	cancelCtx  context.CancelFunc // when ServeHTTP exits
	w          *bufio.Writer      // buffers output in chunks to chunkWriter
	status     string             // e.g. "200 OK"
	statusCode int                // e.g. 200
	Proto      string             // e.g. "HTTP/1.0"
	ProtoMajor int                // e.g. 1
	ProtoMinor int                // e.g. 0
}

func (w *response) Write(data []byte) (n int, err error) {
	return w.w.Write(data)
}

func (w *response) WriteHeader(statusCode int) {
	w.statusCode = statusCode

}
