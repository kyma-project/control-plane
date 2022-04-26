package edp

import (
	"fmt"
	"net/http"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
)

type edpError struct {
	id      string
	code    int
	message string
}

type EDPErrReason = kebError.ErrReason

const (
	ErrEDPConflict   EDPErrReason = "err_edp_internal"
	ErrEDPNotFound   EDPErrReason = "err_edp_not_found"
	ErrEDPBadRequest EDPErrReason = "err_edp_bad_request"
	ErrEDPTimeout    EDPErrReason = "err_edp_timeout"
	ErrEDPOther      EDPErrReason = "err_edp_other"
)

func errorf(id string, code int, format string, args ...interface{}) kebError.ErrorReporter {
	return edpError{id: id, code: code, message: fmt.Sprintf(format, args...)}
}

func NewEDPConflictError(id string, format string, args ...interface{}) kebError.ErrorReporter {
	return errorf(id, http.StatusConflict, format, args...)
}

func NewEDPNotFoundError(id string, format string, args ...interface{}) kebError.ErrorReporter {
	return errorf(id, http.StatusNotFound, format, args...)
}

func NewEDPBadRequestError(id string, format string, args ...interface{}) kebError.ErrorReporter {
	return errorf(id, http.StatusBadRequest, format, args...)
}

func NewEDPOtherError(id string, code int, format string, args ...interface{}) kebError.ErrorReporter {
	return errorf(id, code, format, args...)
}

func (e edpError) Error() string {
	return e.message
}

func (e edpError) Code() int {
	return e.code
}

func (e edpError) Component() kebError.ErrComponent {
	return kebError.ErrEDP
}

func (e edpError) Reason() EDPErrReason {
	reason := ErrEDPOther

	switch e.code {
	case http.StatusConflict:
		reason = ErrEDPConflict
	case http.StatusNotFound:
		reason = ErrEDPNotFound
	case http.StatusBadRequest:
		reason = ErrEDPBadRequest
	case http.StatusRequestTimeout:
		reason = ErrEDPTimeout
	}

	return reason
}

func IsConflictError(err error) bool {
	e, ok := err.(edpError)
	if !ok {
		return false
	}
	return e.Code() == http.StatusConflict
}
