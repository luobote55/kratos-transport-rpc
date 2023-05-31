module github.com/luobote55/kratos-transport-rpc/cmd/protoc-gen-go-mqt

go 1.17

replace github.com/luobote55/kratos-transport-rpc/ => ../../

replace github.com/luobote55/kratos-transport-rpc/transport/mqtt => ../../transport/mqtt

replace github.com/luobote55/kratos-transport-rpc/broker/mqtt => ../../broker/mqtt

replace github.com/luobote55/kratos-transport-rpc/fourth_party/gogogle/api => ../../fourth_party/gogogle/api

require google.golang.org/protobuf v1.30.0
