
#build stage
FROM golang:alpine AS builder
WORKDIR /go/src/app
COPY . .
RUN apk add --no-cache git
RUN go get -d -v ./...
RUN go install -v ./...

#final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /go/bin/app /app
ENV S3_ACCESS_KEY=F6WUUG27HBUFSIXVZL59
ENV S3_SECRET_KEY=BPlIUU6SX0ZxiCMo3tIpCMAUdnmkN9Eo9K42NsRR
ENV S3_ENDPOINT=http://minio:9000
ENTRYPOINT ./app
LABEL Name=upload-service Version=0.0.1
EXPOSE 8080
