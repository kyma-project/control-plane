package avs

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type EvalAssistant interface {
	CreateBasicEvaluationRequest(operations internal.ProvisioningOperation, url string) (*BasicEvaluationCreateRequest, error)
	IsAlreadyCreated(lifecycleData internal.AvsLifecycleData) bool
	IsValid(lifecycleData internal.AvsLifecycleData) bool
	IsInMaintenance(lifecycleData internal.AvsLifecycleData) bool
	SetEvalId(lifecycleData *internal.AvsLifecycleData, evalId int64)
	SetEvalStatus(lifecycleData *internal.AvsLifecycleData, status string)
	GetEvalStatus(lifecycleData internal.AvsLifecycleData) string
	GetOriginalEvalStatus(lifecycleData internal.AvsLifecycleData) string
	IsAlreadyDeleted(lifecycleData internal.AvsLifecycleData) bool
	GetEvaluationId(lifecycleData internal.AvsLifecycleData) int64
	ProvideParentId(pp internal.ProvisioningParameters) int64
	SetDeleted(lifecycleData *internal.AvsLifecycleData, deleted bool)
	provideRetryConfig() *RetryConfig
}

type RetryConfig struct {
	retryInterval time.Duration
	maxTime       time.Duration
}
