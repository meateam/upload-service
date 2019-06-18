#build stage
FROM golang:alpine AS builder
ENV GO111MODULE=on
RUN apk add --no-cache git make
RUN GRPC_HEALTH_PROBE_VERSION=v0.3.0 && \
    wget -qO/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe
WORKDIR /go/src/app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build-app

#final stage
FROM scratch
COPY --from=builder /go/src/app/upload-service /upload-service
COPY --from=builder /bin/grpc_health_probe /bin/grpc_health_probe
LABEL Name=upload-service Version=0.0.1
EXPOSE 8080
ENTRYPOINT ["/upload-service"]