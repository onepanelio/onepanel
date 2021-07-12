package server

import (
	"context"
	api "github.com/onepanelio/core/api/gen"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/server/auth"
)

// ServiceServer contains actions for installed services
type ServiceServer struct {
	api.UnimplementedServiceServiceServer
}

// NewServiceServer creates a new ServiceServer
func NewServiceServer() *ServiceServer {
	return &ServiceServer{}
}

func apiService(service *v1.Service) (apiService *api.Service) {
	return &api.Service{
		Name: service.Name,
		Url:  service.URL,
	}
}

// ListServices returns all of the services in the system
func (c *ServiceServer) ListServices(ctx context.Context, req *api.ListServicesRequest) (*api.ListServicesResponse, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, "", "list", "", "onepanel-service", "")
	if err != nil || !allowed {
		return nil, err
	}

	services, err := client.ListServices(req.Namespace)
	if err != nil {
		return nil, err
	}

	apiServices := make([]*api.Service, len(services))
	for i, service := range services {
		apiServices[i] = apiService(service)
	}

	return &api.ListServicesResponse{
		Count:      int32(len(services)),
		Services:   apiServices,
		Page:       1,
		Pages:      1,
		TotalCount: int32(len(services)),
	}, nil
}

// GetService returns a particular service identified by name
func (c *ServiceServer) GetService(ctx context.Context, req *api.GetServiceRequest) (*api.Service, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, "", "get", "", "onepanel-service", "")
	if err != nil || !allowed {
		return nil, err
	}

	service, err := client.GetService(req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}

	apiService := apiService(service)

	return apiService, nil
}

// HasService checks if the cluster has a service set up and enabled
func (c *ServiceServer) HasService(ctx context.Context, req *api.HasServiceRequest) (*api.HasServiceResponse, error) {
	client := getClient(ctx)
	allowed, err := auth.IsAuthorized(client, "", "get", "", "onepanel-service", "")
	if err != nil || !allowed {
		return nil, err
	}

	hasService, err := client.HasService(req.Name)
	if err != nil {
		return nil, err
	}

	return &api.HasServiceResponse{
		HasService: hasService,
	}, nil
}
