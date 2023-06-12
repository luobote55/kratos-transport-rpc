package main

import (
	"flag"
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
)

/*

protoc --proto_path=. --proto_path=./third_party --proto_path=C:\Users\Administrator\go\pkg\mod\github.com\go-kratos\kratos\v2@v2.6.2 --proto_path=C:\Users\Administrator\go\pkg\mod\github.com\go-kratos\kratos\v2@v2.6.2\third_party --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. --go-http_out=paths=source_relative:. --go-mqtt_out=paths=source_relative:. --go-errors_out=paths=source_relative:. --openapi_out=paths=source_relative:. api/helloworld/v1/greeter.proto



protoc --proto_path=. --proto_path=./third_party --proto_path=./fourth_party --proto_path=C:\Users\Administrator\go\pkg\mod\github.com\go-kratos\kratos\v2@v2.6.2 --proto_path=C:\Users\Administrator\go\pkg\mod\github.com\go-kratos\kratos\v2@v2.6.2\third_party --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. --go-http_out=paths=source_relative:. --go-mqtt_out=paths=source_relative:. --go-errors_out=paths=source_relative:. --openapi_out=paths=source_relative:. api/v1/greeter.proto

protoc --proto_path=. --proto_path=./third_party --proto_path=./fourth_party --proto_path=C:\Users\panda\go\pkg\mod\github.com\go-kratos\kratos\v2@v2.6.2 --proto_path=C:\Users\panda\go\pkg\mod\github.com\go-kratos\kratos\v2@v2.6.2\third_party --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. --go-http_out=paths=source_relative:. --go-mqtt_out=paths=source_relative:. --go-errors_out=paths=source_relative:. --openapi_out=paths=source_relative:. api/v1/greeter.proto

protoc --proto_path=. --proto_path=./third_party --proto_path=./fourth_party --proto_path=C:\Users\panda\go\pkg\mod\github.com\go-kratos\kratos\v2@v2.6.2 --proto_path=C:\Users\panda\go\pkg\mod\github.com\go-kratos\kratos\v2@v2.6.2\third_party --go_out=paths=source_relative:. --go-mqtt_out=paths=source_relative:. --go-errors_out=paths=source_relative:. --openapi_out=paths=source_relative:. api/v1/greeter.proto
protoc --proto_path=. --proto_path=./fourth_party --proto_path=C:\Users\panda\go\pkg\mod\github.com\go-kratos\kratos\v2@v2.6.2 --proto_path=C:\Users\panda\go\pkg\mod\github.com\go-kratos\kratos\v2@v2.6.2\third_party --go_out=paths=source_relative:. --go-mqtt_out=paths=source_relative:. --go-errors_out=paths=source_relative:. --openapi_out=paths=source_relative:. api/v1/greeter.proto

protoc --proto_path=. --proto_path=./third_party --proto_path=./fourth_party --go_out=paths=source_relative:. --go-mqt_out=paths=source_relative:. --go-errors_out=paths=source_relative:. --openapi_out=paths=source_relative:. api/v1/greeter.proto


*/

var (
	showVersion = flag.Bool("version", false, "print the version and exit")
	omitempty   = flag.Bool("omitempty", true, "omit if google.api is empty")
)

func main() {
	flag.Parse()
	if *showVersion {
		fmt.Printf("protoc-gen-go-mqtt %v\n", release)
		return
	}
	protogen.Options{
		ParamFunc: flag.CommandLine.Set,
	}.Run(func(gen *protogen.Plugin) error {
		gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			generateFile(gen, f, *omitempty)
		}
		return nil
	})
}
