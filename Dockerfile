
#build stage
FROM golang:alpine AS builder
RUN GRPC_HEALTH_PROBE_VERSION=v0.2.2 && \
    wget -qO/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe
ENV GO111MODULE=on
WORKDIR /go/src/app
RUN apk add --no-cache git make
COPY . .
RUN make build-app

#final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /go/src/app/upload-service /upload-service
COPY --from=builder /bin/grpc_health_probe /bin/grpc_health_probe
LABEL Name=upload-service Version=0.0.1
EXPOSE 8080
ENTRYPOINT ./upload-service