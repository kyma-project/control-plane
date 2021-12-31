package memory

import (
	"testing"
	"time"

	"github.com/google/uuid"
	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/stretchr/testify/assert"
)

func Test_runtimeState_GetLatestByRuntimeID(t *testing.T) {
	// given
	runtimeStates := NewRuntimeStates()

	olderRuntimeStateID := "older"
	newerRuntimeStateID := "newer"
	expectedRuntimeStateID := "expected"
	fixRuntimeID := "runtime1"

	olderRuntimeState := fixture.FixRuntimeState(olderRuntimeStateID, fixRuntimeID, uuid.NewString())
	olderRuntimeState.ClusterSetup = &reconcilerApi.Cluster{RuntimeID: fixRuntimeID}

	newerRuntimeState := fixture.FixRuntimeState(newerRuntimeStateID, fixRuntimeID, uuid.NewString())
	newerRuntimeState.ClusterSetup = &reconcilerApi.Cluster{RuntimeID: fixRuntimeID}
	newerRuntimeState.CreatedAt = newerRuntimeState.CreatedAt.Add(time.Hour * 1)

	expectedRuntimeState := fixture.FixRuntimeState(expectedRuntimeStateID, fixRuntimeID, uuid.NewString())
	expectedRuntimeState.ClusterSetup = &reconcilerApi.Cluster{RuntimeID: fixRuntimeID}
	expectedRuntimeState.CreatedAt = expectedRuntimeState.CreatedAt.Add(time.Hour * 2)

	runtimeStates.Insert(olderRuntimeState)
	runtimeStates.Insert(expectedRuntimeState)
	runtimeStates.Insert(newerRuntimeState)

	// when
	gotRuntimeState, _ := runtimeStates.GetLatestByRuntimeID(fixRuntimeID)

	// then
	assert.Equal(t, expectedRuntimeState.ID, gotRuntimeState.ID)
}

func Test_runtimeState_GetLatestWithReconcilerInputByRuntimeID(t *testing.T) {
	// given
	runtimeStates := NewRuntimeStates()

	olderRuntimeStateID := "older"
	newestRuntimeStateID := "newest"
	expectedRuntimeStateID := "expected"
	fixRuntimeID := "runtime1"

	olderRuntimeState := fixture.FixRuntimeState(olderRuntimeStateID, fixRuntimeID, uuid.NewString())
	olderRuntimeState.ClusterSetup = &reconcilerApi.Cluster{RuntimeID: fixRuntimeID}

	newestRuntimeStateWithoutReconcilerInput := fixture.FixRuntimeState(newestRuntimeStateID, fixRuntimeID, uuid.NewString())
	newestRuntimeStateWithoutReconcilerInput.CreatedAt = newestRuntimeStateWithoutReconcilerInput.CreatedAt.Add(time.Hour * 2)

	expectedRuntimeState := fixture.FixRuntimeState(expectedRuntimeStateID, fixRuntimeID, uuid.NewString())
	expectedRuntimeState.ClusterSetup = &reconcilerApi.Cluster{RuntimeID: fixRuntimeID}
	expectedRuntimeState.CreatedAt = expectedRuntimeState.CreatedAt.Add(time.Hour * 1)

	runtimeStates.Insert(olderRuntimeState)
	runtimeStates.Insert(expectedRuntimeState)
	runtimeStates.Insert(newestRuntimeStateWithoutReconcilerInput)

	// when
	gotRuntimeState, _ := runtimeStates.GetLatestWithReconcilerInputByRuntimeID(fixRuntimeID)

	// then
	assert.Equal(t, expectedRuntimeState.ID, gotRuntimeState.ID)
}
