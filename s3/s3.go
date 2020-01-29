package s3

import (
	"io"
	"strconv"

	minio "github.com/minio/minio-go/v6"
	"github.com/onepanelio/core/util/env"
)

var objectRange = env.GetEnv("ARTIFACT_RERPOSITORY_OBJECT_RANGE", "-102400")

type Client struct {
	*minio.Client
}

type Config struct {
	AccessKey string
	SecretKey string
	Endpoint  string
	Region    string
	InSecure  bool
}

func NewClient(config Config) (s3Client *Client, err error) {
	var minioClient *minio.Client
	if config.Region != "" {
		minioClient, err = minio.NewWithRegion(config.Endpoint, config.AccessKey, config.SecretKey, !config.InSecure, config.Region)
	} else {
		minioClient, err = minio.New(config.Endpoint, config.AccessKey, config.SecretKey, !config.InSecure)
	}
	if err != nil {
		return
	}
	return &Client{Client: minioClient}, nil
}

func (c *Client) GetObject(bucket, key string) (stream io.ReadCloser, err error) {
	opts := minio.GetObjectOptions{}
	end, err := strconv.Atoi(objectRange)
	if err != nil {
		return
	}
	opts.SetRange(0, int64(end))
	stream, err = c.Client.GetObject(bucket, key, opts)
	if err != nil {
		return
	}
	if stream == nil {
		defer stream.Close()
	}

	return
}
