package apperrors

import (
	"fmt"

	"github.com/pkg/errors"
)

type ErrReason string
type ErrComponent string

type ErrorReporter interface {
	error
	Component() ErrComponent
	Reason() ErrReason
	Code() ErrCode
}

const (
	ErrDB ErrComponent = "db"
	// ErrGrapQLClient ErrComponent = "graphql client"
	ErrK8SClient   ErrComponent = "k8s client"
	ErrProvisioner ErrComponent = "provisioner"
	ErrDirector    ErrComponent = "director"
)

const (
	ErrProvisionerInternal ErrReason = "err_provisioner_internal"
	ErrDirectorNilResponse ErrReason = "err_director_nil_response"
)

type ErrCode int
type CauseCode int

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

type AppError interface {
	Append(string, ...interface{}) AppError
	SetReason(ErrReason) AppError
	SetComponent(ErrComponent) AppError

	Code() ErrCode
	Cause() CauseCode
	Component() ErrComponent
	Reason() ErrReason
	Error() string
}

type appError struct {
	code         ErrCode
	internalCode CauseCode
	reason       ErrReason
	component    ErrComponent
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

func ConvertToAppError(err error) AppError {
	if customErr := AppError(nil); errors.As(err, &customErr) {
		return customErr
	}
	return Internal(err.Error())
}

func (ae appError) Append(additionalFormat string, a ...interface{}) AppError {
	format := additionalFormat + ", " + ae.message
	return errorf(ae.code, ae.internalCode, format, a...)

}

func (ae appError) SetReason(reason ErrReason) AppError {
	ae.reason = reason
	return ae
}

func (ae appError) SetComponent(comp ErrComponent) AppError {
	ae.component = comp
	return ae
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

func (ae appError) Component() ErrComponent {
	if ae.component == "" {
		return ErrProvisioner
	}
	return ae.component
}

func (ae appError) Reason() ErrReason {
	if (ae.component == "" || ae.component == ErrProvisioner) && ae.reason == "" {
		return ErrProvisionerInternal
	}
	return ae.reason
}
