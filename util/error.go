package util

import (
	"errors"
	"fmt"

	"github.com/lib/pq"
)

type UserError struct {
	Code    int
	Message string
}

func NewUserError(code int, message string) *UserError {
	return &UserError{
		Code:    code,
		Message: message,
	}
}

func pqError(err *pq.Error) (code int) {
	switch err.Code {
	case "23505":
		code = 409
	default:
		code = 500
	}
	return
}

func UserErrorWrap(err error, entity string) *UserError {
	var (
		code    int
		message string
		pqErr   *pq.Error
	)
	if errors.As(err, &pqErr) {
		code = pqError(pqErr)
		message = fmt.Sprintf("%v already exists.", entity)
	} else {
		code = 500
		message = "Unknown error."
	}

	return &UserError{
		Code:    code,
		Message: message,
	}
}

func (e *UserError) Error() string {
	return e.Message
}
