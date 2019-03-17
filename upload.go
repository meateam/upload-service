package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	pb "upload-service/proto"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// UploadService is a structure used for uploading files to S3
type UploadService struct {
	s3Client *s3.S3
}

// UploadFile uploads a file to the given bucket in S3
func (s UploadService) UploadFile(file io.Reader, key *string, bucket *string) (*string, error) {
	if key == nil || *key == "" {
		return nil, fmt.Errorf("key is required")
	}

	if bucket == nil || *bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	bucketService := BucketService{s3Client: s.s3Client}
	bucketExists := bucketService.BucketExists(bucket)

	if bucketExists == false {
		bucketExists, err := bucketService.CreateBucket(bucket)
		if err != nil {
			return nil, fmt.Errorf("failed to upload file to %s/%s: %v", *bucket, *key, err)
		}

		if bucketExists == false {
			return nil, fmt.Errorf("failed to upload file to %s/%s: bucket %s does not exist", *bucket, *key, *bucket)
		}
	}

	// Create an uploader with S3 client and custom options
	uploader := s3manager.NewUploaderWithClient(s.s3Client, func(u *s3manager.Uploader) {
		u.PartSize = 32 * 1024 * 1024 // 32MB per part
	})

	// Upload a new object with the file's data to the user's bucket
	output, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: bucket,
		Key:    key,
		Body:   file,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to upload data to %s/%s: %v", *bucket, *key, err)
	}

	return &output.Location, nil
}

// UploadHandler handles upload requests by uploading the file's data to aws-s3 Object Storage
type UploadHandler struct {
	UploadService
}

// Upload is the request handler for file uploads, it is responsible for getting the file
// from the request's body and uploading it to the bucket of the user who uploaded it
func (h UploadHandler) Upload(ctx context.Context, request *pb.UploadRequest) (*pb.UploadResponse, error) {
	bucket := aws.String(request.Bucket)
	key := aws.String(request.Key)
	file := bytes.NewReader(request.File)
	output, err := h.UploadFile(file, key, bucket)

	if err != nil {
		return nil, err
	}

	return &pb.UploadResponse{Output: *output}, nil
}
