package server

import (
	"context"
	"fmt"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/server/auth"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthServer struct {
	fqdns map[string]string // map from namespace -> namespace.domain
}

func NewAuthServer(namespaces []*v1.Namespace, config map[string]string) *AuthServer {
	server := &AuthServer{
		fqdns: make(map[string]string),
	}

	for _, namespace := range namespaces {
		server.fqdns[namespace.Name] = namespace.Name + "." + config["ONEPANEL_DOMAIN"]
	}

	return server
}

func (a *AuthServer) IsAuthorized(ctx context.Context, request *api.IsAuthorizedRequest) (res *api.IsAuthorizedResponse, err error) {
	res = &api.IsAuthorizedResponse{}
	if ctx == nil {
		res.Authorized = false
		return res, status.Error(codes.Unauthenticated, "Unauthenticated.")
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return res, status.Error(codes.Unauthenticated, "Unauthenticated.")
	}

	// If no authorization, we permit access if the x-original-authority == the expected fqdn
	if md.Get("authorization") == nil {
		xOriginalAuthorityStrings := md.Get("x-original-authority")
		if xOriginalAuthorityStrings == nil {
			return res, status.Error(codes.Unauthenticated, "Unauthenticated.")
		}
		xOriginalAuthority := xOriginalAuthorityStrings[0]

		fqdnEnd, ok := a.fqdns[request.Namespace]
		if !ok {
			return res, status.Error(codes.Unauthenticated, "Unauthenticated.")
		}
		fqdn := fmt.Sprintf("%v--%v", request.GetResourceName(), fqdnEnd)

		if fqdn != xOriginalAuthority {
			return res, status.Error(codes.Unauthenticated, "Unauthenticated.")
		}

		res.Authorized = true
		return res, nil
	}

	client := ctx.Value("kubeClient").(*v1.Client)

	//User auth check
	err = a.isValidToken(err, client)
	if err != nil {
		return nil, err
	}
	//Check the request
	allowed, err := auth.IsAuthorized(client, request.Namespace, request.Verb, request.Group, request.Resource, request.ResourceName)
	if err != nil {
		res.Authorized = false
		return res, err
	}

	res.Authorized = allowed
	return res, nil
}

func (a *AuthServer) IsValidToken(ctx context.Context, req *api.IsValidTokenRequest) (res *api.IsValidTokenResponse, err error) {
	if ctx == nil {
		return nil, status.Error(codes.Unauthenticated, "Unauthenticated.")
	}

	client := ctx.Value("kubeClient").(*v1.Client)

	err = a.isValidToken(err, client)
	if err != nil {
		return nil, err
	}

	config, err := client.GetSystemConfig()
	if err != nil {
		return
	}
	res = &api.IsValidTokenResponse{}
	res.Domain = config["ONEPANEL_DOMAIN"]

	return res, nil
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
		return errors.New("No namespaces for onepanel setup.")
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
