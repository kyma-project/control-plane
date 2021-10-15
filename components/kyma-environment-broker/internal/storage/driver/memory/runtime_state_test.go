package memory

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/assert"
)

func Test_runtimeState_GetLastByRuntimeID(t *testing.T) {
	// given
	runtimeStates := NewRuntimeStates()

	olderRuntimeStateID := "older"
	newerRuntimeStateID := "newer"
	expectedRuntimeStateID := "expected"
	fixRuntimeID := "runtime1"

	olderRuntimeState := internal.RuntimeState{
		ID:          olderRuntimeStateID,
		CreatedAt:   time.Now(),
		RuntimeID:   fixRuntimeID,
		OperationID: olderRuntimeStateID,
		KymaConfig: gqlschema.KymaConfigInput{
			Version: olderRuntimeStateID,
		},
		ClusterConfig: gqlschema.GardenerConfigInput{
			KubernetesVersion: olderRuntimeStateID,
		},
	}

	newerRuntimeState := internal.RuntimeState{
		ID:          newerRuntimeStateID,
		CreatedAt:   time.Now().Add(time.Hour * 1),
		RuntimeID:   fixRuntimeID,
		OperationID: newerRuntimeStateID,
		KymaConfig: gqlschema.KymaConfigInput{
			Version: newerRuntimeStateID,
		},
		ClusterConfig: gqlschema.GardenerConfigInput{
			KubernetesVersion: newerRuntimeStateID,
		},
	}

	expectedRuntimeState := internal.RuntimeState{
		ID:          expectedRuntimeStateID,
		CreatedAt:   time.Now().Add(time.Hour * 2),
		RuntimeID:   fixRuntimeID,
		OperationID: expectedRuntimeStateID,
		KymaConfig: gqlschema.KymaConfigInput{
			Version: expectedRuntimeStateID,
		},
		ClusterConfig: gqlschema.GardenerConfigInput{
			KubernetesVersion: expectedRuntimeStateID,
		},
	}

	runtimeStates.Insert(olderRuntimeState)
	runtimeStates.Insert(expectedRuntimeState)
	runtimeStates.Insert(newerRuntimeState)

	// when
	gotRuntimeState, _ := runtimeStates.GetLastByRuntimeID(fixRuntimeID)

	// then
	assert.Equal(t, expectedRuntimeState.ID, gotRuntimeState.ID)
}
