package server

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/server/auth"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"strings"
)

type AuthServer struct{}

func NewAuthServer() *AuthServer {
	return &AuthServer{}
}
func (a *AuthServer) IsWorkspaceAuthenticated(ctx context.Context, request *api.IsWorkspaceAuthenticatedRequest) (*empty.Empty, error) {
	if ctx == nil {
		return &empty.Empty{}, nil
	}
	client := ctx.Value("kubeClient").(*v1.Client)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return &empty.Empty{}, errors.New("Error parsing headers.")
	}
	//Expected format: x-original-authority:[name--default.alexcluster.onepanel.io]
	xOriginalAuth := md.Get("x-original-authority")[0]
	fqdn := md.Get("fqdn")[0]
	if xOriginalAuth == fqdn {
		return &empty.Empty{}, nil
	}
	pos := strings.Index(xOriginalAuth, ".")
	if pos == -1 {
		return &empty.Empty{}, errors.New("Error parsing x-original-authority. No '.' character.")
	}
	workspaceAndNamespace := xOriginalAuth[0:pos]
	pieces := strings.Split(workspaceAndNamespace, "--")
	_, err := auth.IsAuthorized(client, pieces[1], "create", "apps", "statefulsets", pieces[0])
	if err != nil {
		return &empty.Empty{}, err
	}
	return &empty.Empty{}, nil
}

func (a *AuthServer) IsValidToken(ctx context.Context, req *api.IsValidTokenRequest) (res *api.IsValidTokenResponse, err error) {
	if ctx == nil {
		return nil, status.Error(codes.Unauthenticated, "Unauthenticated.")
	}

	client := ctx.Value("kubeClient").(*v1.Client)

	namespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		if err.Error() == "Unauthorized" {
			return nil, status.Error(codes.Unauthenticated, "Unauthenticated.")
		}
		return nil, err
	}
	if len(namespaces) == 0 {
		return nil, errors.New("No namespaces for onepanel setup.")
	}
	namespace := namespaces[0]

	allowed, err := auth.IsAuthorized(client, "", "get", "", "namespaces", namespace.Name)
	if err != nil {
		return nil, err
	}

	if !allowed {
		return nil, status.Error(codes.Unauthenticated, "Unauthenticated.")
	}

	config, err := client.GetSystemConfig()
	if err != nil {
		return
	}
	res = &api.IsValidTokenResponse{}
	res.Domain = config["ONEPANEL_DOMAIN"]

	return res, nil
}
