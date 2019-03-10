package main

import (
	"log"
	"os/signal"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
)

func main() {
	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
    s3SecretKey := os.Getenv("S3_SECRET_KEY")
    s3Endpoint  := os.Getenv("S3_ENDPOINT")
	s3Token     := ""

	// Configure to use S3 Server
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(s3AccessKey, s3SecretKey, s3Token),
		Endpoint:         aws.String(s3Endpoint),
		Region:           aws.String("eu-east-1"),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
	}

	newSession := session.New(s3Config)
	s3Client := s3.New(newSession)
	uploadHandler := UploadHandler{
		UploadService: UploadService{s3Client: s3Client},
		BucketService: BucketService{s3Client: s3Client},
	}
	r := mux.NewRouter()
	r.HandleFunc("/upload", uploadHandler.Upload).Methods("POST")
	http.Handle("/", r)
	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
    }()
	signalChan := make(chan os.Signal, 1)
    signal.Notify(signalChan, os.Interrupt)
    <-signalChan
}
