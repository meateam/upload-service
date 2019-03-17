package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/aws/aws-sdk-go/service/s3"
)

func TestUploadService_UploadFile(t *testing.T) {
	type fields struct {
		s3Client *s3.S3
	}
	type args struct {
		file   io.Reader
		key    *string
		bucket *string
	}

	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	s3SecretKey := os.Getenv("S3_SECRET_KEY")
	s3Endpoint := os.Getenv("S3_ENDPOINT")
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
	s3Client := s3.New(newSession)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *string
		wantErr bool
	}{
		{
			name:   "Generate text file upload",
			fields: fields{s3Client: s3Client},
			args: args{
				key:    aws.String("testfile.txt"),
				bucket: aws.String("testbucket"),
				file:   bytes.NewReader([]byte("Hello, World!")),
			},
			wantErr: false,
			want:    aws.String(fmt.Sprintf("%s/testbucket/testfile.txt", s3Endpoint)),
		},
		{
			name:   "Generated text file upload in a folder",
			fields: fields{s3Client: s3Client},
			args: args{
				key:    aws.String("testfolder/testfile.txt"),
				bucket: aws.String("testbucket"),
				file:   bytes.NewReader([]byte("Hello, World!")),
			},
			wantErr: false,
			want:    aws.String(fmt.Sprintf("%s/testbucket/testfolder/testfile.txt", s3Endpoint)),
		},
		{
			name:   "Upload file with empty key",
			fields: fields{s3Client: s3Client},
			args: args{
				key:    aws.String(""),
				bucket: aws.String("testbucket"),
				file:   bytes.NewReader([]byte("Hello, World!")),
			},
			wantErr: true,
		},
		{
			name:   "Upload file with empty bucket",
			fields: fields{s3Client: s3Client},
			args: args{
				key:    aws.String("testfile.txt"),
				bucket: aws.String(""),
				file:   bytes.NewReader([]byte("Hello, World!")),
			},
			wantErr: true,
		},
		{
			name:   "Upload file with nil key",
			fields: fields{s3Client: s3Client},
			args: args{
				key:    nil,
				bucket: aws.String("testbucket"),
				file:   bytes.NewReader([]byte("Hello, World!")),
			},
			wantErr: true,
		},
		{
			name:   "Upload file with nil bucket",
			fields: fields{s3Client: s3Client},
			args: args{
				key:    aws.String("testfile.txt"),
				bucket: nil,
				file:   bytes.NewReader([]byte("Hello, World!")),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := UploadService{
				s3Client: tt.fields.s3Client,
			}

			got, err := s.UploadFile(tt.args.file, tt.args.key, tt.args.bucket)
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
