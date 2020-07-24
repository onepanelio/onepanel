package server

import (
	"context"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/server/auth"
)

// ComponentServer contains actions for installed components
type ComponentServer struct{}

// NewComponentServer creates a new ComponentServer
func NewComponentServer() *ComponentServer {
	return &ComponentServer{}
}

func apiComponent(component *v1.Component) (apiComponent *api.Component) {
	return &api.Component{
		Name: component.Name,
		Url:  component.URL,
	}
}

// ListComponents returns all of the components in the system
func (c *ComponentServer) ListComponents(ctx context.Context, req *api.ListComponentsRequest) (*api.ListComponentsResponse, error) {
	client := getClient(ctx)
	// TODO update the resource to be components
	allowed, err := auth.IsAuthorized(client, "", "list", "", "namespaces", "")
	if err != nil || !allowed {
		return nil, err
	}

	components, err := client.ListComponents(req.Namespace)
	if err != nil {
		return nil, err
	}

	apiComponents := make([]*api.Component, len(components))
	for i, component := range components {
		apiComponents[i] = apiComponent(component)
	}

	return &api.ListComponentsResponse{
		Count:      int32(len(components)),
		Components: apiComponents,
		Page:       1,
		Pages:      1,
		TotalCount: int32(len(components)),
	}, nil
}

// GetComponent returns a particular component identified by name
func (c *ComponentServer) GetComponent(ctx context.Context, req *api.GetComponentRequest) (*api.Component, error) {
	client := getClient(ctx)
	// TODO update the resource to be components
	allowed, err := auth.IsAuthorized(client, "", "get", "", "namespaces", "")
	if err != nil || !allowed {
		return nil, err
	}

	component, err := client.GetComponent(req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}

	apiComponent := apiComponent(component)

	return apiComponent, nil
}
