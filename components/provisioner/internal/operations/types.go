package operations

import (
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ProcessingResult struct {
	Requeue bool
	Delay   time.Duration
}

type Step interface {
	Name() model.OperationStage
	Run(cluster model.Cluster, operation model.Operation, logger logrus.FieldLogger) (StageResult, error)
	TimeLimit() time.Duration
}

type StageResult struct {
	Stage model.OperationStage
	Delay time.Duration
}

type NonRecoverableError struct {
	error error
}

func (r NonRecoverableError) Error() string {
	return r.error.Error()
}

func NewNonRecoverableError(err error) NonRecoverableError {
	return NonRecoverableError{error: err}
}

type FailureHandler interface {
	HandleFailure(operation model.Operation, cluster model.Cluster) error
}

func ConvertToAppError(err error) apperrors.AppError {
	if nonRecoverErr := (NonRecoverableError{}); errors.As(err, &nonRecoverErr) {
		err = nonRecoverErr.error
	}

	if customErr := apperrors.AppError(nil); errors.As(errors.Cause(err), &customErr) {
		return customErr
	}

	return apperrors.Internal(err.Error())
}
