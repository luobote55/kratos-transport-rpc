module github.com/tx7do/kratos-transport/_example/server/mqtt

go 1.20

replace github.com/luobote55/kratos-transport-rpc => ../../../

require (
	github.com/go-kratos/kratos/v2 v2.6.2
	github.com/luobote55/kratos-transport-rpc v0.0.6
	github.com/luobote55/kratos-transport-rpc/transport/mqtt v0.0.0-20230523062712-fc73649069d1
	github.com/tx7do/kratos-transport v1.0.5
)

require (
	github.com/eclipse/paho.mqtt.golang v1.4.2 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/form/v4 v4.2.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/luobote55/kratos-transport-rpc/broker/mqtt v0.0.0-20230522080409-647405d7ff76 // indirect
	github.com/openzipkin/zipkin-go v0.4.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	go.opentelemetry.io/otel v1.15.1 // indirect
	go.opentelemetry.io/otel/exporters/jaeger v1.11.2 // indirect
	go.opentelemetry.io/otel/exporters/zipkin v1.11.2 // indirect
	go.opentelemetry.io/otel/sdk v1.15.1 // indirect
	go.opentelemetry.io/otel/trace v1.15.1 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sync v0.0.0-20220513210516-0976fa681c29 // indirect
	golang.org/x/sys v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20221010155953-15ba04fc1c0e // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
