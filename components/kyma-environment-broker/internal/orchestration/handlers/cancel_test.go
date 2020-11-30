package handlers

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	fixOrchestrationID = "test-id"
)

func TestCanceler_Cancel(t *testing.T) {
	t.Run("should cancel existing orchestration", func(t *testing.T) {
		s := storage.NewMemoryStorage()
		err := s.Orchestrations().Insert(fixOrchestration())
		require.NoError(t, err)

		c := NewCanceler(s.Operations(), s.Orchestrations(), logrus.New())

		id, err := c.Cancel()

		require.NoError(t, err)
		assert.Equal(t, fixOrchestrationID, id)

		isCanceled, err := isCanceled(s.Orchestrations())
		require.NoError(t, err)

		assert.True(t, isCanceled)
	})
	t.Run("should return error when orchestration not found", func(t *testing.T) {
		s := storage.NewMemoryStorage()
		c := NewCanceler(s.Operations(), s.Orchestrations(), logrus.New())

		_, err := c.Cancel()
		assert.Error(t, err)
	})
}

func TestCanceler_CancelForID(t *testing.T) {
	t.Run("should cancel orchestration", func(t *testing.T) {
		s := storage.NewMemoryStorage()
		err := s.Orchestrations().Insert(fixOrchestration())
		require.NoError(t, err)

		c := NewCanceler(s.Operations(), s.Orchestrations(), logrus.New())

		err = c.CancelForID(fixOrchestrationID)
		require.NoError(t, err)

		isCanceled, err := isCanceled(s.Orchestrations())
		require.NoError(t, err)

		assert.True(t, isCanceled)
	})
	t.Run("should return error when orchestration not found", func(t *testing.T) {
		s := storage.NewMemoryStorage()
		c := NewCanceler(s.Operations(), s.Orchestrations(), logrus.New())

		err := c.CancelForID(fixOrchestrationID)
		assert.Error(t, err)
	})
}

func isCanceled(s storage.Orchestrations) (bool, error) {
	o, err := s.GetByID(fixOrchestrationID)
	if err != nil {
		return false, err
	}
	if o.State == orchestration.Canceled {
		return true, nil
	}
	return false, nil
}

func fixOrchestration() orchestration.Orchestration {
	n := time.Now()
	return orchestration.Orchestration{
		OrchestrationID: fixOrchestrationID,
		State:           orchestration.InProgress,
		CreatedAt:       n,
		UpdatedAt:       n,
		Parameters:      orchestration.Parameters{},
	}
}
