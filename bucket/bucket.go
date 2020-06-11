package bucket

import (
	"fmt"
	"regexp"
	"strings"

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
	if bucket == nil {
		return false
	}

	normalizedBucketName := s.NormalizeCephBucketName(*bucket)
	input := &s3.HeadBucketInput{
		Bucket: aws.String(normalizedBucketName),
	}

	_, err := s.s3Client.HeadBucketWithContext(ctx, input)

	return (err == nil)
}

// CreateBucket creates a bucket with the given bucket name and returns true or false
// if it's created or not, returns an error if it didn't.
func (s Service) CreateBucket(ctx aws.Context, bucket *string) (bool, error) {
	if bucket == nil {
		return false, fmt.Errorf("bucket is nil")
	}

	normalizedBucketName := s.NormalizeCephBucketName(*bucket)
	cparams := &s3.CreateBucketInput{
		Bucket: aws.String(normalizedBucketName), // Required
	}

	// Create a new bucket using the CreateBucket call.
	_, err := s.s3Client.CreateBucketWithContext(ctx, cparams)
	if err != nil {
		return false, fmt.Errorf("failed to create bucket: %v", err)
	}

	versioningInput := &s3.PutBucketVersioningInput{
		Bucket: aws.String(normalizedBucketName), // Required
		VersioningConfiguration: &s3.VersioningConfiguration{
			MFADelete: aws.String("Disabled"),
			Status:    aws.String("Enabled"),
		}, // Required
	}

	// Enable versioning mode
	_, err = s.s3Client.PutBucketVersioning(versioningInput)
	if err != nil {
		_, err := s.deleteBucket(normalizedBucketName)
		if err != nil {
			return false, fmt.Errorf("failed to create bucket: %v", err)
		}
		return false, fmt.Errorf("failed to create bucket: %v", err)
	}
	return true, nil
}

// deleteBucket delete a bucket with the given bucket name and returns true or false
// if it's deleted or not, returns an error if it didn't.
func (s Service) deleteBucket(bucket string) (bool, error) {
	input := &s3.DeleteBucketInput{
		Bucket: aws.String(bucket), // Required
	}

	_, err := s.s3Client.DeleteBucket(input)
	if err != nil {
		return false, fmt.Errorf("failed to remove bucket: %v", err)
	}

	return true, nil
}

// NormalizeCephBucketName gets a bucket name and normalizes it
// according to ceph s3's constraints.
func (s Service) NormalizeCephBucketName(bucketName string) string {
	lowerCaseBucketName := strings.ToLower(bucketName)

	// Make a Regex for catching only letters and numbers.
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")
	return reg.ReplaceAllString(lowerCaseBucketName, "-")
}
