package error

import (
	"strings"

	gcli "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/third_party/machinebox/graphql"
	"github.com/pkg/errors"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	apierr2 "k8s.io/apimachinery/pkg/api/meta"
)

const OperationTimeOutMsg string = "operation has reached the time limit"

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
	ErrKEBInternal              ErrReason = "err_keb_internal"
	ErrKEBTimeOut               ErrReason = "err_keb_timeout"
	ErrProvisionerNilLastError  ErrReason = "err_provisioner_nil_last_error"
	ErrHttpStatusCode           ErrReason = "err_http_status_code"
	ErrReconcilerNilFailures    ErrReason = "err_reconciler_nil_failures"
	ErrClusterNotFound          ErrReason = "err_cluster_not_found"
	ErrK8SUnexpectedServerError ErrReason = "err_k8s_unexpected_server_error"
	ErrK8SUnexpectedObjectError ErrReason = "err_k8s_unexpected_object_error"
	ErrK8SNoMatchError          ErrReason = "err_k8s_no_match_error"
	ErrK8SAmbiguousError        ErrReason = "err_k8s_ambiguous_error"
)

type ErrComponent string

const (
	ErrDB          ErrComponent = "db - keb"
	ErrK8SClient   ErrComponent = "k8s client - keb"
	ErrKEB         ErrComponent = "keb"
	ErrEDP         ErrComponent = "edp"
	ErrProvisioner ErrComponent = "provisioner"
	ErrReconciler  ErrComponent = "reconciler"
	ErrAVS         ErrComponent = "avs"
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

func (err LastError) SetComponent(component ErrComponent) LastError {
	err.component = component
	return err
}

func (err LastError) SetReason(reason ErrReason) LastError {
	err.reason = reason
	return err
}

func (err LastError) SetMessage(msg string) LastError {
	err.message = msg
	return err
}

func TimeoutError(msg string) LastError {
	return LastError{
		message:   msg,
		reason:    ErrKEBTimeOut,
		component: ErrKEB,
	}
}

// resolve error component and reason
func ReasonForError(err error) LastError {
	if err == nil {
		return LastError{}
	}

	cause := errors.Cause(err)

	if lastErr := checkK8SError(cause); lastErr.component == ErrK8SClient {
		lastErr.message = err.Error()
		return lastErr
	}

	if status := ErrorReporter(nil); errors.As(cause, &status) {
		return LastError{
			message:   err.Error(),
			reason:    status.Reason(),
			component: status.Component(),
		}
	}

	if ee, ok := cause.(gcli.ExtendedError); ok {
		var errReason ErrReason
		var errComponent ErrComponent

		reason, found := ee.Extensions()["error_reason"]
		if found {
			if r, ok := reason.(string); ok {
				errReason = ErrReason(r)
			}
		}
		component, found := ee.Extensions()["error_component"]
		if found {
			if c, ok := component.(string); ok {
				errComponent = ErrComponent(c)
			}
		}

		return LastError{
			message:   err.Error(),
			reason:    errReason,
			component: errComponent,
		}
	}

	if strings.Contains(err.Error(), OperationTimeOutMsg) {
		return TimeoutError(err.Error())
	}

	return LastError{
		message:   err.Error(),
		reason:    ErrKEBInternal,
		component: ErrKEB,
	}
}

func checkK8SError(cause error) LastError {
	lastErr := LastError{}
	status := apierr.APIStatus(nil)

	switch {
	case errors.As(cause, &status):
		if apierr.IsUnexpectedServerError(cause) {
			lastErr.reason = ErrK8SUnexpectedServerError
		} else {
			// reason could be an empty unknown ""
			lastErr.reason = ErrReason(apierr.ReasonForError(cause))
		}
		lastErr.component = ErrK8SClient
		return lastErr
	case apierr.IsUnexpectedObjectError(cause):
		lastErr.reason = ErrK8SUnexpectedObjectError
	case apierr2.IsAmbiguousError(cause):
		lastErr.reason = ErrK8SAmbiguousError
	case apierr2.IsNoMatchError(cause):
		lastErr.reason = ErrK8SNoMatchError
	}

	if lastErr.reason != "" {
		lastErr.component = ErrK8SClient
	}

	return lastErr
}
