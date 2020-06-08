package server

import (
	"context"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/server/auth"
)

const (
	TimeLayout = "2006-01-02 15:04:05"
)

func getClient(ctx context.Context) *v1.Client {
	return ctx.Value(auth.ClientContextKey).(*v1.Client)
}
