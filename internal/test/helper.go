package test

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// EmptyAndDeleteBucket empties the Amazon S3 bucket and deletes it.
func EmptyAndDeleteBucket(s3Client *s3.S3, bucket string) error {
	log.Print("removing objects from S3 bucket : ", bucket)

	params := &s3.ListObjectsInput{
		Bucket:  aws.String(bucket),
		MaxKeys: aws.Int64(10000),
	}

	for {
		// Requesting for batch of objects from s3 bucket
		objects, err := s3Client.ListObjects(params)
		if err != nil {
			break
		}

		// Checks if the bucket is already empty
		if len((*objects).Contents) == 0 {
			log.Print("bucket is already empty")
			break
		}
		log.Print("first object in batch | ", *(objects.Contents[0].Key))

		// Creating an array of pointers of ObjectIdentifier
		objectsToDelete := make([]*s3.ObjectIdentifier, 0, 1000)
		for _, object := range (*objects).Contents {
			obj := s3.ObjectIdentifier{
				Key: object.Key,
			}
			objectsToDelete = append(objectsToDelete, &obj)
		}

		// Creating JSON payload for bulk delete
		deleteArray := s3.Delete{Objects: objectsToDelete}
		deleteParams := &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &deleteArray,
		}

		// Running the Bulk delete job (limit 1000)
		_, err = s3Client.DeleteObjects(deleteParams)
		if err != nil {
			return err
		}
		if *(*objects).IsTruncated { //if there are more objects in the bucket, IsTruncated = true
			params.Marker = (*deleteParams).Delete.Objects[len((*deleteParams).Delete.Objects)-1].Key
			log.Print("requesting next batch | ", *(params.Marker))
		} else { // If all objects in the bucket have been cleaned up.
			break
		}
	}

	log.Print("Emptied S3 bucket : ", bucket)
	if _, err := s3Client.DeleteBucket(&s3.DeleteBucketInput{Bucket: aws.String(bucket)}); err != nil {
		log.Printf("failed to DeleteBucket, %v", err)
		return err
	}

	return nil
}
