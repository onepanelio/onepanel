package gcs

import (
	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"io"
)

type Client struct {
	*storage.Client
}

func NewClient(namespace string, serviceAccountJSON string) (gcsClient *Client, err error) {
	ctx := context.Background()
	creds, err := google.CredentialsFromJSON(ctx, []byte(serviceAccountJSON), storage.ScopeReadWrite)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"JSON":      serviceAccountJSON,
			"Error":     err.Error(),
		}).Error("GetGCSClient failed when initializing a new GCS client.")
		return
	}
	client, err := storage.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"JSON":      serviceAccountJSON,
			"Error":     err.Error(),
		}).Error("GetGCSClient failed when initializing a new GCS client.")
		return
	}

	return &Client{Client: client}, nil
}

func (c *Client) GetObject(bucket, key string) (stream io.ReadCloser, err error) {
	ctx := context.Background()
	stream, err = c.Client.Bucket(bucket).Object(key).NewReader(ctx)
	if err != nil {
		return
	}
	if stream == nil {
		defer stream.Close()
	}

	return
}
