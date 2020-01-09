package util

import (
	"errors"
	"fmt"

	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserError struct {
	code    codes.Code
	message string
}

func NewUserError(code codes.Code, message string) *UserError {
	return &UserError{
		code:    code,
		message: message,
	}
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

func NewUserErrorWrap(err error, entity string) *UserError {
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

	return &UserError{
		code:    code,
		message: message,
	}
}

func (e *UserError) Error() string {
	return e.message
}

func (e *UserError) GRPCError() error {
	return status.Errorf(e.code, e.Error())
}
