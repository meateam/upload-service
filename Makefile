# Basic go commands
PROTOC=protoc

# Binary names
BINARY_NAME=upload-service

all: test build
build:
		rm -f proto/*.pb.go
		protoc -I proto/ proto/*.proto --go_out=plugins=grpc:./proto
		go build -o $(BINARY_NAME) -v
test:
		docker-compose -f "docker-compose.yml" up -d minio && \
		S3_ACCESS_KEY=F6WUUG27HBUFSIXVZL59 S3_SECRET_KEY=BPlIUU6SX0ZxiCMo3tIpCMAUdnmkN9Eo9K42NsRR S3_ENDPOINT=http://127.0.0.1:9000 go test -v ./... && \
		docker-compose down && rm -rf data
clean:
		go clean
		rm -rf $(BINARY_NAME) data
run:
		go build -o $(BINARY_NAME) -v
		S3_ACCESS_KEY=F6WUUG27HBUFSIXVZL59 S3_SECRET_KEY=BPlIUU6SX0ZxiCMo3tIpCMAUdnmkN9Eo9K42NsRR S3_ENDPOINT=http://127.0.0.1:9000 ./$(BINARY_NAME)
deps:
		go get -u github.com/golang/protobuf