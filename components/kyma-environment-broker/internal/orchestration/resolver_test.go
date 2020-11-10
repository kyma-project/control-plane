package orchestration

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	gardenerapi "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardenerclient_fake "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1/fake"
	brokerapi "github.com/pivotal-cf/brokerapi/v7/domain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession/dbmodel"
)

const (
	shootNamespace = "garden-kyma"

	globalAccountID1 = "f8576376-603b-40a8-9225-0edc65052463"
	globalAccountID2 = "cb4d9447-8a6c-47d4-a2cd-48fa8121a91e"
	globalAccountID3 = "cb4d9447-8a6c-47d4-a2cd-48fa8121a91e"

	region1 = "westeurope"
	region2 = "centralus"
	region3 = "uksouth"

	plan1 = "azure"
	plan2 = "gcp"
)

func TestResolver_Resolve(t *testing.T) {
	client := newFakeGardenerClient()
	lister := newInstanceListerMock()
	defer lister.AssertExpectations(t)
	logger := logger.NewLogDummy()
	resolver := NewGardenerRuntimeResolver(client, shootNamespace, lister, logger)

	expectedRuntime1 := expectedRuntime{
		shoot:    &shoot1,
		instance: &instance1.Instance,
	}
	expectedRuntime2 := expectedRuntime{
		shoot:    &shoot2,
		instance: &instance2.Instance,
	}
	expectedRuntime3 := expectedRuntime{
		shoot:    &shoot3,
		instance: &instance3.Instance,
	}

	for tn, tc := range map[string]struct {
		Target           orchestration.TargetSpec
		ExpectedRuntimes []expectedRuntime
	}{
		"IncludeAll": {
			Target: orchestration.TargetSpec{
				Include: []orchestration.RuntimeTarget{
					{
						Target: orchestration.TargetAll,
					},
				},
				Exclude: nil,
			},
			ExpectedRuntimes: []expectedRuntime{expectedRuntime1, expectedRuntime2, expectedRuntime3},
		},
		"IncludeAllExcludeOne": {
			Target: orchestration.TargetSpec{
				Include: []orchestration.RuntimeTarget{
					{
						Target: orchestration.TargetAll,
					},
				},
				Exclude: []orchestration.RuntimeTarget{
					{
						GlobalAccount: expectedRuntime2.instance.GlobalAccountID,
						SubAccount:    expectedRuntime2.instance.SubAccountID,
					},
				},
			},
			ExpectedRuntimes: []expectedRuntime{expectedRuntime1, expectedRuntime3},
		},
		"ExcludeAll": {
			Target: orchestration.TargetSpec{
				Include: []orchestration.RuntimeTarget{
					{
						Target: orchestration.TargetAll,
					},
				},
				Exclude: []orchestration.RuntimeTarget{
					{
						Target: orchestration.TargetAll,
					},
				},
			},
			ExpectedRuntimes: []expectedRuntime{},
		},
		"IncludeOne": {
			Target: orchestration.TargetSpec{
				Include: []orchestration.RuntimeTarget{
					{
						GlobalAccount: expectedRuntime2.instance.GlobalAccountID,
						SubAccount:    expectedRuntime2.instance.SubAccountID,
					},
				},
				Exclude: nil,
			},
			ExpectedRuntimes: []expectedRuntime{expectedRuntime2},
		},
		"IncludeRuntime": {
			Target: orchestration.TargetSpec{
				Include: []orchestration.RuntimeTarget{
					{
						RuntimeID: "runtime-id-1",
					},
				},
				Exclude: nil,
			},
			ExpectedRuntimes: []expectedRuntime{expectedRuntime1},
		},
		"IncludeTenant": {
			Target: orchestration.TargetSpec{
				Include: []orchestration.RuntimeTarget{
					{
						GlobalAccount: globalAccountID1,
					},
				},
				Exclude: nil,
			},
			ExpectedRuntimes: []expectedRuntime{expectedRuntime1, expectedRuntime2},
		},
		"IncludeRegion": {
			Target: orchestration.TargetSpec{
				Include: []orchestration.RuntimeTarget{
					{
						Region: "europe|eu|uk",
					},
				},
				Exclude: nil,
			},
			ExpectedRuntimes: []expectedRuntime{expectedRuntime1, expectedRuntime3},
		},
		"IncludePlanName": {
			Target: orchestration.TargetSpec{
				Include: []orchestration.RuntimeTarget{
					{
						PlanName: plan1,
					},
				},
				Exclude: nil,
			},
			ExpectedRuntimes: []expectedRuntime{expectedRuntime2, expectedRuntime3},
		},
	} {
		t.Run(tn, func(t *testing.T) {
			// when
			runtimes, err := resolver.Resolve(tc.Target)

			// then
			assert.Nil(t, err)

			if len(tc.ExpectedRuntimes) != 0 {
				assertRuntimeTargets(t, tc.ExpectedRuntimes, runtimes)
			} else {
				assert.Empty(t, runtimes)
			}
		})
	}
}

func TestResolver_Resolve_GardenerFailure(t *testing.T) {
	// given
	fake := &k8stesting.Fake{}
	client := &gardenerclient_fake.FakeCoreV1beta1{
		Fake: fake,
	}
	fake.AddReactor("list", "shoots", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("Fake gardener client failure")
	})
	lister := newInstanceListerMock()
	defer lister.AssertExpectations(t)
	logger := logger.NewLogDummy()
	resolver := NewGardenerRuntimeResolver(client, shootNamespace, lister, logger)

	// when
	runtimes, err := resolver.Resolve(orchestration.TargetSpec{
		Include: []orchestration.RuntimeTarget{
			{
				Target: orchestration.TargetAll,
			},
		},
		Exclude: nil,
	})

	// then
	assert.NotNil(t, err)
	assert.Len(t, runtimes, 0)
}

func TestResolver_Resolve_StorageFailure(t *testing.T) {
	// given
	client := newFakeGardenerClient()
	lister := &automock.InstanceLister{}
	lister.On("FindAllJoinedWithOperations", mock.Anything).Return(
		nil,
		errors.New("Mock storage failure"),
	)
	defer lister.AssertExpectations(t)
	logger := logger.NewLogDummy()
	resolver := NewGardenerRuntimeResolver(client, shootNamespace, lister, logger)

	// when
	runtimes, err := resolver.Resolve(orchestration.TargetSpec{
		Include: []orchestration.RuntimeTarget{
			{
				Target: orchestration.TargetAll,
			},
		},
		Exclude: nil,
	})

	// then
	assert.NotNil(t, err)
	assert.Len(t, runtimes, 0)
}

var (
	shoot1                  = fixShoot(1, globalAccountID1, region1)
	shoot2                  = fixShoot(2, globalAccountID1, region2)
	shoot3                  = fixShoot(3, globalAccountID2, region3)
	shoot4                  = fixShoot(4, globalAccountID3, region1)
	instance1               = fixInstanceWithOperation(1, globalAccountID1, string(dbmodel.OperationTypeProvision), string(brokerapi.Succeeded), plan2)
	instance2               = fixInstanceWithOperation(2, globalAccountID1, string(dbmodel.OperationTypeProvision), string(brokerapi.Succeeded), plan1)
	instance3               = fixInstanceWithOperation(3, globalAccountID2, string(dbmodel.OperationTypeProvision), string(brokerapi.Succeeded), plan1)
	instance4               = fixInstanceWithOperation(4, globalAccountID3, string(dbmodel.OperationTypeProvision), string(brokerapi.Succeeded), plan1)
	instance4Deprovisioning = fixInstanceWithOperation(4, globalAccountID3, string(dbmodel.OperationTypeDeprovision), string(brokerapi.InProgress), plan1)
	instance5Failed         = fixInstanceWithOperation(5, globalAccountID3, string(dbmodel.OperationTypeProvision), string(brokerapi.Failed), plan1)
	instance6Provisioning   = fixInstanceWithOperation(6, globalAccountID3, string(dbmodel.OperationTypeProvision), string(brokerapi.InProgress), plan2)
	instance7               = fixInstanceWithOperation(7, globalAccountID1, string(dbmodel.OperationTypeProvision), string(brokerapi.Succeeded), plan1)
)

func fixShoot(id int, globalAccountID, region string) gardenerapi.Shoot {
	return gardenerapi.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("shoot%d", id),
			Namespace: shootNamespace,
			Labels: map[string]string{
				globalAccountLabel: globalAccountID,
				subAccountLabel:    fmt.Sprintf("subaccount-id-%d", id),
			},
			Annotations: map[string]string{
				runtimeIDAnnotation: fmt.Sprintf("runtime-id-%d", id),
			},
		},
		Spec: gardenerapi.ShootSpec{
			Region: region,
			Maintenance: &gardenerapi.Maintenance{
				TimeWindow: &gardenerapi.MaintenanceTimeWindow{
					Begin: "030000+0000",
					End:   "040000+0000",
				},
			},
		},
	}
}

func fixInstanceWithOperation(id int, globalAccountID, opType, opState, planName string) internal.InstanceWithOperation {
	return internal.InstanceWithOperation{
		Instance: internal.Instance{
			InstanceID:      fmt.Sprintf("instance-id-%d", id),
			RuntimeID:       fmt.Sprintf("runtime-id-%d", id),
			GlobalAccountID: globalAccountID,
			SubAccountID:    fmt.Sprintf("subaccount-id-%d", id),
			ServicePlanName: planName,
		},
		Type: sql.NullString{
			String: opType,
		},
		State: sql.NullString{
			String: opState,
		},
	}
}

type expectedRuntime struct {
	shoot    *gardenerapi.Shoot
	instance *internal.Instance
}

func newFakeGardenerClient() *gardenerclient_fake.FakeCoreV1beta1 {
	fake := &k8stesting.Fake{}
	client := &gardenerclient_fake.FakeCoreV1beta1{
		Fake: fake,
	}
	fake.AddReactor("list", "shoots", func(action k8stesting.Action) (bool, runtime.Object, error) {
		sl := &gardenerapi.ShootList{
			Items: []gardenerapi.Shoot{
				shoot1,
				shoot2,
				shoot3,
				shoot4,
			},
		}
		return true, sl, nil
	})

	return client
}

func newInstanceListerMock() *automock.InstanceLister {
	lister := &automock.InstanceLister{}
	lister.On("FindAllJoinedWithOperations", mock.Anything).Maybe().Return(
		[]internal.InstanceWithOperation{
			instance1,
			instance2,
			instance3,
			instance4,
			instance4Deprovisioning,
			instance5Failed,
			instance6Provisioning,
			instance7,
		},
		nil,
	)
	return lister
}

func lookupRuntime(runtimeID string, runtimes []internal.Runtime) *internal.Runtime {
	for _, r := range runtimes {
		if r.RuntimeID == runtimeID {
			return &r
		}
	}

	return nil
}

func assertRuntimeTargets(t *testing.T, expectedRuntimes []expectedRuntime, runtimes []internal.Runtime) {
	require.Equal(t, len(expectedRuntimes), len(runtimes))

	for _, e := range expectedRuntimes {
		r := lookupRuntime(e.instance.RuntimeID, runtimes)
		assert.NotNil(t, r)
		assert.Equal(t, e.instance.InstanceID, r.InstanceID)
		assert.Equal(t, e.instance.GlobalAccountID, r.GlobalAccountID)
		assert.Equal(t, e.instance.SubAccountID, r.SubAccountID)
		assert.Equal(t, e.shoot.Name, r.ShootName)
		assert.Equal(t, e.shoot.Spec.Maintenance.TimeWindow.Begin, r.MaintenanceWindowBegin.Format(maintenanceWindowFormat))
		assert.Equal(t, e.shoot.Spec.Maintenance.TimeWindow.End, r.MaintenanceWindowEnd.Format(maintenanceWindowFormat))
	}
}
