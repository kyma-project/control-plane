package apperrors

import "fmt"

const (
	CodeBadGateway ErrCode = 502
	CodeInternal   ErrCode = 500
	CodeForbidden  ErrCode = 403
	CodeBadRequest ErrCode = 400
)

const (
	Unknown        CauseCode = 10
	TenantNotFound CauseCode = 11
)

type ErrCode int

type CauseCode int

type AppError interface {
	Append(string, ...interface{}) AppError
	Code() ErrCode
	Cause() CauseCode
	Error() string
}

type appError struct {
	code         ErrCode
	internalCode CauseCode
	message      string
}

func errorf(code ErrCode, cause CauseCode, format string, a ...interface{}) AppError {
	return appError{code: code, internalCode: cause, message: fmt.Sprintf(format, a...)}
}

func BadGateway(format string, a ...interface{}) AppError {
	return errorf(CodeBadGateway, Unknown, format, a...)
}

func Internal(format string, a ...interface{}) AppError {
	return errorf(CodeInternal, Unknown, format, a...)
}

func Forbidden(format string, a ...interface{}) AppError {
	return errorf(CodeForbidden, Unknown, format, a...)
}

func BadRequest(format string, a ...interface{}) AppError {
	return errorf(CodeBadRequest, Unknown, format, a...)
}

func InvalidTenant(format string, a ...interface{}) AppError {
	return errorf(CodeBadRequest, TenantNotFound, format, a...)
}

func (ae appError) Append(additionalFormat string, a ...interface{}) AppError {
	format := additionalFormat + ", " + ae.message
	return errorf(ae.code, ae.internalCode, format, a...)
}

func (ae appError) Code() ErrCode {
	return ae.code
}

func (ae appError) Error() string {
	return ae.message
}

func (ae appError) Cause() CauseCode {
	return ae.internalCode
}
