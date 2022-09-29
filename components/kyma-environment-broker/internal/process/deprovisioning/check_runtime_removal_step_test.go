package deprovisioning

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckRuntimeRemovalStep(t *testing.T) {
	for _, tc := range []struct {
		givenState       gqlschema.OperationState
		expectedDuration bool
	}{
		{givenState: gqlschema.OperationStatePending, expectedDuration: true},
		{givenState: gqlschema.OperationStateInProgress, expectedDuration: true},
		{givenState: gqlschema.OperationStateSucceeded, expectedDuration: false},
	} {
		t.Run(string(tc.givenState), func(t *testing.T) {
			// given
			log := logrus.New()
			memoryStorage := storage.NewMemoryStorage()
			provisionerClient := provisioner.NewFakeClient()
			svc := NewCheckRuntimeRemovalStep(memoryStorage.Operations(), memoryStorage.Instances(), provisionerClient)
			dOp := fixDeprovisioningOperation().Operation
			memoryStorage.Instances().Insert(internal.Instance{
				GlobalAccountID: "global-acc",
				InstanceID:      dOp.InstanceID,
			})
			provisionerOp, _ := provisionerClient.DeprovisionRuntime(dOp.GlobalAccountID, dOp.RuntimeID)
			provisionerClient.FinishProvisionerOperation(provisionerOp, tc.givenState)
			dOp.ProvisionerOperationID = provisionerOp

			// when
			_, d, err := svc.Run(dOp, log)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.expectedDuration, d > 0)
		})
	}
}

func TestCheckRuntimeRemovalStep_ProvisionerFailed(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()
	provisionerClient := provisioner.NewFakeClient()
	svc := NewCheckRuntimeRemovalStep(memoryStorage.Operations(), memoryStorage.Instances(), provisionerClient)
	dOp := fixDeprovisioningOperation().Operation
	memoryStorage.Operations().InsertOperation(dOp)
	memoryStorage.Instances().Insert(internal.Instance{
		GlobalAccountID: "global-acc",
		InstanceID:      dOp.InstanceID,
	})
	provisionerOp, _ := provisionerClient.DeprovisionRuntime(dOp.GlobalAccountID, dOp.RuntimeID)
	provisionerClient.FinishProvisionerOperation(provisionerOp, gqlschema.OperationStateFailed)
	dOp.ProvisionerOperationID = provisionerOp

	// when
	op, _, err := svc.Run(dOp, log)

	// then
	require.Error(t, err)
	assert.Equal(t, domain.Failed, op.State)
}

func TestCheckRuntimeRemovalStep_InstanceDeleted(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()
	provisionerClient := provisioner.NewFakeClient()
	svc := NewCheckRuntimeRemovalStep(memoryStorage.Operations(), memoryStorage.Instances(), provisionerClient)
	dOp := fixDeprovisioningOperation().Operation
	memoryStorage.Operations().InsertOperation(dOp)

	// when
	_, backoff, err := svc.Run(dOp, log)

	// then
	require.NoError(t, err)
	assert.Zero(t, backoff)
}
