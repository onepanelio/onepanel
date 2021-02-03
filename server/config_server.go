package server

import (
	"context"
	"fmt"
	api "github.com/onepanelio/core/api/gen"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/server/auth"
)

// ConfigServer contains actions for system configuration related items
type ConfigServer struct {
	api.UnimplementedConfigServiceServer
}

// NewConfigServer creates a new ConfigServer
func NewConfigServer() *ConfigServer {
	return &ConfigServer{}
}

func getArtifactRepositoryBucket(client *v1.Client, namespace string) (string, error) {
	if namespace == "" {
		return "", nil
	}

	namespaceConfig, err := client.GetNamespaceConfig(namespace)
	if err != nil {
		return "", err
	}

	if namespaceConfig.ArtifactRepository.S3 != nil {
		return namespaceConfig.ArtifactRepository.S3.Bucket, nil
	}

	return "", fmt.Errorf("unknown artifact repository")
}

// GetConfig returns the system configuration options
func (c *ConfigServer) GetConfig(ctx context.Context, req *api.GetConfigRequest) (*api.GetConfigResponse, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, "", "list", "", "namespaces", "")
	if err != nil || !allowed {
		return nil, err
	}

	sysConfig, err := client.GetSystemConfig()
	if err != nil {
		return nil, err
	}

	nodePool := &api.NodePool{
		Label:   *sysConfig.GetValue("applicationNodePoolLabel"),
		Options: make([]*api.NodePoolOption, 0),
	}

	nodePoolOptions, err := sysConfig.NodePoolOptions()
	if err != nil {
		return nil, err
	}
	type ConfigServer struct{}
	for _, option := range nodePoolOptions {
		nodePool.Options = append(nodePool.Options, &api.NodePoolOption{
			Name:  option.Name,
			Value: option.Value,
		})
	}

	bucket, err := getArtifactRepositoryBucket(client, req.Namespace)
	if err != nil {
		return nil, err
	}

	return &api.GetConfigResponse{
		ApiUrl:   sysConfig["ONEPANEL_API_URL"],
		Domain:   sysConfig["ONEPANEL_DOMAIN"],
		Fqdn:     sysConfig["ONEPANEL_FQDN"],
		NodePool: nodePool,
		Bucket:   bucket,
	}, err
}
