package main

import (
	"net"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	ilogger "github.com/meateam/elasticsearch-logger"
	pb "github.com/meateam/upload-service/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var (
	logger = ilogger.NewLogger()
)

func main() {
	interval := os.Getenv("HEALTH_CHECK_INTERVAL")
	healthCheckInterval, err := strconv.Atoi(interval)
	if err != nil {
		healthCheckInterval = 3
	}
	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	s3SecretKey := os.Getenv("S3_SECRET_KEY")
	s3Endpoint := os.Getenv("S3_ENDPOINT")
	tcpPort := os.Getenv("TCP_PORT")
	s3Token := ""

	// Configure to use S3 Server
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(s3AccessKey, s3SecretKey, s3Token),
		Endpoint:         aws.String(s3Endpoint),
		Region:           aws.String("eu-east-1"),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
	}
	newSession := session.New(s3Config)
	logger.Infof("connected to S3 - %s", s3Endpoint)
	s3Client := s3.New(newSession)
	lis, err := net.Listen("tcp", ":"+tcpPort)
	if err != nil {
		logger.Fatalf("failed to listen: %v", err)
	}
	logger.Infof("listening on port %s", tcpPort)

	logrusEntry := logrus.NewEntry(logger)

	// Shared options for the logger, with a custom gRPC code to log level function.
	loggerOpts := []grpc_logrus.Option{
		grpc_logrus.WithLevels(grpc_logrus.DefaultCodeToLevel),
	}

	fullIgnoredMethodNames := []string{
		"/upload.Upload/UploadMedia",
		"/upload.Upload/UploadMultipart",
		"/upload.Upload/UploadPart",
	}

	serverOpts := append(
		ilogger.ElasticsearchLoggerServerInterceptor(
			logrusEntry,
			ilogger.IgnoreServerMethodsDecider(fullIgnoredMethodNames...),
			loggerOpts...,
		),
		grpc.MaxRecvMsgSize(5120<<20),
	)

	grpcServer := grpc.NewServer(
		serverOpts...,
	)
	server := &UploadHandler{UploadService: &UploadService{s3Client: s3Client}}
	pb.RegisterUploadServer(grpcServer, server)
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	// Health validation GoRoutine
	go func() {
		for {
			_, err := s3Client.ListBuckets(&s3.ListBucketsInput{})
			if err != nil {
				healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
			} else {
				healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
			}
			time.Sleep(time.Second * time.Duration(healthCheckInterval))
		}
	}()
	logger.Infof("serving grpc server on port %s", tcpPort)
	grpcServer.Serve(lis)
}

func ignoreExtractUploadRequest(fullIgnoredMethodNames ...string) func(string) bool {
	return func(fullMethodName string) bool {
		for _, ignoredMethodName := range fullIgnoredMethodNames {
			if ignoredMethodName == fullMethodName {
				return false
			}
		}
		return true
	}
}
