module github.com/luobote55/kratos-transport-rpc/broker/mqtt

go 1.20

replace github.com/luobote55/kratos-transport-rpc => ../../

replace github.com/luobote55/kratos-transport-rpc/transport/mqtt => ../../transport/mqtt

replace github.com/luobote55/kratos-transport-rpc/broker/mqtt => ../../broker/mqtt

require (
	github.com/eclipse/paho.mqtt.golang v1.4.2
	github.com/go-kratos/kratos/v2 v2.6.2
	github.com/luobote55/kratos-transport-rpc v0.0.5
	github.com/pkg/errors v0.9.1
	google.golang.org/protobuf v1.30.0
)

require (
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/openzipkin/zipkin-go v0.4.1 // indirect
	github.com/tx7do/kratos-transport v1.0.5 // indirect
	go.opentelemetry.io/otel v1.15.1 // indirect
	go.opentelemetry.io/otel/exporters/jaeger v1.11.2 // indirect
	go.opentelemetry.io/otel/exporters/zipkin v1.11.2 // indirect
	go.opentelemetry.io/otel/sdk v1.15.1 // indirect
	go.opentelemetry.io/otel/trace v1.15.1 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sync v0.0.0-20220513210516-0976fa681c29 // indirect
	golang.org/x/sys v0.8.0 // indirect
)
