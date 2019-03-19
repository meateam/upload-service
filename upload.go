package main

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
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

// UploadFile uploads a file to the given bucket in S3.
// If metadata is a non-nil map then it will be uploaded with the file.
func (s UploadService) UploadFile(file io.Reader, metadata map[string]*string, key *string, bucket *string) (*string, error) {
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

	input := &s3manager.UploadInput{
		Bucket: bucket,
		Key:    key,
		Body:   file,
	}

	if metadata != nil {
		input.Metadata = metadata
	}

	// Upload a new object with the file's data to the user's bucket
	output, err := uploader.Upload(input)

	if err != nil {
		return nil, fmt.Errorf("failed to upload data to %s/%s: %v", *bucket, *key, err)
	}

	return &output.Location, nil
}

func (s UploadService) completeMultipartUpload(resp *s3.CreateMultipartUploadOutput, completedParts []*s3.CompletedPart) (*s3.CompleteMultipartUploadOutput, error) {
	completeInput := &s3.CompleteMultipartUploadInput{
		Bucket:   resp.Bucket,
		Key:      resp.Key,
		UploadId: resp.UploadId,
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: completedParts,
		},
	}
	return s.s3Client.CompleteMultipartUpload(completeInput)
}

func (s UploadService) uploadPart(resp *s3.CreateMultipartUploadOutput, fileBytes []byte, partNumber int) (*s3.CompletedPart, error) {
	tryNum := 1
	partInput := &s3.UploadPartInput{
		Body:          bytes.NewReader(fileBytes),
		Bucket:        resp.Bucket,
		Key:           resp.Key,
		PartNumber:    aws.Int64(int64(partNumber)),
		UploadId:      resp.UploadId,
		ContentLength: aws.Int64(int64(len(fileBytes))),
	}

	for tryNum <= s.s3Client.MaxRetries() {
		uploadResult, err := s.s3Client.UploadPart(partInput)
		if err != nil {
			if tryNum == s.s3Client.MaxRetries() {
				if aerr, ok := err.(awserr.Error); ok {
					return nil, aerr
				}
				return nil, err
			}

			tryNum++
		} else {
			return &s3.CompletedPart{
				ETag:       uploadResult.ETag,
				PartNumber: aws.Int64(int64(partNumber)),
			}, nil
		}
	}
	return nil, nil
}

// UploadHandler handles upload requests by uploading the file's data to aws-s3 Object Storage
type UploadHandler struct {
	UploadService
}

// UploadMedia is the request handler for file uploads, it is responsible for getting the file
// from the request's body and uploading it to the bucket of the user who uploaded it
func (h UploadHandler) UploadMedia(ctx context.Context, request *pb.UploadMediaRequest) (*pb.UploadMediaResponse, error) {
	bucket := aws.String(request.Bucket)
	key := aws.String(request.Key)
	file := bytes.NewReader(request.File)
	output, err := h.UploadFile(file, nil, key, bucket)

	if err != nil {
		return nil, err
	}

	return &pb.UploadMediaResponse{Output: *output}, nil
}

// UploadMultipart is the request handler for file uploads, it is responsible for getting the file
// from the request's body and uploading it to the bucket of the user who uploaded it
func (h UploadHandler) UploadMultipart(ctx context.Context, request *pb.UploadMultipartRequest) (*pb.UploadMultipartResponse, error) {
	bucket := aws.String(request.Bucket)
	key := aws.String(request.Key)
	file := bytes.NewReader(request.File)
	metadata := make(map[string]*string)

	for k, v := range request.Metadata {
		metadata[k] = aws.String(v)
	}

	output, err := h.UploadFile(file, metadata, key, bucket)

	if err != nil {
		return nil, err
	}

	return &pb.UploadMultipartResponse{Output: *output}, nil
}

// UploadResumableInit ...
func (h UploadHandler) UploadResumableInit(context.Context, *pb.UploadResumableInitRequest) (*pb.UploadResumableInitResponse, error) {
	return &pb.UploadResumableInitResponse{UploadId: ""}, nil
}

// UploadResumablePart ...
func (h UploadHandler) UploadResumablePart(context.Context, *pb.UploadResumablePartRequest) (*pb.UploadResumablePartResponse, error) {
	return &pb.UploadResumablePartResponse{Etag: ""}, nil
}
