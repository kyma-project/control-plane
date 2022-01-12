package broker

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"

	"github.com/stretchr/testify/require"
)

func TestResponseLabels(t *testing.T) {
	t.Run("return all labels", func(t *testing.T) {
		// given
		operation := internal.ProvisioningOperation{}
		operation.ProvisioningParameters.Parameters.Name = "test"

		instance := internal.Instance{InstanceID: "inst1234", DashboardURL: "https://dashbord.test"}

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
		labels := ResponseLabels(operation, instance, "https://example.com", false)

		// then
		require.Len(t, labels, 1)
		require.Equal(t, "test", labels["Name"])
	})
}
