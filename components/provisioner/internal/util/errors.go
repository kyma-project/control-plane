package util

import (
	"strings"
	"testing"

	"github.com/pkg/errors"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"gotest.tools/assert"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

func K8SErrorToAppError(err error) apperrors.AppError {
	var apperr apperrors.AppError
	cause := errors.Cause(err)

	switch {
	case k8serrors.IsBadRequest(cause):
		apperr = apperrors.BadRequest(err.Error())
	case k8serrors.IsForbidden(cause):
		apperr = apperrors.Forbidden(err.Error())
	default:
		apperr = apperrors.Internal(err.Error())
	}

	return apperr.SetComponent(apperrors.ErrClusterK8SClient).SetReason(apperrors.ErrReason(k8serrors.ReasonForError(cause)))
}

func GardenerErrCodesToErrReason(lastErrors ...gardencorev1beta1.LastError) apperrors.ErrReason {
	var codes []gardencorev1beta1.ErrorCode
	var vals []string

	for _, e := range lastErrors {
		if len(e.Codes) > 0 {
			codes = append(codes, e.Codes...)
		}
	}

	for _, code := range codes {
		vals = append(vals, string(code))
	}

	return apperrors.ErrReason(strings.Join(vals, ", "))
}

func CheckErrorType(t *testing.T, err error, errCode apperrors.ErrCode) {
	var appErr apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fail()
	}
	assert.Equal(t, appErr.Code(), errCode)
}
