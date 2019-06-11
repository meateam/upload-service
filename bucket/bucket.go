package bucket

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Service is a structure used for bucket operations on S3
type Service struct {
	s3Client *s3.S3
}

// NewService creates a Service and returns it.
func NewService(s3Client *s3.S3) *Service {
	return &Service{s3Client: s3Client}
}

// BucketExists returns true if a bucket exists and the S3Client has permission
// to access it, false otherwise.
func (s Service) BucketExists(ctx aws.Context, bucket *string) bool {
	input := &s3.HeadBucketInput{
		Bucket: bucket,
	}

	_, err := s.s3Client.HeadBucketWithContext(ctx, input)

	return (err == nil)
}

// CreateBucket creates a bucket with the given bucket name and returns true or false
// if it's created or not, returns an error if it didn't.
func (s Service) CreateBucket(ctx aws.Context, bucket *string) (bool, error) {
	cparams := &s3.CreateBucketInput{
		Bucket: bucket, // Required
	}

	// Create a new bucket using the CreateBucket call.
	_, err := s.s3Client.CreateBucketWithContext(ctx, cparams)
	if err != nil {
		return false, fmt.Errorf("failed to create bucket: %v", err)
	}

	return true, nil
}
