package apperrors

import "fmt"

const (
	CodeBadGateway ErrCode = 502
	CodeInternal   ErrCode = 500
	CodeForbidden  ErrCode = 403
	CodeBadRequest ErrCode = 400
)

const (
	IntCodeBadGateway     IntErrCode = 10
	IntCodeInternal       IntErrCode = 11
	IntCodeForbidden      IntErrCode = 12
	IntCodeBadRequest     IntErrCode = 13
	IntCodeTenantNotFound IntErrCode = 13
)

type ErrCode int

type IntErrCode int

type AppError interface {
	Append(string, ...interface{}) AppError
	Code() ErrCode
	IntCode() IntErrCode
	Error() string
}

type appError struct {
	code         ErrCode
	internalCode IntErrCode
	message      string
}

func errorf(code ErrCode, internalCode IntErrCode, format string, a ...interface{}) AppError {
	return appError{code: code, internalCode: internalCode, message: fmt.Sprintf(format, a...)}
}

func BadGateway(format string, a ...interface{}) AppError {
	return errorf(CodeBadGateway, IntCodeBadGateway, format, a...)
}

func Internal(format string, a ...interface{}) AppError {
	return errorf(CodeInternal, IntCodeInternal, format, a...)
}

func Forbidden(format string, a ...interface{}) AppError {
	return errorf(CodeForbidden, IntCodeForbidden, format, a...)
}

func BadRequest(format string, a ...interface{}) AppError {
	return errorf(CodeBadRequest, IntCodeBadRequest, format, a...)
}

func WrongTenant(format string, a ...interface{}) AppError {
	return errorf(CodeBadRequest, IntCodeTenantNotFound, format, a...)
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

func (ae appError) IntCode() IntErrCode {
	return ae.internalCode
}
