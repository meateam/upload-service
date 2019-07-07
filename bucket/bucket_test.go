package bucket_test

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/meateam/upload-service/bucket"
	"github.com/meateam/upload-service/internal/test"
)

// Declaring global variables.
var s3Endpoint string
var s3Client *s3.S3
var mu sync.Mutex

func init() {
	// Fetch env vars
	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	s3SecretKey := os.Getenv("S3_SECRET_KEY")
	s3Endpoint = os.Getenv("S3_ENDPOINT")
	s3Token := ""

	// Configure to use S3 Server
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(s3AccessKey, s3SecretKey, s3Token),
		Endpoint:         aws.String(s3Endpoint),
		Region:           aws.String("eu-east-1"),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
	}

	// Init real client.
	newSession, err := session.NewSession(s3Config)
	if err != nil {
		log.Fatalf(err.Error())
	}
	s3Client = s3.New(newSession)

	mu.Lock()
	if err := test.EmptyAndDeleteBucket(s3Client, "testbucket"); err != nil {
		log.Printf("test.EmptyAndDeleteBucket failed with error: %v", err)
	}
	if err := test.EmptyAndDeleteBucket(s3Client, "testbucket1"); err != nil {
		log.Printf("test.EmptyAndDeleteBucket failed with error: %v", err)
	}
	if err := test.EmptyAndDeleteBucket(s3Client, "t874777-omer"); err != nil {
		log.Printf("test.EmptyAndDeleteBucket failed with error: %v", err)
	}
	mu.Unlock()
}

func TestBucketService_CreateBucket(t *testing.T) {
	type fields struct {
		s3Client *s3.S3
	}
	type args struct {
		ctx    aws.Context
		bucket *string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:   "create bucket",
			fields: fields{s3Client: s3Client},
			args: args{
				ctx:    context.Background(),
				bucket: aws.String("testbucket"),
			},
			wantErr: false,
			want:    true,
		},
		{
			name:   "create bucket - already exists",
			fields: fields{s3Client: s3Client},
			args: args{
				ctx:    context.Background(),
				bucket: aws.String("testbucket"),
			},
			wantErr: true,
		},
		{
			name:   "create bucket - nil bucket",
			fields: fields{s3Client: s3Client},
			args: args{
				ctx:    context.Background(),
				bucket: nil,
			},
			wantErr: true,
		},
		{
			name:   "create bucket - empty bucket",
			fields: fields{s3Client: s3Client},
			args: args{
				ctx:    context.Background(),
				bucket: aws.String(""),
			},
			wantErr: true,
		},
		{
			name:   "create bucket - invalid bucket",
			fields: fields{s3Client: s3Client},
			args: args{
				ctx:    context.Background(),
				bucket: aws.String("T874777@omer"),
			},
			wantErr: false,
			want:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := bucket.NewService(tt.fields.s3Client)
			mu.Lock()
			got, err := s.CreateBucket(tt.args.ctx, tt.args.bucket)
			mu.Unlock()
			if (err != nil) != tt.wantErr {
				t.Errorf("BucketService.CreateBucket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("BucketService.CreateBucket() = %v, want %v", got, tt.want)
			}
		})

	}
}
func TestBucketService_BucketExists(t *testing.T) {
	s := bucket.NewService(s3Client)

	mu.Lock()
	if _, err := s.CreateBucket(context.Background(), aws.String("testbucket")); err != nil {
		log.Printf("CreateBucket failed with error: %v", err)
	}
	mu.Unlock()
	type fields struct {
		s3Client *s3.S3
	}
	type args struct {
		ctx    aws.Context
		bucket *string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "Bucket Exists",
			fields: fields{s3Client: s3Client},
			args: args{
				ctx:    context.Background(),
				bucket: aws.String("testbucket"),
			},
			want: true,
		},
		{
			name:   "Bucket doesn't Exists",
			fields: fields{s3Client: s3Client},
			args: args{
				ctx:    context.Background(),
				bucket: aws.String("testbucket2"),
			},
			want: false,
		},
		{
			name:   "Bucket nil",
			fields: fields{s3Client: s3Client},
			args: args{
				ctx:    context.Background(),
				bucket: nil,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := bucket.NewService(tt.fields.s3Client)

			if got := s.BucketExists(tt.args.ctx, tt.args.bucket); got != tt.want {
				t.Errorf("BucketService.BucketExists() = %v, want %v", got, tt.want)
			}
		})
	}
}
