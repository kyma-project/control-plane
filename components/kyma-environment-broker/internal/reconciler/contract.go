package reconciler

import (
	"fmt"
	"strings"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
)

type reconcilerError struct {
	components []string
	message    string
}

func (e reconcilerError) Error() string {
	return e.message
}

func (e reconcilerError) Component() kebError.ErrComponent {
	return kebError.ErrReconciler
}

func (e reconcilerError) Reason() kebError.ErrReason {
	if e.components == nil {
		return kebError.ErrReconcilerNilFailures
	}

	return kebError.ErrReason(strings.Join(e.components, ", "))
}

func errorf(components []string, format string, args ...interface{}) kebError.ErrorReporter {
	return reconcilerError{components: components, message: fmt.Sprintf(format, args...)}
}

func NewReconcilerError(failures *[]reconcilerApi.Failure, format string, args ...interface{}) kebError.ErrorReporter {
	var components []string

	if failures == nil {
		return errorf(nil, format, args...)
	}

	for _, f := range *failures {
		components = append(components, f.Component)
	}

	return errorf(components, format, args...)
}

func PrettyFailures(response *reconcilerApi.HTTPClusterResponse) string {
	var errs []string
	failures := response.Failures

	if failures == nil {
		return ""
	}

	for _, f := range *failures {
		errs = append(errs, fmt.Sprintf("component %v failed: %v", f.Component, f.Reason))
	}
	return strings.Join(errs, ", ")
}
