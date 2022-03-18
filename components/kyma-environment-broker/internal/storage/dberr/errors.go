package dberr

import (
	"fmt"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
)

const (
	CodeInternal      = 1
	CodeNotFound      = 2
	CodeAlreadyExists = 3
	CodeConflict      = 4
)

type DBErrReason = kebError.ErrReason

const (
	ErrDBInternal      DBErrReason = "err_db_internal"
	ErrDBNotFound      DBErrReason = "err_db_not_found"
	ErrDBAlreadyExists DBErrReason = "err_db_already_exists"
	ErrDBConflict      DBErrReason = "err_db_conflict"
	ErrDBUnknown       DBErrReason = "err_db_unknown"
)

type Error interface {
	Append(string, ...interface{}) Error
	Code() int
	Error() string
}

type dbError struct {
	code    int
	message string
}

func errorf(code int, format string, a ...interface{}) Error {
	return dbError{code: code, message: fmt.Sprintf(format, a...)}
}

func Internal(format string, a ...interface{}) Error {
	return errorf(CodeInternal, format, a...)
}

func NotFound(format string, a ...interface{}) Error {
	return errorf(CodeNotFound, format, a...)
}

func IsNotFound(err error) bool {
	nf, ok := err.(interface {
		Code() int
	})
	return ok && nf.Code() == CodeNotFound
}

func AlreadyExists(format string, a ...interface{}) Error {
	return errorf(CodeAlreadyExists, format, a...)
}

func Conflict(format string, a ...interface{}) Error {
	return errorf(CodeConflict, format, a...)
}

func (e dbError) Append(additionalFormat string, a ...interface{}) Error {
	format := additionalFormat + ", " + e.message
	return errorf(e.code, format, a...)
}

func (e dbError) Code() int {
	return e.code
}

func (e dbError) Error() string {
	return e.message
}

func (e dbError) Component() kebError.ErrComponent {
	return kebError.ErrDB
}

func (e dbError) Reason() DBErrReason {
	reason := ErrDBUnknown

	switch e.code {
	case CodeInternal:
		reason = ErrDBInternal
	case CodeNotFound:
		reason = ErrDBNotFound
	case CodeAlreadyExists:
		reason = ErrDBAlreadyExists
	case CodeConflict:
		reason = ErrDBConflict
	}

	return reason
}

func IsConflict(err error) bool {
	dbe, ok := err.(Error)
	if !ok {
		return false
	}
	return dbe.Code() == CodeConflict
}
