package server

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/onepanelio/core/api"
	"github.com/onepanelio/core/server/auth"
)

// ConfigServer contains actions for system configuration related items
type ConfigServer struct{}

// NewConfigServer creates a new ConfigServer
func NewConfigServer() *ConfigServer {
	return &ConfigServer{}
}

// GetConfig returns the system configuration options
func (c *ConfigServer) GetConfig(ctx context.Context, req *empty.Empty) (*api.GetConfigResponse, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, "onepanel", "get", "onepanel.io", "configmap", "")
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
		ApiUrl:       sysConfig.GetValueOrEmpty("ONEPANEL_API_URL"),
		Domain:       sysConfig.GetValueOrEmpty("ONEPANEL_DOMAIN"),
		Fqdn:         sysConfig.GetValueOrEmpty("ONEPANEL_FQDN"),
		ProviderType: sysConfig.GetValueOrEmpty("PROVIDER_TYPE"),
		Database: &api.DatabaseConfig{
			DriverName: sysConfig.GetValueOrEmpty("databaseDriverName"),
			Host:       sysConfig.GetValueOrEmpty("databaseHost"),
			Name:       sysConfig.GetValueOrEmpty("databaseName"),
			Port:       sysConfig.GetValueOrEmpty("databasePort"),
		},
		NodePool: nodePool,
	}, err
}
