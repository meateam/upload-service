# upload-service
S3 file Upload Service

## compile proto
In order to compile the proto file make sure you have `protobuf` and `protoc-gen-go`

[example guide to gRPC and protobuf](https://grpc.io/docs/quickstart/go.html)

**Compiling Command:**
`protoc -I proto/ proto/upload_service.proto --go_out=plugins=grpc:./proto`