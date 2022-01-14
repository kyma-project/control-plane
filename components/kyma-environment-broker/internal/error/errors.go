package error

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

type ErrorReason string

const (
	ErrorDBNotFound      ErrorReason = "ERR_DB_NOT_FOUND"
	ErrorDBInternal      ErrorReason = "ERR_DB_INTERNAL"
	ErrorDBAlreadyExists ErrorReason = "ERR_DB_ALREADY_EXISTS"
	ErrorDBConflict      ErrorReason = "ERR_DB_CONFLICT"
	ErrorDBUnknown       ErrorReason = "ERR_DB_UNKNOWN"

	ErrorKEBInternal ErrorReason = "ERR_KEB_INTERNAL"
	ErrorKEBTimeOut  ErrorReason = "ERR_KEB_TIMEOUT"

	ErrorOther ErrorReason = "ERR_UNKNOWN"
)

type ErrorComponent string

const (
	ErrorDB        ErrorComponent = "db"
	ErrorK8SClient ErrorComponent = "k8s client"
	ErrorKEB       ErrorComponent = "keb"
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

	if status := dberr.Error(nil); errors.As(err, &status) {
		return LastError{
			error:     err,
			Reason:    status.Reason(),
			Component: ErrorDB,
		}
	}

	if status := apierr.APIStatus(nil); errors.As(err, &status) {
		return LastError{
			error:     err,
			Reason:    apierr.ReasonForError(status),
			Component: ErrorK8SClient,
		}
	}

	return LastError{
		error:     err,
		Reason:    ErrorKEBInternal,
		Component: ErrorKEB,
	}
}

/*
component should not be empty. only one confirmed serror component
if reason is empty, use err (not nil) to categorize the reason
after SetLastError, component and reason will be set unless both reason and err are empty
reason will be cleaned once a new component comes
*/
func SetLastError(lastErr *LastError, component ErrorComponent, reason ErrorReason, err error, log logrus.FieldLogger) {
	if lastErr == nil {
		log.Warn("last error pointer is nil")
		return
	}
	if component == "" {
		log.Warn("error component for last error cannot be empty")
		return
	}

	if lastErr.lastErrorExists() {
		// cleaned before each step run
		// only warning logged (no alert) in case of resuming, no intervere to the operation process
		log.WithFields(logrus.Fields{lastErr.Component: lastErr.error.Error()}).Warn("old last error is not cleared")
	}

	if component != lastErr.Component {
		// a new component with error
		lastErr.Component = component
		lastErr.Reasons = nil
	}

	// to categorize error reason
	if reason == "" {
		if err == nil {
			return
		}
		reason = categorizeErrorReason(component, errors.Cause(err))
	}

	// to be confirmed?
	if reason == ErrorKEBInternal {
		lastErr.Component = ErrorKEB
	}

	// lastErr.error = err
	for _, r := range lastErr.Reasons {
		if reason == r {
			return
		}
	}
	lastErr.Reasons = append(lastErr.Reasons, reason)
}

func categorizeErrorReason(component ErrorComponent, err error) ErrorReason {
	var reason ErrorReason

	switch component {
	case ErrorDB:
		switch {
		case dberr.IsNotFound(err):
			reason = ErrorDBNotFound
		case dberr.IsInternal(err):
			reason = ErrorDBInternal
		case dberr.IsConflict(err):
			reason = ErrorDBConflict
		case dberr.IsAlreadyExists(err):
			reason = ErrorDBAlreadyExists
		default:
			reason = ErrorDBUnknown
		}
	case ErrorK8SClient:
		reason = ErrorReason(apierr.ReasonForError(err))
		// StatusReasonUnknown
		if reason == "" {
			if e, ok := err.(apierr.APIStatus); !ok || e.Status().Code != 500 {
				reason = ErrorKEBInternal
			}
		}
	default:
		reason = ErrorOther
	}

	return reason
}

func Clean(err *LastError) {
	if err == nil {
		return
	}
	err.Reasons = []ErrorReason{}
	err.Component = ""
	err.error = nil
}
