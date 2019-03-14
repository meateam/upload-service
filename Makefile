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
		go test -v ./...
clean: 
		go clean
		rm -f $(BINARY_NAME)
run:
		go build -o $(BINARY_NAME) -v
		./$(BINARY_NAME)
deps:
		go get -u github.com/golang/protobuf/protoc-gen-go