package main

import (
	"sync"
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	pb "github.com/meateam/upload-service/proto"
)

// UploadService is a structure used for uploading files to S3
type UploadService struct {
	s3Client *s3.S3
	mu			 sync.Mutex
}

// EnsureBucketExists Creates a bucket if it doesn't exist.
func (s *UploadService) EnsureBucketExists(ctx aws.Context, bucket *string) error {
	if ctx == nil {
		return fmt.Errorf("context is required")
	}

	bucketService := BucketService{s3Client: s.s3Client}
	s.mu.Lock()
	defer s.mu.Unlock()
	bucketExists := bucketService.BucketExists(ctx, bucket)

	if bucketExists == false {
		bucketExists, err := bucketService.CreateBucket(ctx, bucket)
		if err != nil {
			return fmt.Errorf("failed to create bucket %s: %v", *bucket, err)
		}

		if bucketExists == false {
			return fmt.Errorf("failed to create bucket %s: bucket does not exist", *bucket)
		}
	}

	return nil
}

// UploadFile uploads a file to the given bucket and key in S3.
// If metadata is a non-nil map then it will be uploaded with the file.
// Returns the file's location and an error if any occured.
func (s *UploadService) UploadFile(ctx aws.Context, file io.Reader, key *string, bucket *string, contentType *string, metadata map[string]*string) (*string, error) {
	if file == nil {
		return nil, fmt.Errorf("file is required")
	}

	if key == nil || *key == "" {
		return nil, fmt.Errorf("key is required")
	}

	if bucket == nil || *bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}

	err := s.EnsureBucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file to %s/%s: %v", *bucket, *key, err)
	}

	// Create an uploader with S3 client and custom options
	uploader := s3manager.NewUploaderWithClient(s.s3Client, func(u *s3manager.Uploader) {
		u.PartSize = 32 * 1024 * 1024 // 32MB per part
	})

	input := &s3manager.UploadInput{
		Bucket:      bucket,
		Key:         key,
		Body:        file,
		ContentType: contentType,
	}

	if metadata != nil {
		input.Metadata = metadata
	}


	// Upload a new object with the file's data to the user's bucket
	output, err := uploader.UploadWithContext(ctx, input)

	if err != nil {
		return nil, fmt.Errorf("failed to upload data to %s/%s: %v", *bucket, *key, err)
	}

	return &output.Location, nil
}

// UploadInit initiates a multipart upload to the given bucket and key in S3 with metadata.
// File metadata is required for multipart upload.
func (s *UploadService) UploadInit(ctx aws.Context, key *string, bucket *string, contentType *string, metadata map[string]*string) (*s3.CreateMultipartUploadOutput, error) {
	if key == nil || *key == "" {
		return nil, fmt.Errorf("key is required")
	}

	if bucket == nil || *bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}

	err := s.EnsureBucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to init upload to %s/%s: %v", *bucket, *key, err)
	}

	input := &s3.CreateMultipartUploadInput{
		Bucket:      bucket,
		Key:         key,
		Metadata:    metadata,
		ContentType: contentType,
	}

	result, err := s.s3Client.CreateMultipartUploadWithContext(ctx, input)
	if err != nil {
		return nil, err
	}

	return result, err
}

// UploadPart uploads a part in a multipart upload of a file.
func (s *UploadService) UploadPart(ctx aws.Context, uploadID *string, key *string, bucket *string, partNumber *int64, body io.ReadSeeker) (*s3.UploadPartOutput, error) {
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

	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}

	err := s.EnsureBucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to upload part to %s/%s: %v", *bucket, *key, err)
	}

	input := &s3.UploadPartInput{
		Body:       body,
		Bucket:     bucket,
		Key:        key,
		PartNumber: partNumber,
		UploadId:   uploadID,
	}

	result, err := s.s3Client.UploadPartWithContext(ctx, input)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ListUploadParts lists the uploaded file parts of a multipart upload of a file.
func (s *UploadService) ListUploadParts(ctx aws.Context, uploadID *string, key *string, bucket *string) (*s3.ListPartsOutput, error) {
	if key == nil || *key == "" {
		return nil, fmt.Errorf("key is required")
	}

	if bucket == nil || *bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	if uploadID == nil || *uploadID == "" {
		return nil, fmt.Errorf("upload id is required")
	}

	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}

	err := s.EnsureBucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to list upload %s parts at %s/%s: %v", *uploadID, *bucket, *key, err)
	}

	listPartsInput := &s3.ListPartsInput{
		UploadId: uploadID,
		Key:      key,
		Bucket:   bucket,
	}

	parts, err := s.s3Client.ListPartsWithContext(ctx, listPartsInput)
	if err != nil {
		return nil, err
	}

	return parts, nil
}

// UploadComplete completes a multipart upload by assembling previously uploaded parts
// assosiated with uploadID.
func (s *UploadService) UploadComplete(ctx aws.Context, uploadID *string, key *string, bucket *string) (*s3.CompleteMultipartUploadOutput, error) {
	if key == nil || *key == "" {
		return nil, fmt.Errorf("key is required")
	}

	if bucket == nil || *bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	if uploadID == nil || *uploadID == "" {
		return nil, fmt.Errorf("upload id is required")
	}

	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}

	err := s.EnsureBucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to upload complete %s parts at %s/%s: %v", *uploadID, *bucket, *key, err)
	}

	parts, err := s.ListUploadParts(ctx, uploadID, key, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed listing upload parts")
	}

	completedMultipartParts := make([]*s3.CompletedPart, 0, len(parts.Parts))

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

	result, err := s.s3Client.CompleteMultipartUploadWithContext(ctx, input)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// HeadObject returns object's details.
func (s *UploadService) HeadObject(ctx aws.Context, key *string, bucket *string) (*s3.HeadObjectOutput, error) {
	if key == nil || *key == "" {
		return nil, fmt.Errorf("key is required")
	}

	if bucket == nil || *bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}
	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}

	obj, err := s.s3Client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{Bucket: bucket, Key: key})
	if err != nil {
		return nil, fmt.Errorf("failed to head object")
	}
	return obj, nil
}

// UploadAbort aborts a multipart upload. After a multipart upload is aborted, no additional parts
// can be uploaded using that upload ID. The storage consumed by any previously uploaded parts will be freed.
// However, if any part uploads are currently in progress, those part uploads might or might not succeed.
// As a result, it might be necessary to abort a given multipart upload multiple times in order to
// completely free all storage consumed by all parts. To verify that all parts have been removed,
// so you don't get charged for the part storage, you should call
// the List Parts operation and ensure the parts list is empty.
func (s *UploadService) UploadAbort(ctx aws.Context, uploadID *string, key *string, bucket *string) (bool, error) {
	if key == nil || *key == "" {
		return false, fmt.Errorf("key is required")
	}

	if bucket == nil || *bucket == "" {
		return false, fmt.Errorf("bucket name is required")
	}

	if uploadID == nil || *uploadID == "" {
		return false, fmt.Errorf("upload id is required")
	}

	if ctx == nil {
		return false, fmt.Errorf("context is required")
	}

	err := s.EnsureBucketExists(ctx, bucket)
	if err != nil {
		return false, fmt.Errorf("failed to list upload %s parts at %s/%s: %v", *uploadID, *bucket, *key, err)
	}

	abortInput := &s3.AbortMultipartUploadInput{
		Bucket:   bucket,
		Key:      key,
		UploadId: uploadID,
	}

	_, err = s.s3Client.AbortMultipartUploadWithContext(ctx, abortInput)
	if err != nil {
		return false, fmt.Errorf("failed aborting multipart upload: %v", err)
	}

	return true, nil
}

// UploadHandler handles upload requests by uploading the file's data to aws-s3 Object Storage
type UploadHandler struct {
	*UploadService
}

// UploadMedia is the request handler for file upload, it is responsible for getting the file
// from the request's body and uploading it to the bucket of the user who uploaded it
func (h UploadHandler) UploadMedia(ctx context.Context, request *pb.UploadMediaRequest) (*pb.UploadMediaResponse, error) {
	location, err := h.UploadFile(ctx,
		bytes.NewReader(request.GetFile()),
		aws.String(request.GetKey()),
		aws.String(request.GetBucket()),
		aws.String(request.GetContentType()),
		nil)

	if err != nil {
		return nil, err
	}

	return &pb.UploadMediaResponse{Location: *location}, nil
}

// UploadMultipart is the request handler for file upload, it is responsible for getting the file
// from the request's body and uploading it to the bucket of the user who uploaded it
func (h UploadHandler) UploadMultipart(ctx context.Context, request *pb.UploadMultipartRequest) (*pb.UploadMultipartResponse, error) {
	metadata := request.GetMetadata()
	if len(metadata) == 0 {
		return nil, fmt.Errorf("metadata is required")
	}

	location, err := h.UploadFile(ctx,
		bytes.NewReader(request.GetFile()),
		aws.String(request.GetKey()),
		aws.String(request.GetBucket()),
		aws.String(request.GetContentType()),
		aws.StringMap(request.GetMetadata()))

	if err != nil {
		return nil, err
	}

	return &pb.UploadMultipartResponse{Location: *location}, nil
}

// UploadInit is the request handler for initiating resumable upload.
func (h UploadHandler) UploadInit(ctx context.Context, request *pb.UploadInitRequest) (*pb.UploadInitResponse, error) {
	result, err := h.UploadService.UploadInit(
		ctx,
		aws.String(request.GetKey()),
		aws.String(request.GetBucket()),
		aws.String(request.GetContentType()),
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

// UploadPart is the request handler for multipart file upload.
// It is fetching file parts from a RPC stream and uploads them concurrently.
// Responds with a stream of upload status for each part streamed.
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

			result, err := h.UploadService.UploadPart(
				stream.Context(),
				aws.String(part.GetUploadId()),
				aws.String(part.GetKey()),
				aws.String(part.GetBucket()),
				aws.Int64(part.GetPartNumber()),
				bytes.NewReader(part.GetPart()))

			var resp *pb.UploadPartResponse

			if err != nil {
				resp = &pb.UploadPartResponse{
					Code:    500,
					Message: fmt.Sprintf("failed uploading part: %v", err),
				}
			} else {
				resp = &pb.UploadPartResponse{
					Code:    200,
					Message: fmt.Sprintf("successfully uploaded part %s", *result.ETag),
				}
			}

			if err := stream.Send(resp); err != nil {
				return err
			}

			return nil
		}()
	}
}

// UploadComplete is the request handler for completing and assembling previously uploaded file parts.
// Responds with the location of the assembled file.
func (h UploadHandler) UploadComplete(ctx context.Context, request *pb.UploadCompleteRequest) (*pb.UploadCompleteResponse, error) {
	_, err := h.UploadService.UploadComplete(ctx,
		aws.String(request.GetUploadId()),
		aws.String(request.GetKey()),
		aws.String(request.GetBucket()))
	if err != nil {
		return nil, err
	}

	obj, err := h.UploadService.HeadObject(ctx, aws.String(request.GetKey()), aws.String(request.GetBucket()))
	if err != nil {
		return nil, err
	}

	return &pb.UploadCompleteResponse{ContentLength: *obj.ContentLength, ContentType: *obj.ContentType}, nil
}

// UploadAbort is the request handler for aborting and freeing previously uploaded parts.
func (h UploadHandler) UploadAbort(ctx context.Context, request *pb.UploadAbortRequest) (*pb.UploadAbortResponse, error) {
	abortStatus, err := h.UploadService.UploadAbort(
		ctx,
		aws.String(request.GetUploadId()),
		aws.String(request.GetKey()),
		aws.String(request.GetBucket()))

	if err != nil {
		return nil, err
	}

	return &pb.UploadAbortResponse{Status: abortStatus}, nil
}
