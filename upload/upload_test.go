package upload_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"testing"

	pb "github.com/meateam/upload-service/proto"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/meateam/upload-service/internal/test"
	"github.com/meateam/upload-service/server"
	"github.com/meateam/upload-service/upload"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// Declaring global variable.
var (
	lis        *bufconn.Listener
	s3Client   *s3.S3
	s3Endpoint string
)

func init() {
	lis = bufconn.Listen(bufSize)
	uploadServer := server.NewServer()

	s3Client = uploadServer.GetHandler().GetService().GetS3Client()
	s3Endpoint = s3Client.Endpoint

	go uploadServer.Serve(lis)

	if err := test.EmptyAndDeleteBucket(s3Client, "testbucket"); err != nil {
		log.Printf("test.EmptyAndDeleteBucket failed with error: %v", err)
	}
	if err := test.EmptyAndDeleteBucket(s3Client, "testbucket1"); err != nil {
		log.Printf("test.EmptyAndDeleteBucket failed with error: %v", err)
	}
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestUploadService_UploadFile(t *testing.T) {
	metadata := make(map[string]*string)
	metadata["test"] = aws.String("testt")
	type fields struct {
		s3Client *s3.S3
	}
	type args struct {
		file        io.Reader
		key         *string
		bucket      *string
		contentType *string
		metadata    map[string]*string
		ctx         context.Context
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
				key:         aws.String("testfile.txt"),
				bucket:      aws.String("testbucket"),
				file:        bytes.NewReader([]byte("Hello, World!")),
				contentType: aws.String("text/plain"),
				metadata:    nil,
				ctx:         context.Background(),
			},
			wantErr: false,
			want:    aws.String(fmt.Sprintf("%s/testbucket/testfile.txt", s3Endpoint)),
		},
		{
			name:   "upload text file in a folder",
			fields: fields{s3Client: s3Client},
			args: args{
				key:         aws.String("testfolder/testfile.txt"),
				bucket:      aws.String("testbucket"),
				file:        bytes.NewReader([]byte("Hello, World!")),
				contentType: aws.String("text/plain"),
				metadata:    metadata,
				ctx:         context.Background(),
			},
			wantErr: false,
			want:    aws.String(fmt.Sprintf("%s/testbucket/testfolder/testfile.txt", s3Endpoint)),
		},
		{
			name:   "upload text file with empty key",
			fields: fields{s3Client: s3Client},
			args: args{
				key:         aws.String(""),
				bucket:      aws.String("testbucket"),
				file:        bytes.NewReader([]byte("Hello, World!")),
				contentType: aws.String("text/plain"),
				metadata:    nil,
				ctx:         context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "upload text file with empty bucket",
			fields: fields{s3Client: s3Client},
			args: args{
				key:         aws.String("testfile.txt"),
				bucket:      aws.String(""),
				file:        bytes.NewReader([]byte("Hello, World!")),
				contentType: aws.String("text/plain"),
				metadata:    nil,
				ctx:         context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "upload text file with nil key",
			fields: fields{s3Client: s3Client},
			args: args{
				key:         nil,
				bucket:      aws.String("testbucket"),
				file:        bytes.NewReader([]byte("Hello, World!")),
				contentType: aws.String("text/plain"),
				metadata:    nil,
				ctx:         context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "upload text file with nil bucket",
			fields: fields{s3Client: s3Client},
			args: args{
				key:         aws.String("testfile.txt"),
				bucket:      nil,
				file:        bytes.NewReader([]byte("Hello, World!")),
				contentType: aws.String("text/plain"),
				metadata:    nil,
				ctx:         context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "upload nil file",
			fields: fields{s3Client: s3Client},
			args: args{
				key:         aws.String("testfile.txt"),
				bucket:      aws.String("testbucket"),
				contentType: aws.String("text/plain"),
				file:        nil,
				metadata:    nil,
				ctx:         context.Background(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := upload.NewService(tt.fields.s3Client)

			got, err := s.UploadFile(
				tt.args.ctx,
				tt.args.file,
				tt.args.key,
				tt.args.bucket,
				tt.args.contentType,
				tt.args.metadata,
			)
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
	hugefile := make([]byte, 5<<20)
	if _, err := rand.Read(hugefile); err != nil {
		t.Errorf("Could not generate file with error: %v", err)
	}

	uploadservice := upload.NewService(s3Client)

	type fields struct {
		UploadService *upload.Service
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
					Key:         "testfile.txt",
					ContentType: "text/plain",
					Bucket:      "testbucket",
					File:        []byte("Hello, World!"),
				},
			},
			wantErr: false,
			want: &pb.UploadMediaResponse{
				Location: fmt.Sprintf("%s/testbucket/testfile.txt", s3Endpoint),
			},
		},
		{
			name:   "UploadMedia - text file - without key",
			fields: fields{UploadService: uploadservice},
			args: args{
				ctx: context.Background(),
				request: &pb.UploadMediaRequest{
					Key:         "",
					ContentType: "text/plain",
					Bucket:      "testbucket",
					File:        []byte("Hello, World!"),
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
					Key:         "testfile.txt",
					ContentType: "text/plain",
					Bucket:      "",
					File:        []byte("Hello, World!"),
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
					Key:         "testfile.txt",
					ContentType: "text/plain",
					Bucket:      "testbucket",
					File:        nil,
				},
			},
			wantErr: false,
			want: &pb.UploadMediaResponse{
				Location: fmt.Sprintf("%s/testbucket/testfile.txt", s3Endpoint),
			},
		},
		{
			name:   "UploadMedia - text file - huge file",
			fields: fields{UploadService: uploadservice},
			args: args{
				ctx: context.Background(),
				request: &pb.UploadMediaRequest{
					Key:         "testfile.txt",
					ContentType: "text/plain",
					Bucket:      "testbucket",
					File:        hugefile,
				},
			},
			wantErr: false,
			want: &pb.UploadMediaResponse{
				Location: fmt.Sprintf("%s/testbucket/testfile.txt", s3Endpoint),
			},
		},
	}

	// Create connection to server
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
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
		key         *string
		bucket      *string
		contentType *string
		metadata    map[string]*string
		ctx         context.Context
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
				key:         aws.String("testfile.txt"),
				bucket:      aws.String("testbucket"),
				contentType: aws.String("text/plain"),
				metadata:    metadata,
				ctx:         context.Background(),
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
				key:         aws.String("testfolder/testfile.txt"),
				contentType: aws.String("text/plain"),
				bucket:      aws.String("testbucket"),
				metadata:    metadata,
				ctx:         context.Background(),
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
				key:         aws.String(""),
				bucket:      aws.String("testbucket"),
				contentType: aws.String("text/plain"),
				metadata:    metadata,
				ctx:         context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "init upload with nil key",
			fields: fields{s3Client: s3Client},
			args: args{
				key:         nil,
				bucket:      aws.String("testbucket"),
				contentType: aws.String("text/plain"),
				metadata:    metadata,
				ctx:         context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "init upload with missing bucket",
			fields: fields{s3Client: s3Client},
			args: args{
				key:         aws.String("testfile.txt"),
				bucket:      aws.String(""),
				contentType: aws.String("text/plain"),
				metadata:    metadata,
				ctx:         context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "init upload with nil bucket",
			fields: fields{s3Client: s3Client},
			args: args{
				key:         aws.String("testfile.txt"),
				bucket:      nil,
				contentType: aws.String("text/plain"),
				metadata:    metadata,
				ctx:         context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "init upload with empty metadata",
			fields: fields{s3Client: s3Client},
			args: args{
				key:         aws.String("testfile.txt"),
				bucket:      aws.String("testbucket"),
				contentType: aws.String("text/plain"),
				metadata:    aws.StringMap(make(map[string]string)),
				ctx:         context.Background(),
			},
			wantErr: false,
			want: &s3.CreateMultipartUploadOutput{
				Bucket: aws.String("testbucket"),
				Key:    aws.String("testfile.txt"),
			},
		},
		{
			name:   "init upload with nil metadata",
			fields: fields{s3Client: s3Client},
			args: args{
				key:         aws.String("testfile.txt"),
				bucket:      aws.String("testbucket"),
				contentType: aws.String("text/plain"),
				metadata:    nil,
				ctx:         context.Background(),
			},
			wantErr: false,
			want: &s3.CreateMultipartUploadOutput{
				Bucket: aws.String("testbucket"),
				Key:    aws.String("testfile.txt"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := upload.NewService(tt.fields.s3Client)

			got, err := s.UploadInit(tt.args.ctx, tt.args.key, tt.args.bucket, tt.args.contentType, tt.args.metadata)
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
	uploadservice := upload.NewService(s3Client)

	type fields struct {
		UploadService *upload.Service
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
					Key:         "testfile.txt",
					ContentType: "text/plain",
					Bucket:      "testbucket",
					Metadata:    metadata,
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
					Key:         "testfolder/testfile.txt",
					ContentType: "text/plain",
					Bucket:      "testbucket",
					Metadata:    metadata,
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
					Key:         "",
					ContentType: "text/plain",
					Bucket:      "testbucket",
					Metadata:    metadata,
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
					ContentType: "text/plain",
					Key:         "testfile.txt",
					Bucket:      "",
					Metadata:    metadata,
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
					ContentType: "text/plain",
					Key:         "testfile.txt",
					Bucket:      "testbucket",
					Metadata:    make(map[string]string),
				},
			},
			wantErr: false,
			want: &pb.UploadInitResponse{
				Key:    "testfile.txt",
				Bucket: "testbucket",
			},
		},
		{
			name:   "UploadInit with nil metadata",
			fields: fields{UploadService: uploadservice},
			args: args{
				ctx: context.Background(),
				request: &pb.UploadInitRequest{
					ContentType: "text/plain",
					Key:         "testfile.txt",
					Bucket:      "testbucket",
					Metadata:    nil,
				},
			},
			wantErr: false,
			want: &pb.UploadInitResponse{
				Key:    "testfile.txt",
				Bucket: "testbucket",
			},
		},
	}

	// Create connection to server
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	// Create client
	client := pb.NewUploadClient(conn)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.UploadInit(tt.args.ctx, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("UploadHandler.UploadInit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) && got.UploadId == "" {
				t.Errorf("UploadHandler.UploadInit() = %v, want %v", got, tt.want)
			}
		})
	}
}

//nolint:gocyclo
func TestUploadService_UploadPart(t *testing.T) {
	metadata := make(map[string]*string)
	metadata["test"] = aws.String("meta")
	file := make([]byte, 50<<20)
	if _, err := rand.Read(file); err != nil {
		t.Errorf("Could not generate file with error: %v", err)
	}
	fileReader := bytes.NewReader(file)

	type fields struct {
		s3Client *s3.S3
	}
	type args struct {
		initKey    *string
		initBucket *string
		key        *string
		bucket     *string
		partNumber *int64
		body       io.ReadSeeker
		ctx        context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "upload part",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("partfile.txt"),
				initBucket: aws.String("testbucket"),
				key:        aws.String("partfile.txt"),
				bucket:     aws.String("testbucket"),
				partNumber: aws.Int64(1),
				body:       fileReader,
				ctx:        context.Background(),
			},
			wantErr: false,
		},
		{
			name:   "upload part in folder",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("testfolder/partfile.txt"),
				initBucket: aws.String("testbucket"),
				key:        aws.String("testfolder/partfile.txt"),
				bucket:     aws.String("testbucket"),
				partNumber: aws.Int64(1),
				body:       fileReader,
				ctx:        context.Background(),
			},
			wantErr: false,
		},
		{
			name:   "upload part with empty key",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("partfile1.txt"),
				initBucket: aws.String("testbucket"),
				key:        aws.String(""),
				bucket:     aws.String("testbucket"),
				partNumber: aws.Int64(1),
				body:       fileReader,
				ctx:        context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "upload part with nil key",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("partfile2.txt"),
				initBucket: aws.String("testbucket"),
				key:        nil,
				bucket:     aws.String("testbucket"),
				partNumber: aws.Int64(1),
				body:       fileReader,
				ctx:        context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "upload part with key mismatch",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("partfile3.txt"),
				initBucket: aws.String("testbucket"),
				key:        aws.String("partfile.txt"),
				bucket:     aws.String("testbucket"),
				partNumber: aws.Int64(1),
				body:       fileReader,
				ctx:        context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "upload part with empty bucket",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("partfile4.txt"),
				initBucket: aws.String("testbucket"),
				key:        aws.String("partfile4.txt"),
				bucket:     aws.String(""),
				partNumber: aws.Int64(1),
				body:       fileReader,
				ctx:        context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "upload part with nil bucket",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("partfile5.txt"),
				initBucket: aws.String("testbucket"),
				key:        aws.String("partfile5.txt"),
				bucket:     nil,
				partNumber: aws.Int64(1),
				body:       fileReader,
				ctx:        context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "upload part with bucket mismatch",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("partfile6.txt"),
				initBucket: aws.String("testbucket"),
				key:        aws.String("partfile6.txt"),
				bucket:     aws.String("testbucket1"),
				partNumber: aws.Int64(1),
				body:       fileReader,
				ctx:        context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "upload part with nil body",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("partfile6.txt"),
				initBucket: aws.String("testbucket"),
				key:        aws.String("partfile6.txt"),
				bucket:     aws.String("testbucket"),
				partNumber: aws.Int64(1),
				body:       nil,
				ctx:        context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "upload part without part number",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("partfile8.txt"),
				initBucket: aws.String("testbucket"),
				key:        aws.String("partfile8.txt"),
				bucket:     aws.String("testbucket"),
				partNumber: nil,
				body:       fileReader,
				ctx:        context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "upload part with part number lower than 1",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("partfile7.txt"),
				initBucket: aws.String("testbucket"),
				key:        aws.String("partfile7.txt"),
				bucket:     aws.String("testbucket"),
				partNumber: aws.Int64(0),
				body:       fileReader,
				ctx:        context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "upload part with part number greater than 10000",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("partfile7.txt"),
				initBucket: aws.String("testbucket"),
				key:        aws.String("partfile7.txt"),
				bucket:     aws.String("testbucket"),
				partNumber: aws.Int64(10001),
				body:       fileReader,
				ctx:        context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := upload.NewService(tt.fields.s3Client)

			initOutput, err := s.UploadInit(
				tt.args.ctx,
				tt.args.initKey,
				tt.args.initBucket,
				aws.String("text/plain"),
				metadata,
			)
			if err != nil {
				t.Errorf("UploadService.UploadInit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got, err := s.UploadPart(
				tt.args.ctx,
				initOutput.UploadId,
				tt.args.key,
				tt.args.bucket,
				tt.args.partNumber,
				tt.args.body,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("UploadService.UploadPart() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got == nil || got.ETag == nil || *got.ETag == "") != tt.wantErr {
				t.Errorf("UploadService.UploadPart() = %v", got)
			}
		})
	}

	t.Run("UploadPart - nil UploadID", func(t *testing.T) {
		s := upload.NewService(s3Client)

		ctx := context.Background()
		got, err := s.UploadPart(
			ctx,
			nil,
			aws.String("testfile10.txt"),
			aws.String("testbucket"),
			aws.Int64(1),
			fileReader,
		)
		if err == nil {
			t.Errorf("UploadService.UploadPart() error = %v, wantErr %v", err, true)
			return
		}
		if got != nil && (got.ETag != nil || *got.ETag != "") {
			t.Errorf("UploadService.UploadPart() = %v", got)
		}
	})
	t.Run("UploadPart - empty UploadID", func(t *testing.T) {
		s := upload.NewService(s3Client)

		ctx := context.Background()
		got, err := s.UploadPart(
			ctx,
			aws.String(""),
			aws.String("testfile10.txt"),
			aws.String("testbucket"),
			aws.Int64(1),
			fileReader,
		)
		if err == nil {
			t.Errorf("UploadService.UploadPart() error = %v, wantErr %v", err, true)
			return
		}
		if got != nil && (got.ETag != nil || *got.ETag != "") {
			t.Errorf("UploadService.UploadPart() = %v", got)
		}
	})
}

func TestUploadService_UploadComplete(t *testing.T) {
	metadata := make(map[string]*string)
	metadata["test"] = aws.String("meta")
	file := make([]byte, 50<<20)
	if _, err := rand.Read(file); err != nil {
		t.Errorf("Could not generate file with error: %v", err)
	}
	fileReader := bytes.NewReader(file)

	type fields struct {
		s3Client *s3.S3
	}
	type args struct {
		initKey    *string
		initBucket *string
		key        *string
		bucket     *string
		ctx        context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "Upload Complete",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("file.txt"),
				initBucket: aws.String("testbucket"),
				key:        aws.String("file.txt"),
				bucket:     aws.String("testbucket"),
				ctx:        context.Background(),
			},
			wantErr: false,
		},
		{
			name:   "Upload Complete to folder",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("testfolder/file.txt"),
				initBucket: aws.String("testbucket"),
				key:        aws.String("testfolder/file.txt"),
				bucket:     aws.String("testbucket"),
				ctx:        context.Background(),
			},
			wantErr: false,
		},
		{
			name:   "Upload Complete with empty key",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("file.txt"),
				initBucket: aws.String("testbucket"),
				key:        aws.String(""),
				bucket:     aws.String("testbucket"),
				ctx:        context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "Upload Complete with nil key",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("file.txt"),
				initBucket: aws.String("testbucket"),
				key:        nil,
				bucket:     aws.String("testbucket"),
				ctx:        context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "Upload Complete with key mismatch",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("file.txt"),
				initBucket: aws.String("testbucket"),
				key:        aws.String("file1.txt"),
				bucket:     aws.String("testbucket"),
				ctx:        context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "Upload Complete with empty bucket",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("file.txt"),
				initBucket: aws.String("testbucket"),
				key:        aws.String("file.txt"),
				bucket:     aws.String(""),
				ctx:        context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "Upload Complete with nil bucket",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("file.txt"),
				initBucket: aws.String("testbucket"),
				key:        aws.String("file.txt"),
				bucket:     nil,
				ctx:        context.Background(),
			},
			wantErr: true,
		},
		{
			name:   "Upload Complete with bucket mismatch",
			fields: fields{s3Client: s3Client},
			args: args{
				initKey:    aws.String("file.txt"),
				initBucket: aws.String("testbucket"),
				key:        aws.String("file1.txt"),
				bucket:     aws.String("testbucket1"),
				ctx:        context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := upload.NewService(tt.fields.s3Client)

			initOutput, err := s.UploadInit(
				tt.args.ctx,
				tt.args.initKey,
				tt.args.initBucket,
				aws.String("text/plain"),
				metadata,
			)
			if err != nil {
				t.Errorf("UploadService.UploadInit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			_, err = s.UploadPart(
				tt.args.ctx,
				initOutput.UploadId,
				tt.args.initKey,
				tt.args.initBucket,
				aws.Int64(1),
				fileReader)

			if err != nil {
				t.Errorf("UploadService.UploadPart() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got, err := s.UploadComplete(tt.args.ctx, initOutput.UploadId, tt.args.key, tt.args.bucket)
			if (err != nil) != tt.wantErr {
				t.Errorf("UploadService.UploadComplete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got == nil) != tt.wantErr {
				t.Errorf("UploadService.UploadComplete() = %v", got)
			}
		})
	}
	t.Run("UploadComplete - empty uploadID ", func(t *testing.T) {
		s := upload.NewService(s3Client)

		ctx := context.Background()
		got, err := s.UploadComplete(ctx, aws.String(""), aws.String("tests.txt"), aws.String("testbucket"))
		if err == nil {
			t.Errorf("UploadService.UploadComplete() error = %v, wantErr %v", err, true)
			return
		}
		if got != nil {
			t.Errorf("UploadService.UploadComplete() = %v", got)
			return
		}
	})

	t.Run("UploadComplete - nil uploadID ", func(t *testing.T) {
		s := upload.NewService(s3Client)

		ctx := context.Background()
		got, err := s.UploadComplete(ctx, nil, aws.String("tests.txt"), aws.String("testbucket"))
		if err == nil {
			t.Errorf("UploadService.UploadComplete() error = %v, wantErr %v", err, true)
			return
		}
		if got != nil {
			t.Errorf("UploadService.UploadComplete() = %v", got)
			return
		}
	})
}

func TestUploadHandler_UploadMultipart(t *testing.T) {
	// Init global values to use in tests.
	file := make([]byte, 5<<20)
	if _, err := rand.Read(file); err != nil {
		t.Errorf("Could not generate file with error: %v", err)
	}
	metadata := make(map[string]string)
	metadata["test"] = "testt"

	type args struct {
		ctx     context.Context
		request *pb.UploadMultipartRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.UploadMultipartResponse
		wantErr bool
	}{
		{
			name: "Upload Multipart",
			args: args{
				ctx: context.Background(),
				request: &pb.UploadMultipartRequest{
					Key:         "testfile.txt",
					Bucket:      "testbucket",
					ContentType: "text/plain",
					File:        file,
					Metadata:    metadata,
				},
			},
			wantErr: false,
			want: &pb.UploadMultipartResponse{
				Location: fmt.Sprintf("%s/testbucket/testfile.txt", s3Endpoint),
			},
		},
		{
			name: "Upload Multipart to folder",
			args: args{
				ctx: context.Background(),
				request: &pb.UploadMultipartRequest{
					Key:         "testfolder/testfile.txt",
					ContentType: "text/plain",
					Bucket:      "testbucket",
					File:        file,
					Metadata:    metadata,
				},
			},
			wantErr: false,
			want: &pb.UploadMultipartResponse{
				Location: fmt.Sprintf("%s/testbucket/testfolder/testfile.txt", s3Endpoint),
			},
		},
		{
			name: "Upload Multipart with empty key",
			args: args{
				ctx: context.Background(),
				request: &pb.UploadMultipartRequest{
					Key:         "",
					ContentType: "text/plain",
					Bucket:      "testbucket",
					File:        file,
					Metadata:    metadata,
				},
			},
			wantErr: true,
		},
		{
			name: "Upload Multipart with empty bucket",
			args: args{
				ctx: context.Background(),
				request: &pb.UploadMultipartRequest{
					ContentType: "text/plain",
					Key:         "testfile.txt",
					Bucket:      "",
					File:        file,
					Metadata:    metadata,
				},
			},
			wantErr: true,
		},
		{
			name: "Upload Multipart with nil file",
			args: args{
				ctx: context.Background(),
				request: &pb.UploadMultipartRequest{
					ContentType: "text/plain",
					Key:         "testfile.txt",
					Bucket:      "testbucket",
					File:        nil,
					Metadata:    metadata,
				},
			},
			wantErr: false,
			want: &pb.UploadMultipartResponse{
				Location: fmt.Sprintf("%s/testbucket/testfile.txt", s3Endpoint),
			},
		},
		{
			name: "Upload Multipart with empty file",
			args: args{
				ctx: context.Background(),
				request: &pb.UploadMultipartRequest{
					ContentType: "text/plain",
					Key:         "testfile.txt",
					Bucket:      "testbucket",
					File:        make([]byte, 0),
					Metadata:    metadata,
				},
			},
			wantErr: false,
			want: &pb.UploadMultipartResponse{
				Location: fmt.Sprintf("%s/testbucket/testfile.txt", s3Endpoint),
			},
		},
		{
			name: "Upload Multipart with nil metadata",
			args: args{
				ctx: context.Background(),
				request: &pb.UploadMultipartRequest{
					Key:         "testfile.txt",
					ContentType: "text/plain",
					Bucket:      "testbucket",
					File:        file,
					Metadata:    nil,
				},
			},
			wantErr: true,
		},
		{
			name: "Upload Multipart with empty metadata",
			args: args{
				ctx: context.Background(),
				request: &pb.UploadMultipartRequest{
					Key:         "testfile.txt",
					ContentType: "text/plain",
					Bucket:      "testbucket",
					File:        file,
					Metadata:    make(map[string]string),
				},
			},
			wantErr: true,
		},
	}

	// Create connection to server
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	// Create client
	client := pb.NewUploadClient(conn)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.UploadMultipart(tt.args.ctx, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("UploadHandler.UploadMultipart() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UploadHandler.UploadMultipart() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUploadService_UploadAbort(t *testing.T) {
	metadata := make(map[string]*string)
	metadata["test"] = aws.String("testt")
	file := make([]byte, 50<<20)
	if _, err := rand.Read(file); err != nil {
		t.Errorf("Could not generate file with error: %v", err)
	}
	fileReader := bytes.NewReader(file)
	type fields struct {
		s3Client *s3.S3
	}
	type args struct {
		ctx    aws.Context
		key    *string
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
			name:   "init upload",
			fields: fields{s3Client: s3Client},
			args: args{
				key:    aws.String("testfile.txt"),
				bucket: aws.String("testbucket"),
				ctx:    context.Background(),
			},
			wantErr: false,
			want:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := upload.NewService(tt.fields.s3Client)

			initOutput, err := s.UploadInit(tt.args.ctx, tt.args.key, tt.args.bucket, aws.String("text/plain"), metadata)
			if err != nil {
				t.Errorf("UploadService.UploadInit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			_, err = s.UploadPart(
				tt.args.ctx,
				initOutput.UploadId,
				tt.args.key,
				tt.args.bucket,
				aws.Int64(1),
				fileReader,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("UploadService.UploadPart() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got, err := s.UploadAbort(tt.args.ctx, initOutput.UploadId, tt.args.key, tt.args.bucket)
			if (err != nil) != tt.wantErr {
				t.Errorf("UploadService.UploadAbort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UploadService.UploadAbort() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TODO: TestUploadHandler_UploadAbort
// TODO: TestUploadHandler_UploadComplete
// TODO: TestUploadHandler_UploadPart
