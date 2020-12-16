package avs

import (
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

const externalEvalCheckType = "HTTPSGET"

type ExternalEvalAssistant struct {
	avsConfig   Config
	retryConfig *RetryConfig
}

func NewExternalEvalAssistant(avsConfig Config) *ExternalEvalAssistant {
	return &ExternalEvalAssistant{
		avsConfig:   avsConfig,
		retryConfig: &RetryConfig{maxTime: 120 * time.Minute, retryInterval: 20 * time.Second},
	}
}

func (eea *ExternalEvalAssistant) CreateBasicEvaluationRequest(operations internal.ProvisioningOperation, url string) (*BasicEvaluationCreateRequest, error) {
	return newBasicEvaluationCreateRequest(operations, eea, url)
}

func (eea *ExternalEvalAssistant) AppendOverrides(inputCreator internal.ProvisionerInputCreator, evaluationId int64, _ internal.ProvisioningParameters) {
	//do nothing
}

func (eea *ExternalEvalAssistant) IsAlreadyCreated(lifecycleData internal.AvsLifecycleData) bool {
	return lifecycleData.AVSEvaluationExternalId != 0
}

func (eea *ExternalEvalAssistant) ProvideSuffix() string {
	return "ext"
}

func (eea *ExternalEvalAssistant) ProvideTesterAccessId(_ internal.ProvisioningParameters) int64 {
	return eea.avsConfig.ExternalTesterAccessId
}

func (eea *ExternalEvalAssistant) ProvideGroupId(_ internal.ProvisioningParameters) int64 {
	return eea.avsConfig.GroupId
}

func (eea *ExternalEvalAssistant) ProvideParentId(_ internal.ProvisioningParameters) int64 {
	return eea.avsConfig.ParentId
}

func (eea *ExternalEvalAssistant) ProvideTags() []*Tag {
	return eea.avsConfig.ExternalTesterTags
}

func (eea *ExternalEvalAssistant) ProvideNewOrDefaultServiceName(defaultServiceName string) string {
	if eea.avsConfig.ExternalTesterService == "" {
		return defaultServiceName
	}
	return eea.avsConfig.ExternalTesterService
}

func (eea *ExternalEvalAssistant) SetEvalId(lifecycleData *internal.AvsLifecycleData, evalId int64) {
	lifecycleData.AVSEvaluationExternalId = evalId
}

func (eea *ExternalEvalAssistant) ProvideCheckType() string {
	return externalEvalCheckType
}

func (eea *ExternalEvalAssistant) IsAlreadyDeleted(lifecycleData internal.AvsLifecycleData) bool {
	return lifecycleData.AVSExternalEvaluationDeleted
}

func (eea *ExternalEvalAssistant) GetEvaluationId(lifecycleData internal.AvsLifecycleData) int64 {
	return lifecycleData.AVSEvaluationExternalId
}

func (eea *ExternalEvalAssistant) markDeleted(lifecycleData *internal.AvsLifecycleData) {
	lifecycleData.AVSExternalEvaluationDeleted = true
}

func (eea *ExternalEvalAssistant) provideRetryConfig() *RetryConfig {
	return eea.retryConfig
}
