package installation

import (
	"testing"

	"github.com/kyma-incubator/hydroform/parallel-install/pkg/components"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/deployment"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/stretchr/testify/require"
)

func TestComponentsStatus_ConsumeEvent(t *testing.T) {
	cmps := []model.KymaComponentConfig{
		{
			Component: "A",
			Namespace: "test",
		},
		{
			Component: "B",
			Namespace: "test",
		},
		{
			Component: "C",
			Namespace: "test",
		},
	}

	t.Run("successful installation", func(t *testing.T) {
		// given
		cs := NewComponentsStatus(cmps)

		// when
		cs.ConsumeEvent(deployment.ProcessUpdate{
			Event:     deployment.ProcessFinished,
			Component: components.KymaComponent{Name: "A", Namespace: "test", Status: components.StatusInstalled},
		})

		// then
		require.False(t, cs.IsFinished())
		require.Equal(t, "1 of 3 components installed", cs.StatusDescription())
		cs.ConsumeEvent(deployment.ProcessUpdate{
			Event:     deployment.ProcessFinished,
			Component: components.KymaComponent{Name: "B", Namespace: "test", Status: components.StatusInstalled},
		})
		require.False(t, cs.IsFinished())
		require.Equal(t, "2 of 3 components installed", cs.StatusDescription())
		cs.ConsumeEvent(deployment.ProcessUpdate{
			Event:     deployment.ProcessFinished,
			Component: components.KymaComponent{Name: "C", Namespace: "test", Status: components.StatusInstalled},
		})
		require.True(t, cs.IsFinished())
		require.Equal(t, "3 of 3 components installed", cs.StatusDescription())
	})

	t.Run("installation failed", func(t *testing.T) {
		// given
		cs := NewComponentsStatus(cmps)

		// when
		cs.ConsumeEvent(deployment.ProcessUpdate{
			Event:     deployment.ProcessRunning,
			Component: components.KymaComponent{Name: "A", Namespace: "test", Status: components.StatusError},
		})

		// then
		require.False(t, cs.IsFinished())
		require.Error(t, cs.ComponentError())
		require.NoError(t, cs.InstallationError())

		cs.ConsumeEvent(deployment.ProcessUpdate{
			Event:     deployment.ProcessExecutionFailure,
			Component: components.KymaComponent{Name: "A", Namespace: "test", Status: components.StatusError},
		})

		require.False(t, cs.IsFinished())
		require.Error(t, cs.ComponentError())
		require.Error(t, cs.InstallationError())
	})
}
