package server

import (
	"context"
	"fmt"
	api "github.com/onepanelio/core/api/gen"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util"
	"github.com/onepanelio/core/server/auth"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AuthServer contains logic for checking Authorization of resources in the system
type AuthServer struct {
	api.UnimplementedAuthServiceServer
}

// NewAuthServer creates a new AuthServer
func NewAuthServer() *AuthServer {
	return &AuthServer{}
}

// IsAuthorized checks if the provided action is authorized.
// No token == unauthorized. This is indicated by a nil ctx.
// Invalid token == unauthorized.
// Otherwise, we check with k8s using all of the provided data in the request.
func (a *AuthServer) IsAuthorized(ctx context.Context, request *api.IsAuthorizedRequest) (res *api.IsAuthorizedResponse, err error) {
	res = &api.IsAuthorizedResponse{}
	if ctx == nil {
		res.Authorized = false
		return res, status.Error(codes.Unauthenticated, "Unauthenticated.")
	}
	//User auth check
	client := getClient(ctx)

	err = a.isValidToken(err, client)
	if err != nil {
		return nil, err
	}

	//Check the request
	allowed, err := auth.IsAuthorized(client, request.IsAuthorized.Namespace, request.IsAuthorized.Verb, request.IsAuthorized.Group, request.IsAuthorized.Resource, request.IsAuthorized.ResourceName)
	if err != nil {
		res.Authorized = false
		return res, util.NewUserError(codes.PermissionDenied, fmt.Sprintf("Namespace: %v, Verb: %v, Group: \"%v\", Resource: %v. Source: %v", request.IsAuthorized.Namespace, request.IsAuthorized.Verb, request.IsAuthorized.Group, request.IsAuthorized.ResourceName, err))
	}

	res.Authorized = allowed
	return res, nil
}

// GetAccessToken is an alias for IsValidToken. It returns a token given a username and hashed token.
func (a *AuthServer) GetAccessToken(ctx context.Context, req *api.GetAccessTokenRequest) (res *api.GetAccessTokenResponse, err error) {
	if ctx == nil {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}

	client := getClient(ctx)
	err = a.isValidToken(err, client)
	if err != nil {
		return nil, err
	}

	config, err := client.GetSystemConfig()
	if err != nil {
		return
	}

	domain := config.Domain()
	if domain == nil {
		return nil, fmt.Errorf("domain is not set")
	}

	// This is for backwards compatibility
	// Originally, when you logged in as the admin, you would get the defaultNamespace as the
	// namespace.
	if req.Username == "admin" {
		nsList, err := client.CoreV1().Namespaces().List(metav1.ListOptions{
			LabelSelector: "onepanel.io/defaultNamespace=true",
		})

		if err != nil {
			return nil, err
		}

		if len(nsList.Items) == 1 {
			req.Username = nsList.Items[0].Name
		}
	}

	res = &api.GetAccessTokenResponse{
		Domain:      *domain,
		AccessToken: client.Token,
		Username:    req.Username,
	}

	return
}

// IsValidToken returns the appropriate token information given an md5 version of the token
// Deprecated: Use GetAccessToken instead
func (a *AuthServer) IsValidToken(ctx context.Context, req *api.IsValidTokenRequest) (res *api.IsValidTokenResponse, err error) {
	if ctx == nil {
		return nil, status.Error(codes.Unauthenticated, "unauthenticated")
	}

	client := getClient(ctx)
	err = a.isValidToken(err, client)
	if err != nil {
		return nil, err
	}

	config, err := client.GetSystemConfig()
	if err != nil {
		return
	}

	domain := config.Domain()
	if domain == nil {
		return nil, fmt.Errorf("domain is not set")
	}

	res = &api.IsValidTokenResponse{
		Domain:   *domain,
		Token:    client.Token,
		Username: req.Username,
	}

	return
}

func (a *AuthServer) isValidToken(err error, client *v1.Client) error {
	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		if err.Error() == "Unauthorized" {
			return status.Error(codes.Unauthenticated, "Unauthenticated.")
		}
		return err
	}
	if len(namespaces) == 0 {
		return errors.New("no namespaces for onepanel setup")
	}
	namespace := namespaces[0]

	allowed, err := auth.IsAuthorized(client, "", "get", "", "namespaces", namespace.Name)
	if err != nil {
		return err
	}

	if !allowed {
		return status.Error(codes.Unauthenticated, "Unauthenticated.")
	}
	return nil
}
