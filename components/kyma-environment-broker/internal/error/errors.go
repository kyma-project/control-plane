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

type ErrorCollector interface {
	error
	Code() int
	Reason() ErrorReason
	Component() ErrorComponent
}

type ErrorReason string

const (
	ErrorKEBInternal ErrorReason = "ERR_KEB_INTERNAL"
	ErrorKEBTimeOut  ErrorReason = "ERR_KEB_TIMEOUT"

	ErrorOther ErrorReason = "ERR_UNKNOWN"
)

type ErrorComponent string

const (
	ErrorDB        ErrorComponent = "db"
	ErrorK8SClient ErrorComponent = "k8s client"
	ErrorKEB       ErrorComponent = "keb"
	ErrorEDP       ErrorComponent = "edp"
)

func (err LastError) GetReason() ErrorReason {
	return err.Reason
}

func (err LastError) GetComponent() ErrorComponent {
	return err.Component
}

// error only has value for categoring reason if needed
func (err LastError) Error() string {
	return err.error.Error()
}

func (err LastError) lastErrorExists() bool {
	if err.Component != "" && err.error != nil {
		return true
	}

	return false
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
			Reason:    apierr.ReasonForError(status),
			Component: ErrorK8SClient,
		}
	}

	if status := ErrorCollector(nil); errors.As(err, &status) {
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

// func Clean(err *LastError) {
// 	if err == nil {
// 		return
// 	}
// 	err.Reasons = []ErrorReason{}
// 	err.Component = ""
// 	err.error = nil
// }
