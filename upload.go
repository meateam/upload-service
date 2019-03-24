package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
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
// Returns the file's location and an error if any occured.
func (s UploadService) UploadFile(file io.Reader, key *string, bucket *string, metadata map[string]*string) (*string, error) {
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

// UploadInit ...
func (s UploadService) UploadInit(key *string, bucket *string, metadata map[string]*string) (*s3.CreateMultipartUploadOutput, error) {
	if key == nil || *key == "" {
		return nil, fmt.Errorf("key is required")
	}

	if bucket == nil || *bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	input := &s3.CreateMultipartUploadInput{
		Bucket:   bucket,
		Key:      key,
		Metadata: metadata,
	}

	result, err := s.s3Client.CreateMultipartUpload(input)
	if err != nil {
		return nil, err
	}

	return result, err
}

// UploadComplete ...
func (s UploadService) UploadComplete(uploadID *string, key *string, bucket *string) (*s3.CompleteMultipartUploadOutput, error) {
	if key == nil || *key == "" {
		return nil, fmt.Errorf("key is required")
	}

	if bucket == nil || *bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	if uploadID == nil || *uploadID == "" {
		return nil, fmt.Errorf("upload id is required")
	}

	listPartsInput := &s3.ListPartsInput{
		UploadId: uploadID,
		Key:      key,
		Bucket:   bucket,
	}

	parts, err := s.s3Client.ListParts(listPartsInput)
	if err != nil {
		return nil, err
	}

	completedMultipartParts := make([]*s3.CompletedPart, len(parts.Parts))

	for _, v := range parts.Parts {
		completedPart := &s3.CompletedPart{
			ETag:       v.ETag,
			PartNumber: v.PartNumber,
		}

		completedMultipartParts = append(completedMultipartParts, completedPart)
	}

	completedMultipartUpload := &s3.CompletedMultipartUpload{
		Parts: completedMultipartParts,
	}

	input := &s3.CompleteMultipartUploadInput{
		Bucket:          bucket,
		Key:             key,
		MultipartUpload: completedMultipartUpload,
		UploadId:        uploadID,
	}

	result, err := s.s3Client.CompleteMultipartUpload(input)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// UploadPart ...
func (s UploadService) UploadPart(uploadID *string, key *string, bucket *string, partNumber *int64, body io.ReadSeeker) (*s3.UploadPartOutput, error) {
	if body == nil {
		return nil, fmt.Errorf("part body is required")
	}

	if key == nil || *key == "" {
		return nil, fmt.Errorf("key is required")
	}

	if bucket == nil || *bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	if uploadID == nil || *uploadID == "" {
		return nil, fmt.Errorf("upload id is required")
	}

	if partNumber == nil {
		return nil, fmt.Errorf("part number is required")
	}

	if *partNumber < 1 || *partNumber > 10000 {
		return nil, fmt.Errorf("part number must be between 1 and 10,000")
	}

	input := &s3.UploadPartInput{
		Body:       body,
		Bucket:     bucket,
		Key:        key,
		PartNumber: partNumber,
		UploadId:   uploadID,
	}

	result, err := s.s3Client.UploadPart(input)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// UploadHandler handles upload requests by uploading the file's data to aws-s3 Object Storage
type UploadHandler struct {
	UploadService
}

// UploadMedia is the request handler for file uploads, it is responsible for getting the file
// from the request's body and uploading it to the bucket of the user who uploaded it
func (h UploadHandler) UploadMedia(ctx context.Context, request *pb.UploadMediaRequest) (*pb.UploadMediaResponse, error) {
	output, err := h.UploadFile(bytes.NewReader(request.GetFile()),
		aws.String(request.GetKey()),
		aws.String(request.GetBucket()),
		nil)

	if err != nil {
		return nil, err
	}

	return &pb.UploadMediaResponse{Output: *output}, nil
}

// UploadMultipart is the request handler for file uploads, it is responsible for getting the file
// from the request's body and uploading it to the bucket of the user who uploaded it
func (h UploadHandler) UploadMultipart(ctx context.Context, request *pb.UploadMultipartRequest) (*pb.UploadMultipartResponse, error) {
	output, err := h.UploadFile(bytes.NewReader(request.GetFile()),
		aws.String(request.GetKey()),
		aws.String(request.GetBucket()),
		aws.StringMap(request.GetMetadata()))

	if err != nil {
		return nil, err
	}

	return &pb.UploadMultipartResponse{Output: *output}, nil
}

// UploadInit ...
func (h UploadHandler) UploadInit(ctx context.Context, request *pb.UploadInitRequest) (*pb.UploadInitResponse, error) {
	result, err := h.UploadService.UploadInit(aws.String(request.GetKey()),
		aws.String(request.GetBucket()),
		aws.StringMap(request.GetMetadata()))

	if err != nil {
		return nil, err
	}

	response := &pb.UploadInitResponse{
		UploadId: *result.UploadId,
		Key:      *result.Key,
		Bucket:   *result.Bucket,
	}

	return response, nil
}

// UploadPart ...
func (h UploadHandler) UploadPart(stream pb.Upload_UploadPartServer) error {
	wg := sync.WaitGroup{}
	for {
		part, err := stream.Recv()

		if err == io.EOF {
			wg.Wait()
			return nil
		}

		if err != nil {
			errResponse := &pb.UploadPartResponse{Code: 500, Message: fmt.Sprintf("failed fetching part: %v", err)}
			if err := stream.Send(errResponse); err != nil {
				return err
			}
		}

		wg.Add(1)
		go func() error {
			defer wg.Done()

			result, err := h.UploadService.UploadPart(aws.String(part.GetUploadId()),
				aws.String(part.GetKey()),
				aws.String(part.GetBucket()),
				aws.Int64(part.GetPartNumber()),
				bytes.NewReader(part.GetPart()))

			resp := &pb.UploadPartResponse{
				Code:    200,
				Message: fmt.Sprintf("successfully uploaded part %s", *result.ETag),
			}

			if err != nil {
				resp = &pb.UploadPartResponse{
					Code:    500,
					Message: fmt.Sprintf("failed uploading part: %v", err),
				}
			}

			if err := stream.Send(resp); err != nil {
				return err
			}

			return nil
		}()
	}
}

// UploadComplete ...
func (h UploadHandler) UploadComplete(ctx context.Context, request *pb.UploadCompleteRequest) (*pb.UploadCompleteResponse, error) {
	result, err := h.UploadService.UploadComplete(aws.String(request.GetUploadId()), aws.String(request.GetKey()), aws.String(request.GetBucket()))
	if err != nil {
		return nil, err
	}

	return &pb.UploadCompleteResponse{Output: *result.Location}, nil
}
