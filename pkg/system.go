package v1

import (
	"github.com/onepanelio/core/pkg/util"
	"google.golang.org/grpc/codes"
)

// ConvertToSystemName converts a name to a system name by prefixing it with "sys-"
func ConvertToSystemName(name string) string {
	return "sys-" + name
}

// NameReservedForSystemError is an error returned whenever the user tries to use a name reserved for the system
func NameReservedForSystemError() error {
	return &util.UserError{Code: codes.InvalidArgument, Message: "Names prefixed with 'sys-' are reserved by the system"}
}
