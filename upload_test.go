package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"reflect"
	"testing"
	"time"
	pb "upload-service/proto"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// Declaring global variables.
var s3Endpoint string
var newSession = session.Must(session.NewSession())
var s3Client *s3.S3
var lis *bufconn.Listener

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
	newSession = session.New(s3Config)
	s3Client = s3.New(newSession)

	lis = bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer(grpc.MaxRecvMsgSize(10000000))
	server := &UploadHandler{UploadService: UploadService{s3Client: s3Client}}
	pb.RegisterUploadServer(grpcServer, server)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

func bufDialer(string, time.Duration) (net.Conn, error) {
	return lis.Dial()
}

func TestUploadService_UploadFile(t *testing.T) {

	type fields struct {
		s3Client *s3.S3
	}
	type args struct {
		file     io.Reader
		key      *string
		bucket   *string
		metadata map[string]*string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *string
		wantErr bool
	}{
		{
			name:   "upload text file",
			fields: fields{s3Client: s3Client},
			args: args{
				key:      aws.String("testfile.txt"),
				bucket:   aws.String("testbucket"),
				file:     bytes.NewReader([]byte("Hello, World!")),
				metadata: nil,
			},
			wantErr: false,
			want:    aws.String(fmt.Sprintf("%s/testbucket/testfile.txt", s3Endpoint)),
		},
		{
			name:   "upload text file in a folder",
			fields: fields{s3Client: s3Client},
			args: args{
				key:      aws.String("testfolder/testfile.txt"),
				bucket:   aws.String("testbucket"),
				file:     bytes.NewReader([]byte("Hello, World!")),
				metadata: nil,
			},
			wantErr: false,
			want:    aws.String(fmt.Sprintf("%s/testbucket/testfolder/testfile.txt", s3Endpoint)),
		},
		{
			name:   "upload text file with empty key",
			fields: fields{s3Client: s3Client},
			args: args{
				key:      aws.String(""),
				bucket:   aws.String("testbucket"),
				file:     bytes.NewReader([]byte("Hello, World!")),
				metadata: nil,
			},
			wantErr: true,
		},
		{
			name:   "upload text file with empty bucket",
			fields: fields{s3Client: s3Client},
			args: args{
				key:      aws.String("testfile.txt"),
				bucket:   aws.String(""),
				file:     bytes.NewReader([]byte("Hello, World!")),
				metadata: nil,
			},
			wantErr: true,
		},
		{
			name:   "upload text file with nil key",
			fields: fields{s3Client: s3Client},
			args: args{
				key:      nil,
				bucket:   aws.String("testbucket"),
				file:     bytes.NewReader([]byte("Hello, World!")),
				metadata: nil,
			},
			wantErr: true,
		},
		{
			name:   "upload text file with nil bucket",
			fields: fields{s3Client: s3Client},
			args: args{
				key:      aws.String("testfile.txt"),
				bucket:   nil,
				file:     bytes.NewReader([]byte("Hello, World!")),
				metadata: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := UploadService{
				s3Client: tt.fields.s3Client,
			}

			got, err := s.UploadFile(tt.args.file, tt.args.key, tt.args.bucket, tt.args.metadata)
			if (err != nil) != tt.wantErr {
				t.Errorf("UploadService.UploadFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != nil && *got != *tt.want {
				t.Errorf("UploadService.UploadFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUploadHandler_UploadMedia(t *testing.T) {
	hugefile := make([]byte, 5000000)
	rand.Read(hugefile)

	uploadservice := UploadService{
		s3Client: s3Client,
	}
	type fields struct {
		UploadService UploadService
	}
	type args struct {
		ctx     context.Context
		request *pb.UploadMediaRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.UploadMediaResponse
		wantErr bool
	}{
		{
			name:   "UploadMedia - text file",
			fields: fields{UploadService: uploadservice},
			args: args{
				ctx: context.Background(),
				request: &pb.UploadMediaRequest{
					Key:    "testfile.txt",
					Bucket: "testbucket",
					File:   []byte("Hello, World!"),
				},
			},
			wantErr: false,
			want: &pb.UploadMediaResponse{
				Output: fmt.Sprintf("%s/testbucket/testfile.txt", s3Endpoint),
			},
		},
		{
			name:   "UploadMedia - text file - without key",
			fields: fields{UploadService: uploadservice},
			args: args{
				ctx: context.Background(),
				request: &pb.UploadMediaRequest{
					Key:    "",
					Bucket: "testbucket",
					File:   []byte("Hello, World!"),
				},
			},
			wantErr: true,
		},
		{
			name:   "UploadMedia - text file - without bucket",
			fields: fields{UploadService: uploadservice},
			args: args{
				ctx: context.Background(),
				request: &pb.UploadMediaRequest{
					Key:    "testfile.txt",
					Bucket: "",
					File:   []byte("Hello, World!"),
				},
			},
			wantErr: true,
		},
		{
			name:   "UploadMedia - text file - with nil file",
			fields: fields{UploadService: uploadservice},
			args: args{
				ctx: context.Background(),
				request: &pb.UploadMediaRequest{
					Key:    "testfile.txt",
					Bucket: "testbucket",
					File:   nil,
				},
			},
			wantErr: false,
			want: &pb.UploadMediaResponse{
				Output: fmt.Sprintf("%s/testbucket/testfile.txt", s3Endpoint),
			},
		},
		{
			name:   "UploadMedia - text file - huge file",
			fields: fields{UploadService: uploadservice},
			args: args{
				ctx: context.Background(),
				request: &pb.UploadMediaRequest{
					Key:    "testfile.txt",
					Bucket: "testbucket",
					File:   hugefile,
				},
			},
			wantErr: false,
			want: &pb.UploadMediaResponse{
				Output: fmt.Sprintf("%s/testbucket/testfile.txt", s3Endpoint),
			},
		},
	}

	// Create connection to server
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	// Create client
	client := pb.NewUploadClient(conn)

	// Iterate over test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.UploadMedia(tt.args.ctx, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("UploadHandler.UploadMedia() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UploadHandler.UploadMedia() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUploadService_UploadInit(t *testing.T) {
	metadata := make(map[string]*string)
	metadata["test"] = aws.String("testt")
	type fields struct {
		s3Client *s3.S3
	}
	type args struct {
		key      *string
		bucket   *string
		metadata map[string]*string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *s3.CreateMultipartUploadOutput
		wantErr bool
	}{
		{
			name:   "init upload",
			fields: fields{s3Client: s3Client},
			args: args{
				key:      aws.String("testfile.txt"),
				bucket:   aws.String("testbucket"),
				metadata: metadata,
			},
			wantErr: false,
			want: &s3.CreateMultipartUploadOutput{
				Bucket: aws.String("testbucket"),
				Key:    aws.String("testfile.txt"),
			},
		},
		{
			name:   "init upload in folder",
			fields: fields{s3Client: s3Client},
			args: args{
				key:      aws.String("testfolder/testfile.txt"),
				bucket:   aws.String("testbucket"),
				metadata: metadata,
			},
			wantErr: false,
			want: &s3.CreateMultipartUploadOutput{
				Bucket: aws.String("testbucket"),
				Key:    aws.String("testfolder/testfile.txt"),
			},
		},
		{
			name:   "init upload with missing key",
			fields: fields{s3Client: s3Client},
			args: args{
				key:      aws.String(""),
				bucket:   aws.String("testbucket"),
				metadata: metadata,
			},
			wantErr: true,
		},
		{
			name:   "init upload with nil key",
			fields: fields{s3Client: s3Client},
			args: args{
				key:      nil,
				bucket:   aws.String("testbucket"),
				metadata: metadata,
			},
			wantErr: true,
		},
		{
			name:   "init upload with missing bucket",
			fields: fields{s3Client: s3Client},
			args: args{
				key:      aws.String("testfile.txt"),
				bucket:   aws.String(""),
				metadata: metadata,
			},
			wantErr: true,
		},
		{
			name:   "init upload with nil bucket",
			fields: fields{s3Client: s3Client},
			args: args{
				key:      aws.String("testfile.txt"),
				bucket:   nil,
				metadata: metadata,
			},
			wantErr: true,
		},
		{
			name:   "init upload with empty metadata",
			fields: fields{s3Client: s3Client},
			args: args{
				key:      aws.String("testfile.txt"),
				bucket:   aws.String("testbucket"),
				metadata: aws.StringMap(make(map[string]string)),
			},
			wantErr: true,
		},
		{
			name:   "init upload with nil metadata",
			fields: fields{s3Client: s3Client},
			args: args{
				key:      aws.String("testfile.txt"),
				bucket:   aws.String("testbucket"),
				metadata: nil,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := UploadService{
				s3Client: tt.fields.s3Client,
			}
			got, err := s.UploadInit(tt.args.key, tt.args.bucket, tt.args.metadata)
			if (err != nil) != tt.wantErr {
				t.Errorf("UploadService.UploadInit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) && got.UploadId == nil {
				t.Errorf("UploadService.UploadInit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUploadHandler_UploadInit(t *testing.T) {
	metadata := make(map[string]string)
	metadata["test"] = "testt"
	uploadservice := UploadService{
		s3Client: s3Client,
	}
	type fields struct {
		UploadService UploadService
	}
	type args struct {
		ctx     context.Context
		request *pb.UploadInitRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.UploadInitResponse
		wantErr bool
	}{
		{
			name:   "UploadInit",
			fields: fields{UploadService: uploadservice},
			args: args{
				ctx: context.Background(),
				request: &pb.UploadInitRequest{
					Key:      "testfile.txt",
					Bucket:   "testbucket",
					Metadata: metadata,
				},
			},
			wantErr: false,
			want: &pb.UploadInitResponse{
				Key:    "testfile.txt",
				Bucket: "testbucket",
			},
		},
		{
			name:   "UploadInit folder",
			fields: fields{UploadService: uploadservice},
			args: args{
				ctx: context.Background(),
				request: &pb.UploadInitRequest{
					Key:      "testfolder/testfile.txt",
					Bucket:   "testbucket",
					Metadata: metadata,
				},
			},
			wantErr: false,
			want: &pb.UploadInitResponse{
				Key:    "testfolder/testfile.txt",
				Bucket: "testbucket",
			},
		},
		{
			name:   "UploadInit with empty key",
			fields: fields{UploadService: uploadservice},
			args: args{
				ctx: context.Background(),
				request: &pb.UploadInitRequest{
					Key:      "",
					Bucket:   "testbucket",
					Metadata: metadata,
				},
			},
			wantErr: true,
		},
		{
			name:   "UploadInit with empty bucket",
			fields: fields{UploadService: uploadservice},
			args: args{
				ctx: context.Background(),
				request: &pb.UploadInitRequest{
					Key:      "testfile.txt",
					Bucket:   "",
					Metadata: metadata,
				},
			},
			wantErr: true,
		},
		{
			name:   "UploadInit with empty metadata",
			fields: fields{UploadService: uploadservice},
			args: args{
				ctx: context.Background(),
				request: &pb.UploadInitRequest{
					Key:      "testfile.txt",
					Bucket:   "testbucket",
					Metadata: make(map[string]string),
				},
			},
			wantErr: true,
		},
		{
			name:   "UploadInit with nil metadata",
			fields: fields{UploadService: uploadservice},
			args: args{
				ctx: context.Background(),
				request: &pb.UploadInitRequest{
					Key:      "testfile.txt",
					Bucket:   "testbucket",
					Metadata: nil,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := UploadHandler{
				UploadService: tt.fields.UploadService,
			}
			got, err := h.UploadInit(tt.args.ctx, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("UploadHandler.UploadInit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) && &got.UploadId == nil {
				t.Errorf("UploadHandler.UploadInit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUploadService_UploadComplete(t *testing.T) {
	type fields struct {
		s3Client *s3.S3
	}
	type args struct {
		uploadID *string
		key      *string
		bucket   *string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *s3.CompleteMultipartUploadOutput
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := UploadService{
				s3Client: tt.fields.s3Client,
			}
			got, err := s.UploadComplete(tt.args.uploadID, tt.args.key, tt.args.bucket)
			if (err != nil) != tt.wantErr {
				t.Errorf("UploadService.UploadComplete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UploadService.UploadComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUploadHandler_UploadComplete(t *testing.T) {
	type fields struct {
		UploadService UploadService
	}
	type args struct {
		ctx     context.Context
		request *pb.UploadCompleteRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.UploadCompleteResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := UploadHandler{
				UploadService: tt.fields.UploadService,
			}
			got, err := h.UploadComplete(tt.args.ctx, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("UploadHandler.UploadComplete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UploadHandler.UploadComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUploadService_UploadPart(t *testing.T) {
	type fields struct {
		s3Client *s3.S3
	}
	type args struct {
		uploadID   *string
		key        *string
		bucket     *string
		partNumber *int64
		body       io.ReadSeeker
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *s3.UploadPartOutput
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := UploadService{
				s3Client: tt.fields.s3Client,
			}
			got, err := s.UploadPart(tt.args.uploadID, tt.args.key, tt.args.bucket, tt.args.partNumber, tt.args.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("UploadService.UploadPart() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UploadService.UploadPart() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUploadHandler_UploadPart(t *testing.T) {
	type fields struct {
		UploadService UploadService
	}
	type args struct {
		stream pb.Upload_UploadPartServer
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := UploadHandler{
				UploadService: tt.fields.UploadService,
			}
			if err := h.UploadPart(tt.args.stream); (err != nil) != tt.wantErr {
				t.Errorf("UploadHandler.UploadPart() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
