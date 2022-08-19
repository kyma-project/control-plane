package broker

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestResponseLabels(t *testing.T) {
	t.Run("return all labels", func(t *testing.T) {
		// given
		operation := internal.ProvisioningOperation{}
		operation.ProvisioningParameters.Parameters.Name = "test"

		instance := internal.Instance{InstanceID: "inst1234", DashboardURL: "https://console.dashbord.test"}

		// when
		labels := ResponseLabels(operation, instance, "https://example.com", true)

		// then
		require.Len(t, labels, 2)
		require.Equal(t, "test", labels["Name"])
		require.Equal(t, "https://example.com/kubeconfig/inst1234", labels["KubeconfigURL"])
	})

	t.Run("disable kubeconfig URL label", func(t *testing.T) {
		// given
		operation := internal.ProvisioningOperation{}
		operation.ProvisioningParameters.Parameters.Name = "test"
		instance := internal.Instance{}

		// when
		labels := ResponseLabels(operation, instance, "https://console.example.com", false)

		// then
		require.Len(t, labels, 1)
		require.Equal(t, "test", labels["Name"])
	})

	t.Run("should return labels with expire info for not expired instance", func(t *testing.T) {
		// given
		operation := internal.ProvisioningOperation{}
		operation.ProvisioningParameters.Parameters.Name = "cluster-test"

		instance := fixture.FixInstance("instanceID")

		// when
		labels := ResponseLabelsWithExpireInfo(operation, instance, "https://example.com", true)

		// then
		require.Len(t, labels, 3)
		assert.Contains(t, labels, trialExpiryDetailsKey)
		require.Equal(t, "cluster-test", labels["Name"])
		require.Equal(t, "https://example.com/kubeconfig/instanceID", labels["KubeconfigURL"])
	})

	t.Run("should return labels with expire info for expired instance", func(t *testing.T) {
		// given
		operation := internal.ProvisioningOperation{}
		operation.ProvisioningParameters.Parameters.Name = "cluster-test"

		instance := fixture.FixInstance("instanceID")
		instance.CreatedAt = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
		expiryDate := time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)
		instance.ExpiredAt = &expiryDate

		// when
		labels := ResponseLabelsWithExpireInfo(operation, instance, "https://example.com", true)

		// then
		require.Len(t, labels, 2)
		assert.Contains(t, labels, trialExpiryDetailsKey)
		assert.NotContains(t, labels, kubeconfigURLKey)
		require.Equal(t, "cluster-test", labels["Name"])
	})
}
