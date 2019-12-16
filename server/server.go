package server

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	ilogger "github.com/meateam/elasticsearch-logger"
	"github.com/meateam/upload-service/object"
	pb "github.com/meateam/upload-service/proto"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.elastic.co/apm/module/apmhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
	configPort                 = "tcp_port"
	configHealthCheckInterval  = "health_check_interval"
	configElasticAPMIgnoreURLS = "us_elastic_apm_ignore_urls"
	configS3Endpoint           = "s3_endpoint"
	configS3Token              = "s3_token"
	configS3AccessKey          = "s3_access_key"
	configS3SecretKey          = "s3_secret_key"
	configS3Region             = "s3_region"
	configS3SSL                = "s3_ssl"
)

func init() {
	viper.SetDefault(configPort, "8080")
	viper.SetDefault(configHealthCheckInterval, 3)
	viper.SetDefault(configElasticAPMIgnoreURLS, "/grpc.health.v1.Health/Check")
	viper.SetDefault(configS3Endpoint, "http://localhost:9000")
	viper.SetDefault(configS3Token, "")
	viper.SetDefault(configS3AccessKey, "")
	viper.SetDefault(configS3SecretKey, "")
	viper.SetDefault(configS3Region, "us-east-1")
	viper.SetDefault(configS3SSL, false)
	viper.AutomaticEnv()
}

// UploadServer is a structure that holds the upload server.
type UploadServer struct {
	*grpc.Server
	logger              *logrus.Logger
	tcpPort             string
	healthCheckInterval int
	objectHandler       *object.Handler
}

// GetHandler returns a copy of the underlying upload handler.
func (s *UploadServer) GetHandler() *object.Handler {
	return s.objectHandler
}

// Serve accepts incoming connections on the listener `lis`, creating a new
// ServerTransport and service goroutine for each. The service goroutines
// read gRPC requests and then call the registered handlers to reply to them.
// Serve returns when `lis.Accept` fails with fatal errors. `lis` will be closed when
// this method returns.
// If `lis` is nil then Serve creates a `net.Listener` with "tcp" network listening
// on the configured `TCP_PORT`, which defaults to "8080".
// Serve will return a non-nil error unless Stop or GracefulStop is called.
func (s UploadServer) Serve(lis net.Listener) {
	listener := lis
	if lis == nil {
		l, err := net.Listen("tcp", ":"+s.tcpPort)
		if err != nil {
			s.logger.Fatalf("failed to listen: %v", err)
		}

		listener = l
	}

	s.logger.Infof("listening and serving grpc server on port %s", s.tcpPort)
	if err := s.Server.Serve(listener); err != nil {
		s.logger.Fatalf("%v", err)
	}
}

// NewServer configures and creates a grpc.Server instance with the upload service
// health check service.
// Configure using environment variables.
// `HEALTH_CHECK_INTERVAL`: Interval to update serving state of the health check server.
// `S3_ACCESS_KEY`: S3 accress key to connect with s3 backend.
// `S3_SECRET_KEY`: S3 secret key to connect with s3 backend.
// `S3_ENDPOINT`: S3 endpoint of s3 backend to connect to.
// `S3_TOKEN`: S3 token of s3 backend to connect to.
// `S3_REGION`: S3 ergion of s3 backend to connect to.
// `S3_SSL`: Enable or Disable SSL on S3 connection.
// `TCP_PORT`: TCP port on which the grpc server would serve on.
func NewServer(logger *logrus.Logger) *UploadServer {
	// Configuration variables
	s3AccessKey := viper.GetString(configS3AccessKey)
	s3SecretKey := viper.GetString(configS3SecretKey)
	s3Endpoint := viper.GetString(configS3Endpoint)
	s3Token := viper.GetString(configS3Token)
	s3Region := viper.GetString(configS3Region)
	s3SSL := viper.GetBool(configS3SSL)

	// Configure to use S3 Server
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(s3AccessKey, s3SecretKey, s3Token),
		Endpoint:         aws.String(s3Endpoint),
		Region:           aws.String(s3Region),
		DisableSSL:       aws.Bool(!s3SSL),
		S3ForcePathStyle: aws.Bool(true),
		HTTPClient:       apmhttp.WrapClient(http.DefaultClient),
	}

	// If no logger is given, create a new default logger for the server.
	if logger == nil {
		logger = ilogger.NewLogger()
	}

	// Open a session to s3.
	newSession, err := session.NewSession(s3Config)
	if err != nil {
		logger.Fatalf(err.Error())
	}
	logger.Infof("connected to S3 - %s", s3Endpoint)

	// Create a client from the s3 session.
	s3Client := s3.New(newSession)

	// Set up grpc server opts with logger interceptor.
	serverOpts := append(
		serverLoggerInterceptor(logger),
		grpc.MaxRecvMsgSize(5120<<20),
	)

	grpcServer := grpc.NewServer(
		serverOpts...,
	)

	// Create a upload handler and register it on the grpc server.
	objectHandler := object.NewHandler(
		object.NewService(s3Client),
		logger,
	)
	pb.RegisterUploadServer(grpcServer, objectHandler)

	// Create a health server and register it on the grpc server.
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	uploadServer := &UploadServer{
		Server:              grpcServer,
		logger:              logger,
		tcpPort:             viper.GetString(configPort),
		healthCheckInterval: viper.GetInt(configHealthCheckInterval),
		objectHandler:       objectHandler,
	}

	// Health check validation goroutine worker.
	go uploadServer.healthCheckWorker(healthServer)

	return uploadServer
}

// serverLoggerInterceptor configures the logger interceptor for the upload server.
func serverLoggerInterceptor(logger *logrus.Logger) []grpc.ServerOption {
	// Create new logrus entry for logger interceptor.
	logrusEntry := logrus.NewEntry(logger)

	ignorePayload := ilogger.IgnoreServerMethodsDecider(
		append(
			strings.Split(viper.GetString(configElasticAPMIgnoreURLS), ","),
			"/upload.Upload/UploadMedia",
			"/upload.Upload/UploadMultipart",
			"/upload.Upload/UploadPart",
		)...,
	)

	ignoreInitialRequest := ilogger.IgnoreServerMethodsDecider(
		append(
			strings.Split(viper.GetString(configElasticAPMIgnoreURLS), ","),
			"/upload.Upload/UploadMedia",
			"/upload.Upload/UploadMultipart",
			"/upload.Upload/UploadPart",
		)...,
	)

	// Shared options for the logger, with a custom gRPC code to log level function.
	loggerOpts := []grpc_logrus.Option{
		grpc_logrus.WithDecider(func(fullMethodName string, err error) bool {
			return ignorePayload(fullMethodName)
		}),
		grpc_logrus.WithLevels(grpc_logrus.DefaultCodeToLevel),
	}

	return ilogger.ElasticsearchLoggerServerInterceptor(
		logrusEntry,
		ignorePayload,
		ignoreInitialRequest,
		loggerOpts...,
	)
}

// healthCheckWorker is running an infinite loop that sets the serving status once
// in s.healthCheckInterval seconds.
func (s UploadServer) healthCheckWorker(healthServer *health.Server) {
	s3Client := s.objectHandler.GetService().GetS3Client()

	for {
		_, err := s3Client.ListBuckets(&s3.ListBucketsInput{})
		if err != nil {
			healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		} else {
			healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
		}

		time.Sleep(time.Second * time.Duration(s.healthCheckInterval))
	}
}
