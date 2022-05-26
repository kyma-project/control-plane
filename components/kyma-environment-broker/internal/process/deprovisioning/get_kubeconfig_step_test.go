package deprovisioning

import (
	"fmt"
	"testing"
	"time"

	kebError "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/error"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	shootResourceGroup = "shoots.core.gardener.cloud"
	instID             = "instance-1"
	runtimeID          = "runtime-1"
	shootName          = "abcdef"
)

func TestGetKubeconfigStep(t *testing.T) {
	t.Run("should proceed when Provisioner throws not found error", func(t *testing.T) {
		// given
		log := logrus.New()
		ms := storage.NewMemoryStorage()
		op := fixture.FixDeprovisioningOperation("deprovisioning-1", instID)
		op.ShootName = shootName
		fakeProvisionerClient := fakeProvisionerClientNotFoundErr{}

		step := NewGetKubeconfigStep(ms.Operations(), fakeProvisionerClient, k8sClientProvider)

		// when
		entry := log.WithFields(logrus.Fields{"step": "TEST"})
		_, _, err := step.Run(op, entry)

		// then
		assert.NoError(t, err)
	})

	t.Run("should retry when Provisioner throws error other than not found error", func(t *testing.T) {
		// given
		log := logrus.New()
		ms := storage.NewMemoryStorage()
		op := fixture.FixDeprovisioningOperation("deprovisioning-1", instID)
		op.ShootName = shootName
		fakeProvisionerClient := fakeProvisionerClientRetryableErr{}

		step := NewGetKubeconfigStep(ms.Operations(), fakeProvisionerClient, k8sClientProvider)

		// when
		entry := log.WithFields(logrus.Fields{"step": "TEST"})
		_, dur, err := step.Run(op, entry)

		// then
		assert.NoError(t, err)
		assert.Equal(t, time.Second*10, dur)
	})
}

type fakeProvisionerClient struct{}

type fakeProvisionerClientNotFoundErr struct {
	fakeProvisionerClient
}

type fakeProvisionerClientRetryableErr struct {
	fakeProvisionerClient
}

func (f fakeProvisionerClient) ProvisionRuntime(accountID, subAccountID string, config gqlschema.ProvisionRuntimeInput) (gqlschema.OperationStatus, error) {
	panic("not implemented")
}

func (f fakeProvisionerClient) DeprovisionRuntime(accountID, runtimeID string) (string, error) {
	panic("not implemented")
}

func (f fakeProvisionerClient) UpgradeRuntime(accountID, runtimeID string, config gqlschema.UpgradeRuntimeInput) (gqlschema.OperationStatus, error) {
	panic("not implemented")
}

func (f fakeProvisionerClient) UpgradeShoot(accountID, runtimeID string, config gqlschema.UpgradeShootInput) (gqlschema.OperationStatus, error) {
	panic("not implemented")
}

func (f fakeProvisionerClient) ReconnectRuntimeAgent(accountID, runtimeID string) (string, error) {
	panic("not implemented")
}

func (f fakeProvisionerClient) RuntimeOperationStatus(accountID, operationID string) (gqlschema.OperationStatus, error) {
	panic("not implemented")
}

func (f fakeProvisionerClient) RuntimeStatus(accountID, runtimeID string) (gqlschema.RuntimeStatus, error) {
	panic("not implemented")
}

func (f fakeProvisionerClientNotFoundErr) RuntimeStatus(accountID, runtimeID string) (gqlschema.RuntimeStatus, error) {
	err := fmt.Errorf("%s \"%s\" not found", shootResourceGroup, shootName)
	err = fmt.Errorf("error getting Shoot for cluster ID %s and name %s, %w", runtimeID, shootName, err)
	err = fmt.Errorf("failed to get Runtime Status, %w", err)

	return gqlschema.RuntimeStatus{}, kebError.WrapAsTemporaryError(err, "failed to execute the request")
}

func (f fakeProvisionerClientRetryableErr) RuntimeStatus(accountID, runtimeID string) (gqlschema.RuntimeStatus, error) {
	err := fmt.Errorf("RuntimeStatus retryable error")
	err = fmt.Errorf("error getting Shoot for cluster ID %s and name %s, %w", runtimeID, shootName, err)
	return gqlschema.RuntimeStatus{}, kebError.WrapAsTemporaryError(err, "failed to execute the request")
}

func k8sClientProvider(kcfg string) (client.Client, error) {
	k8sCli := fake.NewClientBuilder().Build()
	return k8sCli, nil
}
