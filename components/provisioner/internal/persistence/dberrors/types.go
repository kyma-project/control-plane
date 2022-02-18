package dberrors

import (
	"fmt"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
)

type dbErrCode = apperrors.ErrCode

const (
	CodeInternal      dbErrCode = 1
	CodeNotFound      dbErrCode = 2
	CodeAlreadyExists dbErrCode = 3
)

type dbErrReason = apperrors.ErrReason

const (
	ErrDBInternal      dbErrReason = "err_db_internal"
	ErrDBNotFound      dbErrReason = "err_db_not_found"
	ErrDBAlreadyExists dbErrReason = "err_db_already_exists"
	ErrDBUnknown       dbErrReason = "err_db_unknown"
)

// type Error interface {
// 	Append(string, ...interface{}) Error
// 	Code() apperrors.ErrCode
// 	Error() string
// }
type Error = apperrors.AppError

type dbError struct {
	code    dbErrCode
	message string
}

func errorf(code dbErrCode, format string, a ...interface{}) Error {
	return dbError{code: code, message: fmt.Sprintf(format, a...)}
}

func Internal(format string, a ...interface{}) Error {
	return errorf(CodeInternal, format, a...)
}

func NotFound(format string, a ...interface{}) Error {
	return errorf(CodeNotFound, format, a...)
}

func AlreadyExists(format string, a ...interface{}) Error {
	return errorf(CodeAlreadyExists, format, a...)
}

func (e dbError) Append(additionalFormat string, a ...interface{}) Error {
	format := additionalFormat + ", " + e.message
	return errorf(e.code, format, a...)
}

func (e dbError) Code() dbErrCode {
	return e.code
}

func (e dbError) Error() string {
	return e.message
}

func (e dbError) Component() apperrors.ErrComponent {
	return apperrors.ErrDB
}

func (e dbError) Reason() dbErrReason {
	reason := ErrDBUnknown

	switch e.code {
	case CodeInternal:
		reason = ErrDBInternal
	case CodeNotFound:
		reason = ErrDBNotFound
	case CodeAlreadyExists:
		reason = ErrDBAlreadyExists
	}

	return reason
}

func (e dbError) SetReason(reason dbErrReason) Error {
	return e
}

func (e dbError) SetComponent(comp apperrors.ErrComponent) Error {
	return e
}

func (e dbError) Cause() apperrors.CauseCode {
	return -1
}
