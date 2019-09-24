package object

import (
	"fmt"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/meateam/upload-service/bucket"
)

// Service is a structure used for operations on S3 objects.
type Service struct {
	s3Client *s3.S3
	mu       sync.Mutex
}

// NewService creates a Service and returns it.
func NewService(s3Client *s3.S3) *Service {
	return &Service{s3Client: s3Client}
}

// GetS3Client returns the internal s3 client.
func (s *Service) GetS3Client() *s3.S3 {
	return s.s3Client
}

// ensureBucketExists Creates a bucket if it doesn't exist.
func (s *Service) ensureBucketExists(ctx aws.Context, bucketName *string) error {
	if ctx == nil {
		return fmt.Errorf("context is required")
	}
	if bucketName == nil {
		return fmt.Errorf("bucketName is required")
	}

	bucketService := bucket.NewService(s.GetS3Client())
	s.mu.Lock()
	defer s.mu.Unlock()
	bucketExists := bucketService.BucketExists(ctx, bucketName)

	if !bucketExists {
		bucketExists, err := bucketService.CreateBucket(ctx, bucketName)
		if err != nil {
			return fmt.Errorf("failed to create bucket %s: %v", *bucketName, err)
		}

		if !bucketExists {
			return fmt.Errorf("failed to create bucket %s: bucket does not exist", *bucketName)
		}
	}
	*bucketName = bucketService.NormalizeCephBucketName(*bucketName)

	return nil
}

// UploadFile uploads a file to the given bucket and key in S3.
// If metadata is a non-nil map then it will be uploaded with the file.
// Returns the file's location and an error if any occurred.
func (s *Service) UploadFile(
	ctx aws.Context,
	file io.Reader,
	key *string,
	bucket *string,
	contentType *string,
	metadata map[string]*string,
) (*string, error) {
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

	err := s.ensureBucketExists(ctx, bucket)
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
func (s *Service) UploadInit(
	ctx aws.Context,
	key *string,
	bucket *string,
	contentType *string,
	metadata map[string]*string,
) (*s3.CreateMultipartUploadOutput, error) {
	if key == nil || *key == "" {
		return nil, fmt.Errorf("key is required")
	}

	if bucket == nil || *bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}

	err := s.ensureBucketExists(ctx, bucket)
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
func (s *Service) UploadPart(
	ctx aws.Context,
	uploadID *string,
	key *string,
	bucket *string,
	partNumber *int64,
	body io.ReadSeeker,
) (*s3.UploadPartOutput, error) {
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

	err := s.ensureBucketExists(ctx, bucket)
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
func (s *Service) ListUploadParts(
	ctx aws.Context,
	uploadID *string,
	key *string,
	bucket *string,
) (*s3.ListPartsOutput, error) {
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

	err := s.ensureBucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to list upload %s parts at %s/%s: %v", *uploadID, *bucket, *key, err)
	}

	listPartsInput := &s3.ListPartsInput{
		UploadId: uploadID,
		Key:      key,
		Bucket:   bucket,
		MaxParts: aws.Int64(10000),
	}

	parts, err := s.s3Client.ListPartsWithContext(ctx, listPartsInput)
	if err != nil {
		return nil, err
	}

	return parts, nil
}

// UploadComplete completes a multipart upload by assembling previously uploaded parts
// associated with uploadID.
func (s *Service) UploadComplete(
	ctx aws.Context,
	uploadID *string,
	key *string,
	bucket *string,
) (*s3.CompleteMultipartUploadOutput, error) {
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

	err := s.ensureBucketExists(ctx, bucket)
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
func (s *Service) HeadObject(ctx aws.Context, key *string, bucket *string) (*s3.HeadObjectOutput, error) {
	if key == nil || *key == "" {
		return nil, fmt.Errorf("key is required")
	}

	if bucket == nil || *bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}
	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}

	err := s.ensureBucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to HeadObject %s/%s: %v", *bucket, *key, err)
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
func (s *Service) UploadAbort(ctx aws.Context, uploadID *string, key *string, bucket *string) (bool, error) {
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

	err := s.ensureBucketExists(ctx, bucket)
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

// DeleteObjects repeated string  deletes an object from s3,
// It receives a bucket and a slice of *strings to be deleted
// and returns the deleted and errored objects or an error if exists.
func (s *Service) DeleteObjects(ctx aws.Context, bucket *string, keys []*string) (*s3.DeleteObjectsOutput, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}

	if bucket == nil || *bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	if keys == nil || len(keys) <= 0 {
		return nil, fmt.Errorf("keys are required")
	}

	err := s.ensureBucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to DeleteObjects bucket, %s, does not exist: %v", *bucket, err)
	}

	objects := make([]*s3.ObjectIdentifier, 0, len(keys))

	for _, key := range keys {
		objects = append(objects, &s3.ObjectIdentifier{
			Key: key,
		})
	}

	deleteObjectsInput := &s3.DeleteObjectsInput{
		Bucket: bucket,
		Delete: &s3.Delete{
			Objects: objects,
			Quiet:   aws.Bool(false),
		},
	}

	deleteResponse, err := s.s3Client.DeleteObjectsWithContext(ctx, deleteObjectsInput)
	if err != nil {
		return nil, fmt.Errorf("failed to delete objects: %v", err)
	}
	return deleteResponse, nil
}
