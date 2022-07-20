package update

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input/automock"
	automock2 "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/update/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInitialisationStep_OtherOperationIsInProgress(t *testing.T) {

	for tn, tc := range map[string]struct {
		beforeFunc     func(os storage.Operations)
		expectedRepeat bool
	}{
		"in progress provisioning": {
			beforeFunc: func(os storage.Operations) {
				provisioningOperation := fixture.FixProvisioningOperation("p-id", "iid")
				provisioningOperation.State = domain.InProgress
				os.InsertProvisioningOperation(provisioningOperation)
			},
			expectedRepeat: true,
		},
		"succeeded provisioning": {
			beforeFunc: func(os storage.Operations) {
				provisioningOperation := fixture.FixProvisioningOperation("p-id", "iid")
				provisioningOperation.State = domain.Succeeded
				os.InsertProvisioningOperation(provisioningOperation)
			},
			expectedRepeat: false,
		},
		"in progress upgrade shoot": {
			beforeFunc: func(os storage.Operations) {
				op := fixture.FixUpgradeClusterOperation("op-id", "iid")
				op.State = domain.InProgress
				os.InsertUpgradeClusterOperation(op)
			},
			expectedRepeat: true,
		},
		"in progress upgrade kyma": {
			beforeFunc: func(os storage.Operations) {
				op := fixture.FixUpgradeKymaOperation("op-id", "iid")
				op.State = domain.InProgress
				os.InsertUpgradeKymaOperation(op)
			},
			expectedRepeat: true,
		},
		"in progress update": {
			beforeFunc: func(os storage.Operations) {
				op := fixture.FixUpdatingOperation("op-id", "iid")
				op.State = domain.InProgress
				os.InsertUpdatingOperation(op)
			},
			expectedRepeat: true,
		},
		"in progress deprovisioning": {
			beforeFunc: func(os storage.Operations) {
				op := fixture.FixDeprovisioningOperation("op-id", "iid")
				op.State = domain.InProgress
				os.InsertDeprovisioningOperation(op)
			},
			expectedRepeat: true,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			db := storage.NewMemoryStorage()
			os := db.Operations()
			is := db.Instances()
			rs := db.RuntimeStates()
			inst := fixture.FixInstance("iid")
			state := fixture.FixRuntimeState("op-id", "Runtime-iid", "op-id")
			is.Insert(inst)
			rs.Insert(state)
			ver := &internal.RuntimeVersionData{
				Version: "2.4.0",
				Origin:  internal.Defaults,
			}
			rvc := &automock2.RuntimeVersionConfiguratorForUpdating{}
			rvc.On("ForUpdating",
				mock.AnythingOfType("internal.UpdatingOperation")).
				Return(ver, nil)
			builder := &automock.CreatorForPlan{}
			builder.On("CreateUpgradeShootInput", mock.Anything).Return(&fixture.SimpleInputCreator{}, nil)
			step := NewInitialisationStep(is, os, rs, rvc, builder)
			updatingOperation := fixture.FixUpdatingOperation("up-id", "iid")
			updatingOperation.State = orchestration.Pending
			os.InsertUpdatingOperation(updatingOperation)
			tc.beforeFunc(os)

			// when
			_, d, err := step.Run(updatingOperation, logrus.New())

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.expectedRepeat, d != 0)
		})
	}
}
