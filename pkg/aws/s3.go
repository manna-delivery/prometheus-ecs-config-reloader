package aws

import (
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"

	"log"
)

// DownloadObject - Get object value from S3 as string
func DownloadObject(bucket, item string) string {
	s3Service := s3.New(sharedSession)

	result, err := s3Service.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(item),
	})
	if err != nil {
		log.Fatalf("Unable to download item %q/%q, %v", bucket, item, err)
	}
	defer result.Body.Close()
	valueBytes, err := ioutil.ReadAll(result.Body)
	if err != nil {
		log.Fatalf("Unable to read item %q/%q, %v", bucket, item, err)
	}

	return string(valueBytes)
}
