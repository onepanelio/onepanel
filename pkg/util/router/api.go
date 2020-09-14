package router

import "fmt"

// API provides methods to generate urls for the API
type API interface {
	UpdateWorkspaceStatus(namespace, uid string) string
}

// api is a basic implementation of router.API
type api struct {
	protocol string
	fqdn     string
}

// UpdateWorkspaceStatus generates a url to update the status of a workspace
func (a *api) UpdateWorkspaceStatus(namespace, uid string) string {
	// <protocol><fqdn>/apis/v1beta1/{namespace}/workspaces/{uid}/status
	return fmt.Sprintf("%v%v/apis/v1beta1/%v/workspaces/%v/status", a.protocol, a.fqdn, namespace, uid)
}

// NewAPIRouter creates a new api router used to generate urls for the api
func NewAPIRouter(protocol, fqdn string) (API, error) {
	return &api{
		protocol: protocol,
		fqdn:     fqdn,
	}, nil
}

// NewRelativeAPIRouter creates an api router that does relative routes, with no protocol or fqdn
func NewRelativeAPIRouter() (API, error) {
	return &api{
		protocol: "",
		fqdn:     "",
	}, nil
}
