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
	ErrEDPconflict   EDPErrReason = "ERR_EDP_INTERNAL"
	ErrEDPNotFound   EDPErrReason = "ERR_EDP_NOT_FOUND"
	ErrEDPBadRequest EDPErrReason = "ERR_EDP_BAD_REQUEST"
	ErrEDPOther      EDPErrReason = "ERR_EDP_OTHER"
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

func (e edpError) Component() string {
	return kebError.ErrorEDP
}

func (e edpError) Reason() EDPErrReason {
	reason := ErrEDPOther

	switch e.code {
	case http.StatusConflict:
		reason = ErrEDPconflict
	case http.StatusNotFound:
		reason = ErrEDPNotFound
	case http.StatusBadRequest:
		reason = ErrEDPBadRequest
	}

	return reason
}

func IsConflictError(err error) bool {
	e, ok := err.(kebError.ErrorReporter)
	if !ok {
		return false
	}
	return e.Code() == http.StatusConflict
}
