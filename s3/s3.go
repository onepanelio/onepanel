package s3

import (
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/onepanelio/core/util/env"
)

var objectRange = env.GetEnv("ARTIFACT_RERPOSITORY_OBJECT_RANGE", "bytes=-102400")

type Client struct {
	*s3.S3
}

type Config struct {
	AccessKey string
	SecretKey string
	Endpoint  string
	Region    string
}

func NewClient(config Config) (s3Client *Client, err error) {
	session, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, ""),
		Region:      aws.String(config.Region),
		Endpoint:    aws.String(config.Endpoint),
	})
	if err != nil {
		return nil, err
	}
	return &Client{S3: s3.New(session)}, nil
}

func (c *Client) GetObject(bucket, key string) (stream io.ReadCloser, err error) {
	out, err := c.S3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Range:  aws.String(objectRange),
	})
	if err != nil {
		return
	}

	stream = out.Body
	if stream == nil {
		defer out.Body.Close()
	}

	return
}
