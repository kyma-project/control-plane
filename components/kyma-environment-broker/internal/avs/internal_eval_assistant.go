package avs

import (
	"strconv"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

const (
	EvaluationIdKey = "avs_bridge.config.evaluations.cluster.id"
	AvsBridgeAPIKey = "avs_bridge.config.availabilityService.apiKey"
	ComponentName   = "avs-bridge"
)

type InternalEvalAssistant struct {
	avsConfig   Config
	retryConfig *RetryConfig
}

func NewInternalEvalAssistant(avsConfig Config) *InternalEvalAssistant {
	return &InternalEvalAssistant{
		avsConfig:   avsConfig,
		retryConfig: &RetryConfig{maxTime: 10 * time.Minute, retryInterval: 1 * time.Minute},
	}
}

func (iec *InternalEvalAssistant) CreateBasicEvaluationRequest(operations internal.ProvisioningOperation, url string) (*BasicEvaluationCreateRequest, error) {
	return newBasicEvaluationCreateRequest(operations, iec, url)
}

func (iec *InternalEvalAssistant) AppendOverrides(inputCreator internal.ProvisionerInputCreator, evaluationId int64, pp internal.ProvisioningParameters) {
	apiKey := iec.avsConfig.ApiKey
	if broker.IsTrialPlan(pp.PlanID) && iec.avsConfig.IsTrialConfigured() {
		apiKey = iec.avsConfig.TrialApiKey
	}
	inputCreator.AppendOverrides(ComponentName, []*gqlschema.ConfigEntryInput{
		{
			Key:   EvaluationIdKey,
			Value: strconv.FormatInt(evaluationId, 10),
		},
		{
			Key:   AvsBridgeAPIKey,
			Value: apiKey,
		},
	})
}

func (iec *InternalEvalAssistant) IsAlreadyCreated(lifecycleData internal.AvsLifecycleData) bool {
	return lifecycleData.AvsEvaluationInternalId != 0
}

func (iec *InternalEvalAssistant) ProvideSuffix() string {
	return "int"
}

func (iec *InternalEvalAssistant) ProvideTesterAccessId(pp internal.ProvisioningParameters) int64 {
	if broker.IsTrialPlan(pp.PlanID) && iec.avsConfig.IsTrialConfigured() {
		return iec.avsConfig.TrialInternalTesterAccessId
	}
	return iec.avsConfig.InternalTesterAccessId
}

func (iec *InternalEvalAssistant) ProvideGroupId(pp internal.ProvisioningParameters) int64 {
	if broker.IsTrialPlan(pp.PlanID) && iec.avsConfig.IsTrialConfigured() {
		return iec.avsConfig.TrialGroupId
	}
	return iec.avsConfig.GroupId
}

func (iec *InternalEvalAssistant) ProvideParentId(pp internal.ProvisioningParameters) int64 {
	if broker.IsTrialPlan(pp.PlanID) && iec.avsConfig.IsTrialConfigured() {
		return iec.avsConfig.TrialParentId
	}
	return iec.avsConfig.ParentId
}

func (iec *InternalEvalAssistant) ProvideCheckType() string {
	return ""
}

func (iec *InternalEvalAssistant) ProvideTags() []*Tag {
	return iec.avsConfig.InternalTesterTags
}

func (iec *InternalEvalAssistant) ProvideNewOrDefaultServiceName(defaultServiceName string) string {
	if iec.avsConfig.InternalTesterService == "" {
		return defaultServiceName
	}
	return iec.avsConfig.InternalTesterService
}

func (iec *InternalEvalAssistant) SetEvalId(lifecycleData *internal.AvsLifecycleData, evalId int64) {
	lifecycleData.AvsEvaluationInternalId = evalId
}

func (iec *InternalEvalAssistant) IsAlreadyDeleted(lifecycleData internal.AvsLifecycleData) bool {
	return lifecycleData.AVSInternalEvaluationDeleted
}

func (iec *InternalEvalAssistant) GetEvaluationId(lifecycleData internal.AvsLifecycleData) int64 {
	return lifecycleData.AvsEvaluationInternalId
}

func (iec *InternalEvalAssistant) markDeleted(lifecycleData *internal.AvsLifecycleData) {
	lifecycleData.AVSInternalEvaluationDeleted = true
}

func (iec *InternalEvalAssistant) provideRetryConfig() *RetryConfig {
	return iec.retryConfig
}
