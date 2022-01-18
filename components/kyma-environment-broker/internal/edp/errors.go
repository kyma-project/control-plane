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

type EDPErrorReason = kebError.ErrorReason

const (
	ErrorEDPconflict   EDPErrorReason = "ERR_EDP_INTERNAL"
	ErrorEDPNotFound   EDPErrorReason = "ERR_EDP_NOT_FOUND"
	ErrorEDPBadRequest EDPErrorReason = "ERR_EDP_BAD_REQUEST"
	ErrorEDPOther      EDPErrorReason = "ERR_EDP_OTHER"
)

func errorf(id string, code int, format string, args ...interface{}) kebError.ErrorCollector {
	return edpError{id: id, code: code, message: fmt.Sprintf(format, args...)}
}

func NewEDPConflictError(id string, format string, args ...interface{}) kebError.ErrorCollector {
	return errorf(id, http.StatusConflict, format, args...)
}

func NewEDPNotFoundError(id string, format string, args ...interface{}) kebError.ErrorCollector {
	return errorf(id, http.StatusNotFound, format, args...)
}

func NewEDPBadRequestError(id string, format string, args ...interface{}) kebError.ErrorCollector {
	return errorf(id, http.StatusBadRequest, format, args...)
}

func NewEDPOtherError(id string, code int, format string, args ...interface{}) kebError.ErrorCollector {
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

func (e edpError) Reason() string {
	reason := ErrorEDPOther

	switch e.code {
	case http.StatusConflict:
		reason = ErrorEDPconflict
	case http.StatusNotFound:
		reason = ErrorEDPNotFound
	case http.StatusBadRequest:
		reason = ErrorEDPBadRequest
	}

	return reason
}

func IsConflictError(err error) bool {
	e, ok := err.(kebError.ErrorCollector)
	if !ok {
		return false
	}
	return e.Code() == http.StatusConflict
}
