module github.com/luobote55/kratos-transport-rpc/cmd/protoc-gen-go-mqtt

//replace github.com/luobote55/kratos-transport-rpc/ => ../../

//replace github.com/luobote55/kratos-transport-rpc/transport/mqtt => ../../transport/mqtt
//
//replace github.com/luobote55/kratos-transport-rpc/broker/mqtt => ../../broker/mqtt
//
replace github.com/luobote55/kratos-transport-rpc/fourth_party/gogogle/api => ../../fourth_party/gogogle/api

go 1.20

require (
	github.com/luobote55/kratos-transport-rpc/fourth_party/gogogle/api v0.0.0-00010101000000-000000000000
	github.com/luobote55/kratos-transport-rpc/transport/mqtt v0.0.0-20230603014742-a3c4b51c663a
	google.golang.org/protobuf v1.30.0
)

require (
	github.com/eclipse/paho.mqtt.golang v1.4.2 // indirect
	github.com/go-kratos/kratos/v2 v2.6.2 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/form/v4 v4.2.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/luobote55/kratos-transport-rpc v0.0.7 // indirect
	github.com/luobote55/kratos-transport-rpc/broker/mqtt v0.0.0-20230603012757-306c4bdf4e01 // indirect
	github.com/openzipkin/zipkin-go v0.4.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/tx7do/kratos-transport v1.0.5 // indirect
	go.opentelemetry.io/otel v1.16.0 // indirect
	go.opentelemetry.io/otel/exporters/jaeger v1.11.2 // indirect
	go.opentelemetry.io/otel/exporters/zipkin v1.11.2 // indirect
	go.opentelemetry.io/otel/metric v1.16.0 // indirect
	go.opentelemetry.io/otel/sdk v1.15.1 // indirect
	go.opentelemetry.io/otel/trace v1.16.0 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sync v0.0.0-20220513210516-0976fa681c29 // indirect
	golang.org/x/sys v0.8.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230525234020-1aefcd67740a // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
