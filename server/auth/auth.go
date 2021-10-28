package auth

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	api "github.com/onepanelio/core/api/gen"
	"github.com/onepanelio/core/pkg/util"
	log "github.com/sirupsen/logrus"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	v1 "github.com/onepanelio/core/pkg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	authorizationv1 "k8s.io/api/authorization/v1"
)

type key int

const (
	// ContextClientKey is the key used to identify the Client value in Context
	ContextClientKey key = iota
)

func getBearerToken(ctx context.Context) (*string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.WithFields(log.Fields{
			"Method": "getBearerToken",
		}).Error("Unable to get metadata from incoming context")
		return nil, false
	}

	prefix := "Bearer "
	for _, t := range md.Get("authorization") {
		if !strings.HasPrefix(t, prefix) {
			return nil, false
		}
		t = strings.ReplaceAll(t, prefix, "")
		if t == "null" {
			return nil, false
		}
		return &t, true
	}

	for _, c := range md.Get("grpcgateway-cookie") {
		header := http.Header{}
		header.Add("Cookie", c)
		req := &http.Request{
			Header: header,
		}
		t, _ := req.Cookie("auth-token")
		if t != nil {
			return &t.Value, true
		}
	}

	for _, t := range md.Get("onepanel-auth-token") {
		return &t, true
	}

	for _, t := range md.Get("onepanel-access-token") {
		return &t, true
	}

	log.WithFields(log.Fields{
		"Method": "getBearerToken",
	}).Error("Unable to get BearerToken:", md)

	return nil, false
}

func getClient(ctx context.Context, kubeConfig *v1.Config, db *v1.DB, sysConfig v1.SystemConfig) (context.Context, error) {
	if kubeConfig == nil {
		return nil, fmt.Errorf("getClient - nil passed in for kubeConfig")
	}
	if db == nil {
		return nil, fmt.Errorf("getClient - nil passed in for db")
	}

	bearerToken, ok := getBearerToken(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, `Missing or invalid "authorization" header.`)
	}
	if bearerToken == nil {
		return nil, status.Error(codes.Unauthenticated, "Bearer token is nil")
	}

	kubeConfig.BearerToken = *bearerToken

	client, err := v1.NewClient(kubeConfig, db, sysConfig)
	if err != nil {
		return nil, err
	}
	client.Token = kubeConfig.BearerToken

	return context.WithValue(ctx, ContextClientKey, client), nil
}

func IsAuthorized(c *v1.Client, namespace, verb, group, resource, name string) (allowed bool, err error) {
	review, err := c.AuthorizationV1().SelfSubjectAccessReviews().Create(&authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      verb,
				Group:     group,
				Resource:  resource,
				Name:      name,
			},
		},
	})

	deniedMsg := fmt.Sprintf(`Permission denied. Namespace: '%v', Verb: '%v', Group: '%v', Resource '%v', Name: '%v'`, namespace, verb, group, resource, name)
	if err != nil {
		return false, status.Error(codes.PermissionDenied, deniedMsg)
	}
	allowed = review.Status.Allowed
	if !allowed {
		return false, status.Error(codes.PermissionDenied, deniedMsg)
	}

	return
}

func verifyLogin(client *v1.Client, tokenRequest *api.GetAccessTokenRequest) (rawToken string, err error) {
	namespaces := []*v1.Namespace{{
		Name: "onepanel",
	}}

	additionalNamespaces, err := client.ListOnepanelEnabledNamespaces()
	if err != nil {
		return "", err
	}

	namespaces = append(namespaces, additionalNamespaces...)

	for _, namespace := range namespaces {
		namespaceName := namespace.Name
		accountsList, err := client.CoreV1().ServiceAccounts(namespaceName).List(v1.ListOptions{})
		if err != nil {
			return "", err
		}

		authTokenSecretName := ""
		for _, serviceAccount := range accountsList.Items {
			if serviceAccount.Name != tokenRequest.Username {
				continue
			}
			for _, secret := range serviceAccount.Secrets {
				if strings.Contains(secret.Name, "-token-") {
					authTokenSecretName = secret.Name
					break
				}
			}
		}
		if authTokenSecretName == "" {
			continue
		}

		secret, err := client.CoreV1().Secrets(namespaceName).Get(authTokenSecretName, v12.GetOptions{})
		if err != nil {
			return "", err
		}

		currentTokenBytes := md5.Sum(secret.Data["token"])
		currentTokenString := hex.EncodeToString(currentTokenBytes[:])

		if tokenRequest.Token != fmt.Sprintf("%s", currentTokenString) {
			continue
		}

		return string(secret.Data["token"]), nil
	}

	return "", util.NewUserError(codes.InvalidArgument, fmt.Sprintf("unknown username/token '%v'", tokenRequest.Username))
}

// UnaryInterceptor performs authentication checks.
// The two main cases are:
//   1. Is the token valid? This is used for logging in.
//   2. Is there a token? There should be a token for everything except logging in.
func UnaryInterceptor(kubeConfig *v1.Config, db *v1.DB, sysConfig v1.SystemConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		// Check if the provided token is valid. This does not require a token in the header.
		if info.FullMethod == "/api.AuthService/GetAccessToken" {
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				return resp, errors.New("unable to get metadata from incoming context")
			}

			getAccessTokenRequest, ok := req.(*api.GetAccessTokenRequest)
			if !ok {
				return resp, errors.New("invalid request object for GetAccessTokenRequest")
			}

			defaultClient, err := v1.GetDefaultClientWithDB(db)
			if err != nil {
				return nil, err
			}

			rawToken, err := verifyLogin(defaultClient, getAccessTokenRequest)
			if err != nil {
				return nil, err
			}

			sysConfig, err := defaultClient.GetSystemConfig()
			if err != nil {
				return nil, err
			}

			md.Set("authorization", "Bearer "+rawToken)

			ctx, err = getClient(ctx, kubeConfig, db, sysConfig)
			if err != nil {
				ctx = nil
			}

			return handler(ctx, req)
		}
		if info.FullMethod == "/api.AuthService/IsValidToken" {
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				return resp, errors.New("unable to get metadata from incoming context")
			}

			tokenRequest, ok := req.(*api.IsValidTokenRequest)
			if !ok {
				return resp, errors.New("invalid request object for GetAccessTokenRequest")
			}
			getAccessTokenRequest := &api.GetAccessTokenRequest{
				Username: tokenRequest.Username,
				Token:    tokenRequest.Token,
			}

			defaultClient, err := v1.GetDefaultClientWithDB(db)
			if err != nil {
				return nil, err
			}

			rawToken, err := verifyLogin(defaultClient, getAccessTokenRequest)
			if err != nil {
				return nil, err
			}

			sysConfig, err := defaultClient.GetSystemConfig()
			if err != nil {
				return nil, err
			}

			md.Set("authorization", "Bearer "+rawToken)

			ctx, err = getClient(ctx, kubeConfig, db, sysConfig)
			if err != nil {
				ctx = nil
			}

			return handler(ctx, req)
		}
		if info.FullMethod == "/api.AuthService/IsAuthorized" {
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				ctx = nil
				return handler(ctx, req)
			}

			//Expected format: x-original-authority:[name--default.alexcluster.onepanel.io]
			if xOriginalAuthStrings := md.Get("x-original-authority"); xOriginalAuthStrings != nil {
				xOriginalAuth := xOriginalAuthStrings[0]
				dotIndex := strings.Index(xOriginalAuth, ".")
				if dotIndex != -1 {
					workspaceAndNamespace := xOriginalAuth[0:dotIndex]
					pieces := strings.Split(workspaceAndNamespace, "--")
					if len(pieces) > 1 {
						workspaceName := pieces[0]
						namespace := pieces[len(pieces)-1]

						isAuthorizedRequest, ok := req.(*api.IsAuthorizedRequest)
						if ok {
							isAuthorizedRequest.IsAuthorized.Namespace = namespace
							isAuthorizedRequest.IsAuthorized.Resource = "workspaces"
							isAuthorizedRequest.IsAuthorized.Group = "onepanel.io"
							isAuthorizedRequest.IsAuthorized.ResourceName = workspaceName
							isAuthorizedRequest.IsAuthorized.Verb = "get"
						}
					}
				}
			}
		}

		// This guy checks for the token
		ctx, err = getClient(ctx, kubeConfig, db, sysConfig)
		if err != nil {
			return
		}

		return handler(ctx, req)
	}
}

// StreamingInterceptor provides an authentication wrapper around streaming requests.
func StreamingInterceptor(kubeConfig *v1.Config, db *v1.DB, sysConfig v1.SystemConfig) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		ctx, err := getClient(ss.Context(), kubeConfig, db, sysConfig)
		if err != nil {
			return
		}
		wrapped := grpc_middleware.WrapServerStream(ss)
		wrapped.WrappedContext = ctx

		return handler(srv, wrapped)
	}
}
