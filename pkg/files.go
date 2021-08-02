package v1

import (
	"fmt"
	"github.com/minio/minio-go/v6"
	"github.com/onepanelio/core/pkg/util"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"net/url"
	"strings"
	"time"
)

// GetPresignedURLDownload represents the information available when downloading an object
type GetPresignedURLDownload struct {
	URL  string
	Size int64
}

// ListFiles returns an array of files for the given namespace/key
func (c *Client) ListFiles(namespace, key string) (files []*File, err error) {
	config, err := c.GetNamespaceConfig(namespace)
	if err != nil {
		return
	}

	if config.ArtifactRepository.S3 == nil {
		return nil, util.NewUserError(codes.Internal, "S3 compatible artifact repository not set")
	}

	files = make([]*File, 0)

	if len(key) > 0 && strings.HasPrefix(key, "/") {
		key = key[1:]
	}

	if len(key) > 0 {
		if string(key[len(key)-1]) != "/" {
			key += "/"
		}
	}

	s3Client, err := c.GetS3Client(namespace, config.ArtifactRepository.S3)
	if err != nil {
		return nil, err
	}

	doneCh := make(chan struct{})
	defer close(doneCh)
	for objInfo := range s3Client.ListObjects(config.ArtifactRepository.S3.Bucket, key, false, doneCh) {
		if objInfo.Key == key {
			continue
		}

		isDirectory := (objInfo.ETag == "" || strings.HasSuffix(objInfo.Key, "/")) && objInfo.Size == 0

		newFile := &File{
			Path:         objInfo.Key,
			Name:         FilePathToName(objInfo.Key),
			Extension:    FilePathToExtension(objInfo.Key),
			Size:         objInfo.Size,
			LastModified: objInfo.LastModified,
			ContentType:  objInfo.ContentType,
			Directory:    isDirectory,
		}
		files = append(files, newFile)
	}

	return
}

// GetObjectPresignedURL generates a presigned url for the object that is valid for 24 hours.
func (c *Client) GetObjectPresignedURL(namespace, key string) (download *GetPresignedURLDownload, err error) {
	config, err := c.GetNamespaceConfig(namespace)
	if err != nil {
		return
	}

	s3Client, err := c.GetPublicS3Client(namespace, config.ArtifactRepository.S3)
	if err != nil {
		return
	}

	objInfo, err := s3Client.StatObject(config.ArtifactRepository.S3.Bucket, key, minio.StatObjectOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Key":       key,
			"Error":     err.Error(),
		}).Error("StatObject")
		return
	}

	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", fmt.Sprintf("attachment; filename=\"%s\"", key))
	presignedURL, err := s3Client.PresignedGetObject(config.ArtifactRepository.S3.Bucket, key, time.Hour*24, reqParams)
	if err != nil {
		log.WithFields(log.Fields{
			"Namespace": namespace,
			"Key":       key,
			"Error":     err.Error(),
		}).Error("PresignedGetObject")
		return
	}

	return &GetPresignedURLDownload{
		URL:  presignedURL.String(),
		Size: objInfo.Size,
	}, nil
}
