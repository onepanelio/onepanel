package s3

import (
	"io/ioutil"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Client struct {
	*s3.S3
}

type Object struct {
	Key          *string    `json:"key"`
	LastModified *time.Time `json:"last_modified"`
	Size         *int64     `json:"size"`
	IsPrefix     bool       `json:"is_prefix"`
}

func NewClient(id, secret, region string) (s3Client *Client, err error) {
	session, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(id, secret, ""),
		Region:      aws.String(region),
	})
	if err != nil {
		return nil, err
	}
	return &Client{S3: s3.New(session)}, nil
}

func (c *Client) GetObject(bucket, key string) (content *[]byte, err error) {
	out, err := c.S3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return
	}

	defer out.Body.Close()
	data, err := ioutil.ReadAll(out.Body)
	if err != nil {
		return
	}
	content = &data

	return
}
