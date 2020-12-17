package server

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/onepanelio/core/api/gen"
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

// GetConfig returns the system configuration options
func (c *ConfigServer) GetConfig(ctx context.Context, req *empty.Empty) (*api.GetConfigResponse, error) {
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

	return &api.GetConfigResponse{
		ApiUrl:   sysConfig["ONEPANEL_API_URL"],
		Domain:   sysConfig["ONEPANEL_DOMAIN"],
		Fqdn:     sysConfig["ONEPANEL_FQDN"],
		NodePool: nodePool,
	}, err
}
