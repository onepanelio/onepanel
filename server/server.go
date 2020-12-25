package server

import (
	"context"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/server/auth"
	"strings"
)

const (
	TimeLayout = "2006-01-02 15:04:05"
)

func getClient(ctx context.Context) *v1.Client {
	return ctx.Value(auth.ContextClientKey).(*v1.Client)
}

// IsNameReservedForSystem returns true if the name is reserved for the system
func IsNameReservedForSystem(name string) bool {
	return strings.HasPrefix(name, "sys-")
}
