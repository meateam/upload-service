module github.com/meateam/upload-service

go 1.13

require (
	github.com/aws/aws-sdk-go v1.23.21
	github.com/golang/protobuf v1.3.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0
	github.com/meateam/elasticsearch-logger v1.1.3-0.20190901111807-4e8b84fb9fda
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/viper v1.4.0
	go.elastic.co/apm/module/apmhttp v1.5.0
	golang.org/x/net v0.0.0-20190912160710-24e19bdeb0f2
	google.golang.org/grpc v1.23.1
)

replace github.com/meateam/upload-service/bucket => ./bucket

replace github.com/meateam/upload-service/object => ./object

replace github.com/meateam/upload-service/internal/test => ./internal/test

replace github.com/meateam/upload-service/server => ./server
