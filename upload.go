package main

import (
	"io"
	"fmt"
	"net/http"

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
	BucketService
}

// Upload is the request handler for file uploads, it is responsible for getting the file
// from the request's body and uploading it to the bucket of the user who uploaded it
func (h UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20) // 32 MB
	file, handler, err := r.FormFile("uploadfile")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	defer file.Close()
	defer r.MultipartForm.RemoveAll()

	// TODO: Should be getting bucket name based on the user who's uploading this file
	bucket := aws.String("testbucket")
	key := aws.String(handler.Filename)

	// Create the bucket to upload to if it doesn't exist
	bucketExists := h.BucketExists(bucket)

	if bucketExists == false {
		bucketExists, err := h.CreateBucket(bucket)
		if err != nil {
			http.Error(w, fmt.Errorf("failed to upload file to %s/%s: %v", *bucket, *key, err).Error(), 500)
			return
		}

		if bucketExists == false {
			http.Error(w, fmt.Errorf("failed to upload file to %s/%s: bucket %s does not exist", *bucket, *key, *bucket).Error(), 500)
			return
		}
	}

	output, err := h.UploadFile(file, key, bucket)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	w.Write([]byte(*output))
}
