package provisioning

import (
	"testing"
	"time"

	automock2 "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input/automock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/provisioning/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
)

const (
	statusOperationID            = "17f3ddba-1132-466d-a3c5-920f544d7ea6"
	statusInstanceID             = "9d75a545-2e1e-4786-abd8-a37b14e185b9"
	statusRuntimeID              = "ef4e3210-652c-453e-8015-bba1c1cd1e1c"
	statusGlobalAccountID        = "abf73c71-a653-4951-b9c2-a26d6c2cccbd"
	statusProvisionerOperationID = "e04de524-53b3-4890-b05a-296be393e4ba"

	dashboardURL               = "http://runtime.com"
	fixAvsEvaluationInternalId = int64(1234)
)

func TestInitialisationStep_Run(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	operation := fixOperationRuntimeStatus(broker.GCPPlanID, internal.GCP)
	st.Operations().InsertOperation(operation)
	st.Instances().Insert(fixture.FixInstance(operation.InstanceID))
	rvc := &automock.RuntimeVersionConfiguratorForProvisioning{}
	v := &internal.RuntimeVersionData{
		Version: "1.21.0",
		Origin:  internal.Defaults,
	}
	rvc.On("ForProvisioning", mock.Anything).Return(v, nil)
	ri := &simpleInputCreator{
		provider: internal.GCP,
	}
	builder := &automock2.CreatorForPlan{}
	builder.On("CreateProvisionInput", operation.ProvisioningParameters, *v).Return(ri, nil)

	step := NewInitialisationStep(st.Operations(), st.Instances(), builder, rvc)

	// when
	op, retry, err := step.Run(operation, logrus.New())

	// then
	assert.NoError(t, err)
	assert.Zero(t, retry)
	assert.Equal(t, *v, op.RuntimeVersion)
	assert.Equal(t, ri, op.InputCreator)

	inst, _ := st.Instances().GetByID(operation.InstanceID)
	// make sure the provider is saved into the instance
	assert.Equal(t, internal.GCP, inst.Provider)
}

func fixOperationRuntimeStatus(planId string, provider internal.CloudProvider) internal.Operation {
	provisioningOperation := fixture.FixProvisioningOperationWithProvider(statusOperationID, statusInstanceID, provider)
	provisioningOperation.State = domain.InProgress
	provisioningOperation.ProvisionerOperationID = statusProvisionerOperationID
	provisioningOperation.InstanceDetails.RuntimeID = runtimeID
	provisioningOperation.ProvisioningParameters.PlanID = planId
	provisioningOperation.ProvisioningParameters.ErsContext.GlobalAccountID = statusGlobalAccountID
	provisioningOperation.RuntimeVersion = internal.RuntimeVersionData{}

	return provisioningOperation
}

func fixOperationRuntimeStatusWithProvider(planId string, provider internal.CloudProvider) internal.Operation {
	provisioningOperation := fixture.FixProvisioningOperationWithProvider(statusOperationID, statusInstanceID, provider)
	provisioningOperation.State = ""
	provisioningOperation.ProvisionerOperationID = statusProvisionerOperationID
	provisioningOperation.ProvisioningParameters.PlanID = planId
	provisioningOperation.ProvisioningParameters.ErsContext.GlobalAccountID = statusGlobalAccountID
	provisioningOperation.ProvisioningParameters.Parameters.Provider = &provider

	return provisioningOperation
}

func fixInstanceRuntimeStatus() internal.Instance {
	instance := fixture.FixInstance(statusInstanceID)
	instance.RuntimeID = statusRuntimeID
	instance.DashboardURL = dashboardURL
	instance.GlobalAccountID = statusGlobalAccountID

	return instance
}

func fixAvsEvaluation() *avs.BasicEvaluationCreateResponse {
	return &avs.BasicEvaluationCreateResponse{
		DefinitionType:   "av",
		Name:             "fake-internal-eval",
		Description:      "",
		Service:          "",
		URL:              "",
		CheckType:        "",
		Interval:         180,
		TesterAccessId:   int64(999),
		Timeout:          30000,
		ReadOnly:         false,
		ContentCheck:     "",
		ContentCheckType: "",
		Threshold:        30000,
		GroupId:          int64(4321),
		Visibility:       "PUBLIC",
		DateCreated:      time.Now().Unix(),
		DateChanged:      time.Now().Unix(),
		Owner:            "johndoe@xyz.corp",
		Status:           "ACTIVE",
		Alerts:           nil,
		Tags: []*avs.Tag{
			{
				Content:      "already-exist-tag",
				TagClassId:   123456,
				TagClassName: "already-exist-tag-classname",
			},
		},
		Id:                         fixAvsEvaluationInternalId,
		LegacyCheckId:              fixAvsEvaluationInternalId,
		InternalInterval:           60,
		AuthType:                   "AUTH_NONE",
		IndividualOutageEventsOnly: false,
		IdOnTester:                 "",
	}
}
