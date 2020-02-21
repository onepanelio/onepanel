package util

import (
	"errors"
	"fmt"

	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewUserError(code codes.Code, message string) error {
	return status.Errorf(code, message)
}

func pqError(err *pq.Error) (code codes.Code) {
	switch err.Code {
	case "23505":
		code = codes.AlreadyExists
	default:
		code = codes.Unknown
	}
	return
}

func NewUserErrorWrap(err error, entity string) error {
	var (
		code    codes.Code
		message string
		pqErr   *pq.Error
	)
	if errors.As(err, &pqErr) {
		code = pqError(pqErr)
		message = fmt.Sprintf("%v already exists.", entity)
	} else {
		code = codes.Unknown
		message = "Unknown error."
	}

	return NewUserError(code, message)
}
