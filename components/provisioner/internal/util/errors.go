package util

import (
	"errors"
	"testing"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"gotest.tools/assert"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

func K8SErrorToAppError(err error) apperrors.AppError {
	var apperr apperrors.AppError

	switch {
	case k8serrors.IsBadRequest(err):
		apperr = apperrors.BadRequest(err.Error())
	case k8serrors.IsForbidden(err):
		apperr = apperrors.Forbidden(err.Error())
	default:
		apperr = apperrors.Internal(err.Error())
	}

	return apperr.SetComponent(apperrors.ErrorK8SClient).SetReason(apperrors.ErrReason(k8serrors.ReasonForError(err)))
}

func DBErrorToAppError(err error) apperrors.AppError {
	var apperr apperrors.AppError

	switch {
	case k8serrors.IsBadRequest(err):
		apperr = apperrors.BadRequest(err.Error())
	case k8serrors.IsForbidden(err):
		apperr = apperrors.Forbidden(err.Error())
	default:
		apperr = apperrors.Internal(err.Error())
	}

	return apperr.SetComponent(apperrors.ErrorK8SClient).SetReason(apperrors.ErrReason(k8serrors.ReasonForError(err)))
}

func CheckErrorType(t *testing.T, err error, errCode apperrors.ErrCode) {
	var appErr apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fail()
	}
	assert.Equal(t, appErr.Code(), errCode)
}
