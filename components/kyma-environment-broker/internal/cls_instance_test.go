package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCLSInstance(t *testing.T) {
	t.Run("should set default values", func(t *testing.T) {
		t.Parallel()

		i := NewCLSInstance("fake-global-account", "eu")
		require.Equal(t, "fake-global-account", i.GlobalAccountID())
		require.Equal(t, "eu", i.Region())
		require.Zero(t, i.Version())
		require.NotEmpty(t, i.ID())
		require.NotEmpty(t, i.CreatedAt())
		require.Empty(t, i.References())
		require.Empty(t, i.BeingRemovedBy())
	})

	t.Run("should override default values with opts", func(t *testing.T) {
		t.Parallel()

		time := time.Now().Add(-1 * time.Hour)
		i := NewCLSInstance("fake-global-account", "eu", WithVersion(42), WithID("fake-id"), WithCreatedAt(time), WithBeingRemovedBy("skr-1"))
		require.Equal(t, "fake-global-account", i.GlobalAccountID())
		require.Equal(t, "eu", i.Region())
		require.Equal(t, 42, i.Version())
		require.Equal(t, "fake-id", i.ID())
		require.Equal(t, time, i.CreatedAt())
		require.Equal(t, "skr-1", i.BeingRemovedBy())
	})

	t.Run("should manage references", func(t *testing.T) {
		t.Parallel()

		i := NewCLSInstance("fake-global-account", "eu", WithReferences("skr-1"))
		require.ElementsMatch(t, []string{"skr-1"}, i.References())

		i.AddReference("skr-2")
		require.ElementsMatch(t, []string{"skr-1", "skr-2"}, i.References())
		require.True(t, i.IsReferencedBy("skr-1"))
		require.True(t, i.IsReferencedBy("skr-2"))

		err := i.RemoveReference("skr-2")
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"skr-1"}, i.References())
		require.True(t, i.IsReferencedBy("skr-1"))
		require.False(t, i.IsReferencedBy("skr-2"))

		err = i.RemoveReference("skr-3")
		require.Error(t, err)
		require.ElementsMatch(t, []string{"skr-1"}, i.References())
		require.True(t, i.IsReferencedBy("skr-1"))
		require.False(t, i.IsReferencedBy("skr-2"))
	})

	t.Run("should track changed references", func(t *testing.T) {
		t.Parallel()

		i := NewCLSInstance("fake-global-account", "eu", WithReferences("skr-1"))
		require.Empty(t, i.Events())

		i.AddReference("skr-2")
		require.Len(t, i.Events(), 1)
		require.Equal(t, i.Events()[0].(CLSInstanceReferencedEvent).SKRInstanceID, "skr-2")

		err := i.RemoveReference("skr-2")
		require.NoError(t, err)
		require.Len(t, i.Events(), 2)
		require.Equal(t, i.Events()[0].(CLSInstanceReferencedEvent).SKRInstanceID, "skr-2")
		require.Equal(t, i.Events()[1].(CLSInstanceUnreferencedEvent).SKRInstanceID, "skr-2")

		err = i.RemoveReference("skr-3")
		require.Error(t, err)
		require.Len(t, i.Events(), 2)
		require.Equal(t, i.Events()[0].(CLSInstanceReferencedEvent).SKRInstanceID, "skr-2")
		require.Equal(t, i.Events()[1].(CLSInstanceUnreferencedEvent).SKRInstanceID, "skr-2")

		i.AddReference("skr-4")
		require.Len(t, i.Events(), 3)
		require.Equal(t, i.Events()[0].(CLSInstanceReferencedEvent).SKRInstanceID, "skr-2")
		require.Equal(t, i.Events()[1].(CLSInstanceUnreferencedEvent).SKRInstanceID, "skr-2")
		require.Equal(t, i.Events()[2].(CLSInstanceReferencedEvent).SKRInstanceID, "skr-4")
	})

	t.Run("should set bein removed by if last reference is removed", func(t *testing.T) {
		t.Parallel()

		i := NewCLSInstance("fake-global-account", "eu", WithReferences("skr-1"))
		require.False(t, i.IsBeingRemoved())

		err := i.RemoveReference("skr-1")
		require.NoError(t, err)
		require.True(t, i.IsBeingRemoved())
		require.Equal(t, "skr-1", i.BeingRemovedBy())
	})
}
