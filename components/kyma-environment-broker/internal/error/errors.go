package error

import (
	"github.com/pkg/errors"
	apierr "k8s.io/apimachinery/pkg/api/errors"
)

// error reporter
type LastError struct {
	message   string
	reason    ErrReason
	component ErrComponent
}

type ErrorReporter interface {
	error
	Reason() ErrReason
	Component() ErrComponent
}

type ErrReason string

const (
	ErrorKEBInternal ErrReason = "ERR_KEB_INTERNAL"
	ErrorKEBTimeOut  ErrReason = "ERR_KEB_TIMEOUT"
)

type ErrComponent string

const (
	ErrorDB           ErrComponent = "db - keb"
	ErrorK8SClient    ErrComponent = "k8s client"
	ErrorKEB          ErrComponent = "keb"
	ErrorEDP          ErrComponent = "edp"
	ErrorGrapQLClient ErrComponent = "graphql client"
)

func (err LastError) Reason() ErrReason {
	return err.reason
}

func (err LastError) Component() ErrComponent {
	return err.component
}

func (err LastError) Error() string {
	return err.message
}

func ReasonForError(err error) LastError {
	if err == nil {
		return LastError{}
	}

	cause := errors.Cause(err)

	if status := apierr.APIStatus(nil); errors.As(cause, &status) {
		return LastError{
			message:   err.Error(),
			reason:    ErrReason(apierr.ReasonForError(cause)),
			component: ErrorK8SClient,
		}
	}

	if status := ErrorReporter(nil); errors.As(cause, &status) {
		return LastError{
			message:   err.Error(),
			reason:    status.Reason(),
			component: status.Component(),
		}
	}

	return LastError{
		message:   err.Error(),
		reason:    ErrorKEBInternal,
		component: ErrorKEB,
	}
}
