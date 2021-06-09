package suspension

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dberr"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuspension(t *testing.T) {
	// given
	provisioning := NewDummyQueue()
	deprovisioning := NewDummyQueue()
	st := storage.NewMemoryStorage()

	svc := NewContextUpdateHandler(st.Operations(), provisioning, deprovisioning, logrus.New())
	instance := fixInstance(fixActiveErsContext())
	st.Instances().Insert(*instance)

	// when
	err := svc.Handle(instance, fixInactiveErsContext())
	require.NoError(t, err)

	// then
	op, _ := st.Operations().GetDeprovisioningOperationByInstanceID("instance-id")
	assertQueue(t, deprovisioning, op.ID)
	assertQueue(t, provisioning)

	assert.Equal(t, domain.LastOperationState("pending"), op.State)
	assert.Equal(t, instance.InstanceID, op.InstanceID)
}

func TestSuspension_Retrigger(t *testing.T) {
	t.Run("should skip suspension when temporary deprovisioning operation already succeeded", func(t *testing.T) {
		// given
		provisioning := NewDummyQueue()
		deprovisioning := NewDummyQueue()
		st := storage.NewMemoryStorage()

		svc := NewContextUpdateHandler(st.Operations(), provisioning, deprovisioning, logrus.New())
		instance := fixInstance(fixInactiveErsContext())
		st.Instances().Insert(*instance)
		st.Operations().InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:         "suspended-op-id",
				Version:    0,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				InstanceID: instance.InstanceID,
				State:      domain.Succeeded,
			},
			Temporary: true,
		})

		// when
		err := svc.Handle(instance, fixInactiveErsContext())
		require.NoError(t, err)

		// then
		op, _ := st.Operations().GetDeprovisioningOperationByInstanceID("instance-id")
		assertQueue(t, deprovisioning)
		assertQueue(t, provisioning)

		assert.Equal(t, domain.Succeeded, op.State)
		assert.Equal(t, instance.InstanceID, op.InstanceID)
	})

	t.Run("should retrigger deprovisioning when existing temporary deprovisioning operation failed", func(t *testing.T) {
		// given
		provisioning := NewDummyQueue()
		deprovisioning := NewDummyQueue()
		st := storage.NewMemoryStorage()

		svc := NewContextUpdateHandler(st.Operations(), provisioning, deprovisioning, logrus.New())
		instance := fixInstance(fixInactiveErsContext())
		st.Instances().Insert(*instance)
		st.Operations().InsertDeprovisioningOperation(internal.DeprovisioningOperation{
			Operation: internal.Operation{
				ID:         "suspended-op-id",
				Version:    0,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
				InstanceID: instance.InstanceID,
				State:      domain.Failed,
			},
			Temporary: true,
		})

		// when
		err := svc.Handle(instance, fixInactiveErsContext())
		require.NoError(t, err)

		// then
		op, _ := st.Operations().GetDeprovisioningOperationByInstanceID("instance-id")
		assertQueue(t, deprovisioning, op.ID)
		assertQueue(t, provisioning)

		assert.Equal(t, domain.LastOperationState("pending"), op.State)
		assert.Equal(t, instance.InstanceID, op.InstanceID)
	})

}

func assertQueue(t *testing.T, q *dummyQueue, id ...string) {
	t.Helper()
	if len(id) == 0 {
		assert.Empty(t, q.IDs)
		return
	}
	assert.Equal(t, q.IDs, id)
}

func TestUnsuspension(t *testing.T) {
	// given
	provisioning := NewDummyQueue()
	deprovisioning := NewDummyQueue()
	st := storage.NewMemoryStorage()

	svc := NewContextUpdateHandler(st.Operations(), provisioning, deprovisioning, logrus.New())
	instance := fixInstance(fixInactiveErsContext())
	instance.InstanceDetails.ShootName = "c-012345"
	instance.InstanceDetails.ShootDomain = "c-012345.sap.com"

	st.Instances().Insert(*instance)

	deprovisioningOperation := fixture.FixDeprovisioningOperation("d-op", "instance-id")
	deprovisioningOperation.Temporary = true
	st.Operations().InsertDeprovisioningOperation(deprovisioningOperation)

	// when
	err := svc.Handle(instance, fixActiveErsContext())
	require.NoError(t, err)

	// then
	op, err := st.Operations().GetProvisioningOperationByInstanceID("instance-id")
	require.NoError(t, err)
	assertQueue(t, deprovisioning)
	assertQueue(t, provisioning, op.ID)

	assert.Equal(t, domain.LastOperationState(orchestration.Pending), op.State)
	assert.Equal(t, instance.InstanceID, op.InstanceID)
	assert.Equal(t, "c-012345", op.ShootName)
	assert.Equal(t, "c-012345.sap.com", op.ShootDomain)
}

func TestUnsuspensionForDeprovisioningInstance(t *testing.T) {
	// given
	provisioning := NewDummyQueue()
	deprovisioning := NewDummyQueue()
	st := storage.NewMemoryStorage()

	svc := NewContextUpdateHandler(st.Operations(), provisioning, deprovisioning, logrus.New())
	instance := fixInstance(fixInactiveErsContext())
	instance.InstanceDetails.ShootName = "c-012345"
	instance.InstanceDetails.ShootDomain = "c-012345.sap.com"

	st.Instances().Insert(*instance)
	deprovisioningOperation := fixture.FixDeprovisioningOperation("d-op", "instance-id")
	deprovisioningOperation.Temporary = false
	st.Operations().InsertDeprovisioningOperation(deprovisioningOperation)

	// when
	err := svc.Handle(instance, fixActiveErsContext())
	require.NoError(t, err)

	// then
	_, err = st.Operations().GetProvisioningOperationByInstanceID("instance-id")
	assert.True(t, dberr.IsNotFound(err))
	assertQueue(t, deprovisioning)
	assertQueue(t, provisioning)
}

func TestUnsuspensionWithoutShootname(t *testing.T) {
	// given
	provisioning := NewDummyQueue()
	deprovisioning := NewDummyQueue()
	st := storage.NewMemoryStorage()

	svc := NewContextUpdateHandler(st.Operations(), provisioning, deprovisioning, logrus.New())
	instance := fixInstance(fixInactiveErsContext())
	instance.InstanceDetails.ShootName = ""
	instance.InstanceDetails.ShootDomain = ""
	instance.DashboardURL = "https://console.c-7f1eb9e.kyma-dev.shoot.canary.k8s-hana.ondemand.com"

	st.Instances().Insert(*instance)

	// when
	err := svc.Handle(instance, fixActiveErsContext())
	require.NoError(t, err)

	// then
	op, err := st.Operations().GetProvisioningOperationByInstanceID("instance-id")
	require.NoError(t, err)
	assertQueue(t, deprovisioning)
	assertQueue(t, provisioning, op.ID)

	assert.Equal(t, domain.LastOperationState(orchestration.Pending), op.State)
	assert.Equal(t, instance.InstanceID, op.InstanceID)
	assert.Equal(t, "c-7f1eb9e", op.ShootName)
	assert.Equal(t, "c-7f1eb9e.kyma-dev.shoot.canary.k8s-hana.ondemand.com", op.ShootDomain)
}

func fixInstance(ersContext internal.ERSContext) *internal.Instance {
	instance := fixture.FixInstance("instance-id")
	instance.ServicePlanID = broker.TrialPlanID
	instance.Parameters.ErsContext = ersContext

	return &instance
}

func fixActiveErsContext() internal.ERSContext {
	return internal.ERSContext{
		Active: ptr.Bool(true),
	}
}

func fixInactiveErsContext() internal.ERSContext {
	return internal.ERSContext{
		Active: ptr.Bool(false),
	}
}

type dummyQueue struct {
	IDs []string
}

func NewDummyQueue() *dummyQueue {
	return &dummyQueue{
		IDs: []string{},
	}
}

func (q *dummyQueue) Add(id string) {
	q.IDs = append(q.IDs, id)
}
