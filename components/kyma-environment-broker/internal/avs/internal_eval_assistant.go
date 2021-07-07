package avs

import (
	"strconv"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
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
		retryConfig: &RetryConfig{maxTime: 10 * time.Minute, retryInterval: 30 * time.Second},
	}
}

func (iec *InternalEvalAssistant) CreateBasicEvaluationRequest(operations internal.ProvisioningOperation, url string) (*BasicEvaluationCreateRequest, error) {
	return newBasicEvaluationCreateRequest(operations, iec, url)
}

func (iec *InternalEvalAssistant) AppendOverrides(inputCreator internal.ProvisionerInputCreator, evaluationId int64, pp internal.ProvisioningParameters) {
	apiKey := iec.avsConfig.ApiKey
	if (broker.IsTrialPlan(pp.PlanID) || broker.IsFreemiumPlan(pp.PlanID)) && iec.avsConfig.IsTrialConfigured() {
		apiKey = iec.avsConfig.TrialApiKey
	}
	inputCreator.AppendOverrides(ComponentName, []*gqlschema.ConfigEntryInput{
		{
			Key:   EvaluationIdKey,
			Value: strconv.FormatInt(evaluationId, 10),
		},
		{
			Key:    AvsBridgeAPIKey,
			Value:  apiKey,
			Secret: ptr.Bool(true),
		},
	})
}

func (iec *InternalEvalAssistant) IsAlreadyCreated(lifecycleData internal.AvsLifecycleData) bool {
	return lifecycleData.AvsEvaluationInternalId != 0
}

func (iec *InternalEvalAssistant) IsValid(lifecycleData internal.AvsLifecycleData) bool {
	return iec.IsAlreadyCreated(lifecycleData) && !iec.IsAlreadyDeleted(lifecycleData)
}

func (iec *InternalEvalAssistant) ProvideSuffix() string {
	return "int"
}

func (iec *InternalEvalAssistant) ProvideTesterAccessId(pp internal.ProvisioningParameters) int64 {
	if (broker.IsTrialPlan(pp.PlanID) || broker.IsFreemiumPlan(pp.PlanID)) && iec.avsConfig.IsTrialConfigured() {
		return iec.avsConfig.TrialInternalTesterAccessId
	}
	return iec.avsConfig.InternalTesterAccessId
}

func (iec *InternalEvalAssistant) ProvideGroupId(pp internal.ProvisioningParameters) int64 {
	if (broker.IsTrialPlan(pp.PlanID) || broker.IsFreemiumPlan(pp.PlanID)) && iec.avsConfig.IsTrialConfigured() {
		return iec.avsConfig.TrialGroupId
	}
	return iec.avsConfig.GroupId
}

func (iec *InternalEvalAssistant) ProvideParentId(pp internal.ProvisioningParameters) int64 {
	if (broker.IsTrialPlan(pp.PlanID) || broker.IsFreemiumPlan(pp.PlanID)) && iec.avsConfig.IsTrialConfigured() {
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

func (iec *InternalEvalAssistant) SetEvalStatus(lifecycleData *internal.AvsLifecycleData, status string) {
	current := lifecycleData.AvsInternalEvaluationStatus.Current
	if ValidStatus(current) {
		lifecycleData.AvsInternalEvaluationStatus.Original = current
	}
	lifecycleData.AvsInternalEvaluationStatus.Current = status
}

func (iec *InternalEvalAssistant) GetEvalStatus(lifecycleData internal.AvsLifecycleData) string {
	return lifecycleData.AvsInternalEvaluationStatus.Current
}

func (iec *InternalEvalAssistant) GetOriginalEvalStatus(lifecycleData internal.AvsLifecycleData) string {
	return lifecycleData.AvsInternalEvaluationStatus.Original
}

func (iec *InternalEvalAssistant) IsInMaintenance(lifecycleData internal.AvsLifecycleData) bool {
	return lifecycleData.AvsInternalEvaluationStatus.Current == StatusMaintenance
}

func (iec *InternalEvalAssistant) IsAlreadyDeleted(lifecycleData internal.AvsLifecycleData) bool {
	return lifecycleData.AVSInternalEvaluationDeleted
}

func (iec *InternalEvalAssistant) GetEvaluationId(lifecycleData internal.AvsLifecycleData) int64 {
	return lifecycleData.AvsEvaluationInternalId
}

func (iec *InternalEvalAssistant) SetDeleted(lifecycleData *internal.AvsLifecycleData, deleted bool) {
	lifecycleData.AVSInternalEvaluationDeleted = deleted
}

func (iec *InternalEvalAssistant) provideRetryConfig() *RetryConfig {
	return iec.retryConfig
}
