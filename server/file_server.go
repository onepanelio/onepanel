package server

import (
	"context"
	api "github.com/onepanelio/core/api/gen"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/server/auth"
	"math"
	"sort"
	"strings"
	"time"
)

// FileServer is an implementation of the grpc WorkflowServer
type FileServer struct {
	api.UnimplementedFileServiceServer
}

// NewFileServer creates a new FileServer
func NewFileServer() *FileServer {
	return &FileServer{}
}

// ListFiles returns a list of files from the configured cloud storage
// Directories come first, and then it is sorted alphabetically
func (s *FileServer) ListFiles(ctx context.Context, req *api.ListFilesRequest) (*api.ListFilesResponse, error) {
	// TODO resource is workflows for now, should it be something else?
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	if req.Page < 0 {
		req.Page = 1
	}
	if req.PerPage <= 0 {
		req.PerPage = 15
	}

	files, err := client.ListFiles(req.Namespace, req.Path)
	if err != nil {
		return nil, err
	}

	apiFiles := make([]*api.File, len(files))
	for i, file := range files {
		apiFiles[i] = &api.File{
			Path:         file.Path,
			Name:         file.Name,
			Extension:    file.Extension,
			Directory:    file.Directory,
			Size:         file.Size,
			ContentType:  file.ContentType,
			LastModified: file.LastModified.UTC().Format(time.RFC3339),
		}
	}

	sort.SliceStable(apiFiles, func(i, j int) bool {
		lhFile := apiFiles[i]
		rhFile := apiFiles[j]

		if (lhFile.Directory && rhFile.Directory) ||
			(!lhFile.Directory && !rhFile.Directory) {
			return strings.Compare(strings.ToLower(lhFile.Name), strings.ToLower(rhFile.Name)) < 0
		}

		if lhFile.Directory {
			return true
		}

		return false
	})

	start := (req.Page - 1) * req.PerPage
	if start < 0 {
		start = 0
	}

	end := int(start + req.PerPage)
	if end > len(apiFiles) {
		end = len(apiFiles)
	}
	parts := apiFiles[start:end]

	count := len(parts)
	totalCount := len(apiFiles)

	return &api.ListFilesResponse{
		Count:      int32(count),
		Page:       req.Page,
		Pages:      int32(math.Ceil(float64(totalCount) / float64(req.PerPage))),
		TotalCount: int32(totalCount),
		Files:      parts,
		ParentPath: v1.FilePathToParentPath(req.Path),
	}, nil
}

// GetObjectDownloadPresignedURL returns a downloadable url for a given object
func (s *FileServer) GetObjectDownloadPresignedURL(ctx context.Context, req *api.GetObjectPresignedUrlRequest) (*api.GetPresignedUrlResponse, error) {
	// TODO resource is workflows for now, should it be something else?
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "argoproj.io", "workflows", "")
	if err != nil || !allowed {
		return nil, err
	}

	downloadObj, err := client.GetObjectPresignedURL(req.Namespace, req.Key)
	if err != nil {
		return nil, err
	}

	return &api.GetPresignedUrlResponse{
		Url:  downloadObj.URL,
		Size: downloadObj.Size,
	}, nil
}
