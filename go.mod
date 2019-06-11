module github.com/meateam/upload-service

go 1.12

require (
	github.com/aws/aws-sdk-go v1.19.22
	github.com/golang/protobuf v1.3.1
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/meateam/elasticsearch-logger v1.1.0
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/viper v1.3.2
	golang.org/x/net v0.0.0-20190311183353-d8887717615a
	google.golang.org/grpc v1.21.0
)

replace github.com/meateam/upload-service/bucket => ./bucket

replace github.com/meateam/upload-service/upload => ./upload

replace github.com/meateam/upload-service/internal/test => ./internal/test

replace github.com/meateam/upload-service/server => ./server

replace go.elastic.co/apm/module/apmgrpc => github.com/omrishtam/apm-agent-go/module/apmgrpc v1.3.1-0.20190514172539-1b2e35db8668
