package gcs

import (
	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"io"
)

/*
	Client is a struct used for accessing Google Cloud Storage.
*/
type Client struct {
	*storage.Client
}

/*
	NewClient handles the details of initializing the connection to Google Cloud Storage.
	- Note that the permissions are set to ReadWrite.
*/
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

/* GetObject retrieves a specific object from Google Cloud Storage.
- Function Name is meant to be consistent with S3's.
*/
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
