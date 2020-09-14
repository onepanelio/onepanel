package router

import (
	"fmt"
)

// Web provides methods to generate urls for the web client
// this can be used to generate urls for workspaces or workflows when they are ready.
type Web interface {
	WorkflowExecution(namespace, uid string) string
}

// web is a basic implementation of router.Web
type web struct {
	protocol string
	fqdn     string
}

// WorkflowExecution generates a url to view a specific workflow
func (w *web) WorkflowExecution(namespace, uid string) string {
	// <protocol><fqdn>/<namespace>/workflows/<uid>
	return fmt.Sprintf("%v%v/%v/workflows/%v", w.protocol, w.fqdn, namespace, uid)
}

// NewWebRouter creates a new web router used to generate urls for the web client
func NewWebRouter(protocol, fqdn string) (Web, error) {
	return &web{
		protocol: protocol,
		fqdn:     fqdn,
	}, nil
}

// NewRelativeWebRouter creates a web router that does relative routes, with no protocol or fqdn
func NewRelativeWebRouter() (Web, error) {
	return &web{
		protocol: "",
		fqdn:     "",
	}, nil
}
