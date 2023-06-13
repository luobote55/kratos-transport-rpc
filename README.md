# v0.1.1
# 调用rpc化,
1. mqtt的调用模拟成rpc调用
2. 打包成http报文，以实现metadata
3. Marshal/Unmarshal any type for format json/proto.
4. kratos proto server ./xxx.proto --->   xxx_mqt.pb.go

# todo：
1. middleware for Metrics/Tracing/Logging

# 编译kratos
cd .\cmd\kratos\ && go build
# 编译protoc-gen-go-mqt
cd .\cmd\protoc-gen-go-mqt\ && go build

## 拷贝kratos、protoc-gen-go-mqt至bin目录
kratos proto client ./api/v1/greeter.proto
## 产生greeter_mqt_pb.go

cd .\server\mqtt\  
go test -v

reference:
[Kratos官方示例代码库](https://github.com/go-kratos/examples)
[Kratos](https://github.com/go-kratos/kratos)
[kratos-transport](https://github.com/tx7do/kratos-transport)
