package provisioning

import (
	"errors"
	"testing"
	"time"

	"github.com/pivotal-cf/brokerapi/v8/domain"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler"
	hyperscalerMocks "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/hyperscaler/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestResolveCredentialsStepHappyPath_Run(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixOperationRuntimeStatus(broker.GCPPlanID, internal.GCP)
	err := memoryStorage.Operations().InsertProvisioningOperation(operation)
	assert.NoError(t, err)

	accountProviderMock := &hyperscalerMocks.AccountProvider{}
	accountProviderMock.On("GardenerCredentials", hyperscaler.GCP, statusGlobalAccountID).Return(hyperscaler.Credentials{
		Name:            "gardener-secret-gcp",
		HyperscalerType: "gcp",
		CredentialData:  map[string][]byte{},
	}, nil)

	step := NewResolveCredentialsStep(memoryStorage.Operations(), accountProviderMock)

	// when
	operation, repeat, err := step.Run(operation, log)

	assert.NoError(t, err)

	// then
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
	assert.Equal(t, domain.InProgress, operation.State)
	require.NotNil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
	assert.Equal(t, "gardener-secret-gcp", *operation.ProvisioningParameters.Parameters.TargetSecret)
}

func TestResolveCredentialsStepHappyPathTrialDefaultProvider_Run(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixOperationRuntimeStatus(broker.TrialPlanID, internal.Azure)
	err := memoryStorage.Operations().InsertProvisioningOperation(operation)
	assert.NoError(t, err)

	accountProviderMock := &hyperscalerMocks.AccountProvider{}
	accountProviderMock.On("GardenerSharedCredentials", hyperscaler.Azure).Return(hyperscaler.Credentials{
		Name:            "gardener-secret-azure",
		HyperscalerType: "azure",
		CredentialData:  map[string][]byte{},
	}, nil)

	step := NewResolveCredentialsStep(memoryStorage.Operations(), accountProviderMock)

	// when
	operation, repeat, err := step.Run(operation, log)

	assert.NoError(t, err)

	// then
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
	assert.Equal(t, domain.InProgress, operation.State)
	require.NotNil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
	assert.Equal(t, "gardener-secret-azure", *operation.ProvisioningParameters.Parameters.TargetSecret)
}

func TestResolveCredentialsStepHappyPathTrialGivenProvider_Run(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixOperationRuntimeStatusWithProvider(broker.TrialPlanID, internal.GCP)

	err := memoryStorage.Operations().InsertProvisioningOperation(operation)
	assert.NoError(t, err)

	accountProviderMock := &hyperscalerMocks.AccountProvider{}
	accountProviderMock.On("GardenerSharedCredentials", hyperscaler.GCP).Return(hyperscaler.Credentials{
		Name:            "gardener-secret-gcp",
		HyperscalerType: "gcp",
		CredentialData:  map[string][]byte{},
	}, nil)

	step := NewResolveCredentialsStep(memoryStorage.Operations(), accountProviderMock)

	// when
	operation, repeat, err := step.Run(operation, log)

	assert.NoError(t, err)

	// then
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), repeat)
	assert.Empty(t, operation.State)
	require.NotNil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
	assert.Equal(t, "gardener-secret-gcp", *operation.ProvisioningParameters.Parameters.TargetSecret)
}

func TestResolveCredentialsStepRetry_Run(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixOperationRuntimeStatus(broker.GCPPlanID, internal.GCP)
	err := memoryStorage.Operations().InsertProvisioningOperation(operation)
	assert.NoError(t, err)

	accountProviderMock := &hyperscalerMocks.AccountProvider{}
	accountProviderMock.On("GardenerCredentials", hyperscaler.GCP, statusGlobalAccountID).Return(hyperscaler.Credentials{}, errors.New("Failed!"))

	step := NewResolveCredentialsStep(memoryStorage.Operations(), accountProviderMock)

	operation.UpdatedAt = time.Now()

	// when
	operation, repeat, err := step.Run(operation, log)

	assert.NoError(t, err)

	// then
	assert.NoError(t, err)
	assert.Equal(t, 10*time.Second, repeat)
	assert.Nil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
	assert.Equal(t, domain.InProgress, operation.State)

	operation, repeat, err = step.Run(operation, log)

	assert.NoError(t, err)
	assert.Equal(t, 10*time.Second, repeat)
	assert.Equal(t, domain.InProgress, operation.State)
	assert.Nil(t, operation.ProvisioningParameters.Parameters.TargetSecret)
}
