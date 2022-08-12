package runtime

import (
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/stretchr/testify/assert"
)

func TestConverting_Provisioning(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.InProgress, time.Now()))

	// then
	assert.Equal(t, runtime.StateProvisioning, dto.Status.State)
}

func TestConverting_Provisioned(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))

	// then
	assert.Equal(t, runtime.StateSucceeded, dto.Status.State)
}

func TestConverting_ProvisioningFailed(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Failed, time.Now()))

	// then
	assert.Equal(t, runtime.StateFailed, dto.Status.State)
}

func TestConverting_Updating(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
	svc.ApplyUpdateOperations(&dto, []internal.UpdatingOperation{{
		Operation: internal.Operation{
			CreatedAt: time.Now().Add(time.Second),
			ID:        "prov-id",
			State:     domain.InProgress,
		},
	}}, 1)

	// then
	assert.Equal(t, runtime.StateUpdating, dto.Status.State)
}

func TestConverting_UpdateFailed(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
	svc.ApplyUpdateOperations(&dto, []internal.UpdatingOperation{{
		Operation: internal.Operation{
			CreatedAt: time.Now().Add(time.Second),
			ID:        "prov-id",
			State:     domain.Failed,
		},
	}}, 1)

	// then
	assert.Equal(t, runtime.StateError, dto.Status.State)
}

func TestConverting_Suspending(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
	svc.ApplySuspensionOperations(&dto, fixSuspensionOperation(domain.InProgress, time.Now().Add(time.Second)))

	// then
	assert.Equal(t, runtime.StateDeprovisioning, dto.Status.State)
}

func TestConverting_Deprovisioning(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
	svc.ApplyDeprovisioningOperation(&dto, fixDeprovisionOperation(domain.InProgress, time.Now().Add(time.Second)))

	// then
	assert.Equal(t, runtime.StateDeprovisioning, dto.Status.State)
}

func TestConverting_SuspendedAndUpdated(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
	svc.ApplySuspensionOperations(&dto, fixSuspensionOperation(domain.Succeeded, time.Now().Add(time.Second)))
	svc.ApplyUpdateOperations(&dto, []internal.UpdatingOperation{{
		Operation: internal.Operation{
			CreatedAt: time.Now().Add(2 * time.Second),
			ID:        "prov-id",
			State:     domain.Succeeded,
		},
	}}, 1)

	// then
	assert.Equal(t, runtime.StateSuspended, dto.Status.State)
}

func TestConverting_SuspendedAndUpdateFAiled(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))
	svc.ApplySuspensionOperations(&dto, fixSuspensionOperation(domain.Succeeded, time.Now().Add(time.Second)))
	svc.ApplyUpdateOperations(&dto, []internal.UpdatingOperation{{
		Operation: internal.Operation{
			CreatedAt: time.Now().Add(2 * time.Second),
			ID:        "prov-id",
			State:     domain.Failed,
		},
	}}, 1)

	// then
	assert.Equal(t, runtime.StateSuspended, dto.Status.State)
}

func TestConverting_ProvisionedOperationWithoutStagesAndVersion(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperation(domain.Succeeded, time.Now()))

	// then
	assert.Equal(t, []string{}, dto.Status.Provisioning.FinishedStagesOrdered)
	assert.Equal(t, "", dto.Status.Provisioning.RuntimeVersion)
}

func TestConverting_ProvisionedOperationWithStagesAndVersion(t *testing.T) {
	// given
	instance := fixInstance()
	svc := NewConverter("eu")

	// when
	dto, _ := svc.NewDTO(instance)
	svc.ApplyProvisioningOperation(&dto, fixProvisioningOperationWithStagesAndVersion(domain.Succeeded, time.Now()))

	// then
	assert.Equal(t, []string{"start", "create_runtime", "check_kyma", "post_actions"}, dto.Status.Provisioning.FinishedStagesOrdered)
	assert.Equal(t, "2.0", dto.Status.Provisioning.RuntimeVersion)
}

func fixSuspensionOperation(state domain.LastOperationState, createdAt time.Time) []internal.DeprovisioningOperation {
	return []internal.DeprovisioningOperation{{
		Operation: internal.Operation{
			CreatedAt: createdAt,
			ID:        "s-id",
			State:     state,
			Temporary: true,
		},
	}}
}

func fixDeprovisionOperation(state domain.LastOperationState, createdAt time.Time) *internal.DeprovisioningOperation {
	return &internal.DeprovisioningOperation{
		Operation: internal.Operation{
			CreatedAt: createdAt,
			ID:        "s-id",
			State:     state,
		},
	}
}

func fixInstance() internal.Instance {
	return internal.Instance{
		InstanceID:                  "instance-id",
		RuntimeID:                   "runtime-id",
		GlobalAccountID:             "global-account-id",
		SubscriptionGlobalAccountID: "subgid",
		SubAccountID:                "sub-account-id",
	}
}

func fixProvisioningOperation(state domain.LastOperationState, createdAt time.Time) *internal.ProvisioningOperation {
	return &internal.ProvisioningOperation{
		Operation: internal.Operation{
			CreatedAt: createdAt,
			ID:        "prov-id",
			State:     state,
		},
	}
}

func fixProvisioningOperationWithStagesAndVersion(state domain.LastOperationState, createdAt time.Time) *internal.ProvisioningOperation {
	return &internal.ProvisioningOperation{
		Operation: internal.Operation{
			CreatedAt:             createdAt,
			ID:                    "prov-id",
			State:                 state,
			FinishedStagesOrdered: "start,create_runtime,check_kyma,post_actions",
			RuntimeVersion: internal.RuntimeVersionData{
				Version:      "2.0",
				Origin:       "default",
				MajorVersion: 2,
			},
		},
	}
}
