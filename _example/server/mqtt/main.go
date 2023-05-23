package main

import (
	"context"
	"github.com/go-kratos/kratos/v2"
	v1 "github.com/luobote55/kratos-transport-rpc/api/v1"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/luobote55/kratos-transport-rpc/broker"
	"github.com/luobote55/kratos-transport-rpc/transport/mqtt"
	api "github.com/tx7do/kratos-transport/_example/api/manual"
)

const (
	EmqxBroker        = "tcp://broker.emqx.io:1883"
	EmqxCnBroker      = "tcp://broker-cn.emqx.io:1883"
	EclipseBroker     = "tcp://mqtt.eclipseprojects.io:1883"
	MosquittoBroker   = "tcp://test.mosquitto.org:1883"
	HiveMQBroker      = "tcp://broker.hivemq.com:1883"
	LocalEmxqBroker   = "tcp://127.0.0.1:1883"
	LocalRabbitBroker = "tcp://user:bitnami@127.0.0.1:1883"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Infof("Topic %s, Headers: %+v, Payload: %+v\n", topic, headers, msg)
	return nil
}

func main() {
	ctx := context.Background()

	mqttSrv := mqtt.NewServer(
		mqtt.WithAddress([]string{EmqxCnBroker}),
		mqtt.WithCodec("json"),
	)

	_ = mqttSrv.RegisterSubscriber(ctx,
		"topic/bobo/#",
		func(ctx context.Context, msg broker.Event) (broker.Any, error) {
			startTime := time.Now()
			<-time.NewTicker(time.Millisecond * 10).C // process time
			req := msg.Data().(*v1.HelloRequest)
			return &v1.HelloReply{
				Message:  req.Name,
				TimeFrom: req.TimeFrom,
				TimeRecv: startTime.UnixNano(),
				TimeTo:   time.Now().UnixNano(),
			}, nil
		}, func(identifier string) broker.Any {
			if identifier == "HelloRequest" {
				return &v1.HelloRequest{}
			}
			return nil
		})

	app := kratos.New(
		kratos.Name("mqtt"),
		kratos.Server(
			mqttSrv,
		),
	)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
