package object

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/sirupsen/logrus"

	pb "github.com/meateam/upload-service/proto"
)

// Handler handles object operation requests by uploading the file's data to aws-s3 Object Storage.
type Handler struct {
	service *Service
	logger  *logrus.Logger
}

// NewHandler creates a Handler and returns it.
func NewHandler(service *Service, logger *logrus.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// GetService returns the internal upload service.
func (h Handler) GetService() *Service {
	return h.service
}

// UploadMedia is the request handler for file upload, it is responsible for getting the file
// from the request's body and uploading it to the bucket of the user who uploaded it
func (h Handler) UploadMedia(
	ctx context.Context,
	request *pb.UploadMediaRequest,
) (*pb.UploadMediaResponse, error) {
	location, err := h.service.UploadFile(ctx,
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
func (h Handler) UploadMultipart(
	ctx context.Context,
	request *pb.UploadMultipartRequest,
) (*pb.UploadMultipartResponse, error) {
	metadata := request.GetMetadata()
	if len(metadata) == 0 {
		return nil, fmt.Errorf("metadata is required")
	}

	location, err := h.service.UploadFile(ctx,
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
func (h Handler) UploadInit(
	ctx context.Context,
	request *pb.UploadInitRequest,
) (*pb.UploadInitResponse, error) {
	result, err := h.service.UploadInit(
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
func (h Handler) UploadPart(stream pb.Upload_UploadPartServer) error {
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
		go func() {
			defer wg.Done()

			result, err := h.service.UploadPart(
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
				h.logger.Errorf("failed to send response in stream:  %v", err)
			}
		}()
	}
}

// UploadComplete is the request handler for completing and assembling previously uploaded file parts.
// Responds with the location of the assembled file.
func (h Handler) UploadComplete(
	ctx context.Context,
	request *pb.UploadCompleteRequest,
) (*pb.UploadCompleteResponse, error) {
	_, err := h.service.UploadComplete(ctx,
		aws.String(request.GetUploadId()),
		aws.String(request.GetKey()),
		aws.String(request.GetBucket()))
	if err != nil {
		return nil, err
	}

	obj, err := h.service.HeadObject(ctx, aws.String(request.GetKey()), aws.String(request.GetBucket()))
	if err != nil {
		return nil, err
	}

	return &pb.UploadCompleteResponse{ContentLength: *obj.ContentLength, ContentType: *obj.ContentType}, nil
}

// UploadAbort is the request handler for aborting and freeing previously uploaded parts.
func (h Handler) UploadAbort(
	ctx context.Context,
	request *pb.UploadAbortRequest,
) (*pb.UploadAbortResponse, error) {
	abortStatus, err := h.service.UploadAbort(
		ctx,
		aws.String(request.GetUploadId()),
		aws.String(request.GetKey()),
		aws.String(request.GetBucket()))

	if err != nil {
		return nil, err
	}

	return &pb.UploadAbortResponse{Status: abortStatus}, nil
}

// DeleteObjects is the request handler for deleting objects.
// It responds with a slice of deleted object keys and a slice of failed object keys.
func (h Handler) DeleteObjects(
	ctx context.Context,
	request *pb.DeleteObjectsRequest,
) (*pb.DeleteObjectsResponse, error) {
	deleteResponse, err := h.service.DeleteObjects(
		ctx,
		aws.String(request.GetBucket()),
		aws.StringSlice(request.GetKeys()),
	)
	if err != nil {
		return nil, err
	}

	deletedKeys := make([]string, 0, len(deleteResponse.Deleted))
	for _, deletedObject := range deleteResponse.Deleted {
		deletedKeys = append(deletedKeys, *(deletedObject.Key))
	}

	failedKeys := make([]string, 0, len(deleteResponse.Errors))
	for _, erroredObject := range deleteResponse.Errors {
		failedKeys = append(failedKeys, *(erroredObject.Key))
	}

	return &pb.DeleteObjectsResponse{Deleted: deletedKeys, Failed: failedKeys}, nil
}
