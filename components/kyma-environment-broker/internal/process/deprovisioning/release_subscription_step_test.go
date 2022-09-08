package deprovisioning

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	hyperscalerMocks "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestReleaseSubscriptionStepHappyPath_Run(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.GCPPlanID)
	instance := fixGCPInstance(operation.InstanceID)

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	accountProviderMock := &hyperscalerMocks.AccountProvider{}
	accountProviderMock.On("MarkUnusedGardenerSecretBindingAsDirty", hyperscaler.GCP, instance.GetSubscriptionGlobalAccoundID()).Return(nil)

	step := NewReleaseSubscriptionStep(memoryStorage.Instances(), accountProviderMock)

	// when
	operation, repeat, err := step.Run(operation, log)

	assert.NoError(t, err)

	// then
	accountProviderMock.AssertNumberOfCalls(t, "MarkUnusedGardenerSecretBindingAsDirty", 1)
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
	assert.Equal(t, domain.Succeeded, operation.State)
}

func TestReleaseSubscriptionStepForTrial_Run(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.TrialPlanID)
	instance := fixGCPInstance(operation.InstanceID)

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	accountProviderMock := &hyperscalerMocks.AccountProvider{}
	accountProviderMock.On("MarkUnusedGardenerSecretBindingAsDirty", hyperscaler.GCP, instance.GetSubscriptionGlobalAccoundID()).Return(nil)

	step := NewReleaseSubscriptionStep(memoryStorage.Instances(), accountProviderMock)

	// when
	operation, repeat, err := step.Run(operation, log)

	assert.NoError(t, err)

	// then
	accountProviderMock.AssertNotCalled(t, "MarkUnusedGardenerSecretBindingAsDirty")
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
	assert.Equal(t, domain.Succeeded, operation.State)
}

func TestReleaseSubscriptionStepInstanceNotFound_Run(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.GCPPlanID)
	instance := fixGCPInstance(operation.InstanceID)

	accountProviderMock := &hyperscalerMocks.AccountProvider{}
	accountProviderMock.On("MarkUnusedGardenerSecretBindingAsDirty", hyperscaler.GCP, instance.GetSubscriptionGlobalAccoundID()).Return(nil)

	step := NewReleaseSubscriptionStep(memoryStorage.Instances(), accountProviderMock)

	// when
	operation, repeat, err := step.Run(operation, log)

	assert.NoError(t, err)

	// then
	accountProviderMock.AssertNotCalled(t, "MarkUnusedGardenerSecretBindingAsDirty")
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
	assert.Equal(t, domain.Succeeded, operation.State)
}

func TestReleaseSubscriptionStepProviderNotFound_Run(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.GCPPlanID)
	instance := fixGCPInstance(operation.InstanceID)
	instance.Provider = "unknown"

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	accountProviderMock := &hyperscalerMocks.AccountProvider{}
	accountProviderMock.On("MarkUnusedGardenerSecretBindingAsDirty", hyperscaler.GCP, instance.GetSubscriptionGlobalAccoundID()).Return(nil)

	step := NewReleaseSubscriptionStep(memoryStorage.Instances(), accountProviderMock)

	// when
	operation, repeat, err := step.Run(operation, log)

	assert.NoError(t, err)

	// then
	accountProviderMock.AssertNotCalled(t, "MarkUnusedGardenerSecretBindingAsDirty")
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
	assert.Equal(t, domain.Succeeded, operation.State)
}

func TestReleaseSubscriptionStepGardenerCallFails_Run(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixDeprovisioningOperationWithPlanID(broker.GCPPlanID)
	instance := fixGCPInstance(operation.InstanceID)

	err := memoryStorage.Instances().Insert(instance)
	assert.NoError(t, err)

	accountProviderMock := &hyperscalerMocks.AccountProvider{}
	accountProviderMock.On("MarkUnusedGardenerSecretBindingAsDirty", hyperscaler.GCP, instance.GetSubscriptionGlobalAccoundID()).Return(errors.New("failed to release subscription for tenant. Gardener Account pool is not configured"))

	step := NewReleaseSubscriptionStep(memoryStorage.Instances(), accountProviderMock)

	// when
	operation, repeat, err := step.Run(operation, log)

	assert.NoError(t, err)

	// then
	assert.NoError(t, err)
	assert.Equal(t, 10*time.Second, repeat)
	assert.Equal(t, domain.Succeeded, operation.State)
}

func fixGCPInstance(instanceID string) internal.Instance {
	instance := fixture.FixInstance(instanceID)
	instance.Provider = "GCP"
	return instance
}

func fixDeprovisioningOperationWithPlanID(planID string) internal.Operation {
	deprovisioningOperation := fixture.FixDeprovisioningOperationAsOperation(operationID, instanceID)
	deprovisioningOperation.ProvisioningParameters.PlanID = planID
	deprovisioningOperation.ProvisioningParameters.ErsContext.GlobalAccountID = globalAccountID
	deprovisioningOperation.ProvisioningParameters.ErsContext.SubAccountID = subAccountID
	return deprovisioningOperation
}
