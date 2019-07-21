# Basic go commands
PROTOC=protoc

# Binary names
BINARY_NAME=upload-service

all: clean deps fmt test build
build: build-proto build-app
test:
		docker-compose -f "docker-compose.yml" up -d minio && \
		S3_ACCESS_KEY=F6WUUG27HBUFSIXVZL59 S3_SECRET_KEY=BPlIUU6SX0ZxiCMo3tIpCMAUdnmkN9Eo9K42NsRR S3_ENDPOINT=http://127.0.0.1:9000 go test -v ./... && \
		docker-compose down && sudo rm -rf data
clean:
		go clean
		sudo rm -rf $(BINARY_NAME)
run: build
		./$(BINARY_NAME)
		S3_ACCESS_KEY=F6WUUG27HBUFSIXVZL59 S3_SECRET_KEY=BPlIUU6SX0ZxiCMo3tIpCMAUdnmkN9Eo9K42NsRR S3_ENDPOINT=http://127.0.0.1:9000 ./$(BINARY_NAME)
deps:
		go get -u github.com/golang/protobuf/protoc-gen-go
build-app:
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME) -v
build-proto:
		rm -f proto/*.pb.go
		protoc -I proto/ proto/*.proto --go_out=plugins=grpc:./proto
fmt:
		./gofmt.sh