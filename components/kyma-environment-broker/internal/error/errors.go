package error

import (
	"github.com/pkg/errors"
	apierr "k8s.io/apimachinery/pkg/api/errors"
)

type Error interface {
	error

	GetReason() ErrorReason
	GetComponent() ErrorComponent
}

// component and error to confirm an error alert
type LastError struct {
	error
	Reason    ErrorReason
	Component ErrorComponent
}

type ErrorReporter interface {
	error
	Reason() ErrorReason
	Component() ErrorComponent
}

type ErrorReason string

const (
	ErrorKEBInternal ErrorReason = "ERR_KEB_INTERNAL"
	ErrorKEBTimeOut  ErrorReason = "ERR_KEB_TIMEOUT"
)

type ErrorComponent string

const (
	ErrorDB           ErrorComponent = "db"
	ErrorK8SClient    ErrorComponent = "k8s client"
	ErrorKEB          ErrorComponent = "keb"
	ErrorEDP          ErrorComponent = "edp"
	ErrorGrapQLClient ErrorComponent = "graphql client"
)

func (err LastError) GetReason() ErrorReason {
	return err.Reason
}

func (err LastError) GetComponent() ErrorComponent {
	return err.Component
}

func (err LastError) Error() string {
	return err.error.Error()
}

func ReasonForError(err error) LastError {
	if err == nil {
		return LastError{}
	}

	// if status := dberr.Error(nil); errors.As(err, &status) {
	// 	return LastError{
	// 		error:     err,
	// 		Reason:    status.Reason(),
	// 		Component: ErrorDB,
	// 	}
	// }

	if status := apierr.APIStatus(nil); errors.As(err, &status) {
		return LastError{
			error:     err,
			Reason:    ErrorReason(apierr.ReasonForError(err)),
			Component: ErrorK8SClient,
		}
	}

	if status := ErrorReporter(nil); errors.As(err, &status) {
		return LastError{
			error:     err,
			Reason:    status.Reason(),
			Component: status.Component(),
		}
	}

	return LastError{
		error:     err,
		Reason:    ErrorKEBInternal,
		Component: ErrorKEB,
	}
}
