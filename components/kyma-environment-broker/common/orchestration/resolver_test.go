package orchestration

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	brokerapi "github.com/pivotal-cf/brokerapi/v8/domain"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8s "k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
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
	lister := newRuntimeListerMock()
	defer lister.AssertExpectations(t)
	logger := newLogDummy()
	resolver := NewGardenerRuntimeResolver(client, shootNamespace, lister, logger)

	expectedRuntime1 := expectedRuntime{
		shoot:   &shoot1,
		runtime: &runtime1,
	}
	expectedRuntime2 := expectedRuntime{
		shoot:   &shoot2,
		runtime: &runtime2,
	}
	expectedRuntime3 := expectedRuntime{
		shoot:   &shoot3,
		runtime: &runtime3,
	}
	expectedRuntime10 := expectedRuntime{
		shoot:   &shoot10,
		runtime: &runtime10,
	}

	for tn, tc := range map[string]struct {
		Target           TargetSpec
		ExpectedRuntimes []expectedRuntime
	}{
		"IncludeAll": {
			Target: TargetSpec{
				Include: []RuntimeTarget{
					{
						Target: TargetAll,
					},
				},
				Exclude: nil,
			},
			ExpectedRuntimes: []expectedRuntime{expectedRuntime1, expectedRuntime2, expectedRuntime3, expectedRuntime10},
		},
		"IncludeAllExcludeOne": {
			Target: TargetSpec{
				Include: []RuntimeTarget{
					{
						Target: TargetAll,
					},
				},
				Exclude: []RuntimeTarget{
					{
						GlobalAccount: expectedRuntime2.runtime.GlobalAccountID,
						SubAccount:    expectedRuntime2.runtime.SubAccountID,
					},
				},
			},
			ExpectedRuntimes: []expectedRuntime{expectedRuntime1, expectedRuntime3, expectedRuntime10},
		},
		"ExcludeAll": {
			Target: TargetSpec{
				Include: []RuntimeTarget{
					{
						Target: TargetAll,
					},
				},
				Exclude: []RuntimeTarget{
					{
						Target: TargetAll,
					},
				},
			},
			ExpectedRuntimes: []expectedRuntime{},
		},
		"IncludeOne": {
			Target: TargetSpec{
				Include: []RuntimeTarget{
					{
						GlobalAccount: expectedRuntime2.runtime.GlobalAccountID,
						SubAccount:    expectedRuntime2.runtime.SubAccountID,
					},
				},
				Exclude: nil,
			},
			ExpectedRuntimes: []expectedRuntime{expectedRuntime2},
		},
		"IncludeRuntime": {
			Target: TargetSpec{
				Include: []RuntimeTarget{
					{
						RuntimeID: "runtime-id-1",
					},
				},
				Exclude: nil,
			},
			ExpectedRuntimes: []expectedRuntime{expectedRuntime1},
		},
		"IncludeInstance": {
			Target: TargetSpec{
				Include: []RuntimeTarget{
					{
						InstanceID: "instance-id-1",
					},
				},
				Exclude: nil,
			},
			ExpectedRuntimes: []expectedRuntime{expectedRuntime1},
		},
		"IncludeTenant": {
			Target: TargetSpec{
				Include: []RuntimeTarget{
					{
						GlobalAccount: globalAccountID1,
					},
				},
				Exclude: nil,
			},
			ExpectedRuntimes: []expectedRuntime{expectedRuntime1, expectedRuntime2, expectedRuntime10},
		},
		"IncludeRegion": {
			Target: TargetSpec{
				Include: []RuntimeTarget{
					{
						Region: "europe|eu|uk",
					},
				},
				Exclude: nil,
			},
			ExpectedRuntimes: []expectedRuntime{expectedRuntime1, expectedRuntime3, expectedRuntime10},
		},
		"IncludePlanName": {
			Target: TargetSpec{
				Include: []RuntimeTarget{
					{
						PlanName: plan1,
					},
				},
				Exclude: nil,
			},
			ExpectedRuntimes: []expectedRuntime{expectedRuntime2, expectedRuntime3, expectedRuntime10},
		},
		"IncludeShoot": {
			Target: TargetSpec{
				Include: []RuntimeTarget{
					{
						Shoot: expectedRuntime1.shoot.GetName(),
					},
				},
				Exclude: nil,
			},
			ExpectedRuntimes: []expectedRuntime{expectedRuntime1},
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
	fake := k8stesting.Fake{}
	client := gardener.NewDynamicFakeClient()
	client.Fake = fake
	fake.AddReactor("list", "shoots", func(action k8stesting.Action) (bool, k8s.Object, error) {
		return true, nil, errors.New("Fake gardener client failure")
	})
	lister := newRuntimeListerMock()
	defer lister.AssertExpectations(t)
	logger := newLogDummy()
	resolver := NewGardenerRuntimeResolver(client, shootNamespace, lister, logger)

	// when
	runtimes, err := resolver.Resolve(TargetSpec{
		Include: []RuntimeTarget{
			{
				Target: TargetAll,
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
	lister := &RuntimeListerMock{}
	lister.On("ListAllRuntimes").Return(
		nil,
		errors.New("Mock storage failure"),
	)
	defer lister.AssertExpectations(t)
	logger := newLogDummy()
	resolver := NewGardenerRuntimeResolver(client, shootNamespace, lister, logger)

	// when
	runtimes, err := resolver.Resolve(TargetSpec{
		Include: []RuntimeTarget{
			{
				Target: TargetAll,
			},
		},
		Exclude: nil,
	})

	// then
	assert.NotNil(t, err)
	assert.Len(t, runtimes, 0)
}

var (
	shoot1 = fixShoot(1, globalAccountID1, region1)
	shoot2 = fixShoot(2, globalAccountID1, region2)
	shoot3 = fixShoot(3, globalAccountID2, region3)
	shoot4 = fixShoot(4, globalAccountID3, region1)
	shoot5 = fixShoot(5, globalAccountID1, region1)
	shoot6 = fixShoot(6, globalAccountID1, region1)
	// shoot7 is purposefully missing to test missing cluster scenario
	shoot8  = fixShoot(8, globalAccountID1, region1)
	shoot9  = fixShoot(9, globalAccountID1, region1)
	shoot10 = fixShoot(10, globalAccountID1, region1)
	shoot11 = fixShoot(11, globalAccountID1, region1)

	runtime1  = fixRuntimeDTO(1, globalAccountID1, plan2, runtimeOpState{provision: string(brokerapi.Succeeded)})
	runtime2  = fixRuntimeDTO(2, globalAccountID1, plan1, runtimeOpState{provision: string(brokerapi.Succeeded)})
	runtime3  = fixRuntimeDTO(3, globalAccountID2, plan1, runtimeOpState{provision: string(brokerapi.Succeeded)})
	runtime4  = fixRuntimeDTO(4, globalAccountID3, plan1, runtimeOpState{provision: string(brokerapi.Succeeded), deprovision: string(brokerapi.InProgress)})
	runtime5  = fixRuntimeDTO(5, globalAccountID3, plan1, runtimeOpState{provision: string(brokerapi.Failed)})
	runtime6  = fixRuntimeDTO(6, globalAccountID3, plan2, runtimeOpState{provision: string(brokerapi.InProgress)})
	runtime7  = fixRuntimeDTO(7, globalAccountID1, plan1, runtimeOpState{provision: string(brokerapi.Succeeded)})
	runtime8  = fixRuntimeDTO(8, globalAccountID1, plan1, runtimeOpState{provision: string(brokerapi.Succeeded), suspension: string(brokerapi.Succeeded)})
	runtime9  = fixRuntimeDTO(9, globalAccountID1, plan1, runtimeOpState{provision: string(brokerapi.Succeeded), suspension: string(brokerapi.InProgress)})
	runtime10 = fixRuntimeDTO(10, globalAccountID1, plan1, runtimeOpState{provision: string(brokerapi.Succeeded), suspension: string(brokerapi.Succeeded), unsuspension: string(brokerapi.Succeeded)})
	runtime11 = fixRuntimeDTO(11, globalAccountID1, plan1, runtimeOpState{provision: string(brokerapi.Succeeded), suspension: string(brokerapi.Succeeded), unsuspension: string(brokerapi.Failed)})
)

func fixShoot(id int, globalAccountID, region string) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "core.gardener.cloud/v1beta1",
			"kind":       "Shoot",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("shoot%d", id),
				"namespace": shootNamespace,
				"labels": map[string]interface{}{
					globalAccountLabel: globalAccountID,
					subAccountLabel:    fmt.Sprintf("subaccount-id-%d", id),
				},
				"annotations": map[string]interface{}{
					runtimeIDAnnotation: fmt.Sprintf("runtime-id-%d", id),
				},
			},
			"spec": map[string]interface{}{
				"region": region,
				"maintenance": map[string]interface{}{
					"timeWindow": map[string]interface{}{
						"begin": "030000+0000",
						"end":   "040000+0000",
					},
				},
			},
		},
	}
}

type runtimeOpState struct {
	provision    string
	deprovision  string
	suspension   string
	unsuspension string
}

func fixRuntimeDTO(id int, globalAccountID, planName string, state runtimeOpState) runtime.RuntimeDTO {
	rt := runtime.RuntimeDTO{
		InstanceID:      fmt.Sprintf("instance-id-%d", id),
		RuntimeID:       fmt.Sprintf("runtime-id-%d", id),
		GlobalAccountID: globalAccountID,
		SubAccountID:    fmt.Sprintf("subaccount-id-%d", id),
		ServicePlanName: planName,
		Status: runtime.RuntimeStatus{
			Provisioning: &runtime.Operation{
				State:     state.provision,
				CreatedAt: time.Now(),
			},
		},
	}

	deprovTime := time.Now().Add(time.Minute)
	if state.suspension != "" {
		rt.Status.Suspension = &runtime.OperationsData{}
		rt.Status.Suspension.Count = 1
		rt.Status.Suspension.TotalCount = 1
		rt.Status.Suspension.Data = []runtime.Operation{
			{
				State:     state.suspension,
				CreatedAt: deprovTime,
			},
		}
		state.deprovision = state.suspension
	}

	if state.deprovision != "" {
		rt.Status.Deprovisioning = &runtime.Operation{
			State:     state.deprovision,
			CreatedAt: deprovTime,
		}
	}

	if state.unsuspension != "" {
		rt.Status.Unsuspension = &runtime.OperationsData{}
		rt.Status.Unsuspension.Count = 1
		rt.Status.Unsuspension.TotalCount = 1
		rt.Status.Unsuspension.Data = []runtime.Operation{
			{
				State:     state.unsuspension,
				CreatedAt: deprovTime.Add(time.Minute),
			},
		}
	}

	return rt
}

type expectedRuntime struct {
	shoot   *unstructured.Unstructured
	runtime *runtime.RuntimeDTO
}

func newFakeGardenerClient() *dynamicfake.FakeDynamicClient {
	client := gardener.NewDynamicFakeClient(
		&shoot1,
		&shoot2,
		&shoot3,
		&shoot4,
		&shoot5,
		&shoot6,
		&shoot8,
		&shoot9,
		&shoot10,
	)

	return client
}

func newRuntimeListerMock() *RuntimeListerMock {
	lister := &RuntimeListerMock{}
	lister.On("ListAllRuntimes").Maybe().Return(
		[]runtime.RuntimeDTO{
			runtime1,
			runtime2,
			runtime3,
			runtime4,
			runtime5,
			runtime6,
			runtime7,
			runtime8,
			runtime9,
			runtime10,
		},
		nil,
	)
	return lister
}

func newLogDummy() *logrus.Entry {
	rawLgr := logrus.New()
	rawLgr.Out = ioutil.Discard
	lgr := rawLgr.WithField("testing", true)

	return lgr
}

func lookupRuntime(runtimeID string, runtimes []Runtime) *Runtime {
	for _, r := range runtimes {
		if r.RuntimeID == runtimeID {
			return &r
		}
	}

	return nil
}

func assertRuntimeTargets(t *testing.T, expectedRuntimes []expectedRuntime, runtimes []Runtime) {
	require.Equal(t, len(expectedRuntimes), len(runtimes))

	for _, e := range expectedRuntimes {
		r := lookupRuntime(e.runtime.RuntimeID, runtimes)
		s := gardener.Shoot{*e.shoot}
		require.NotNil(t, r)
		assert.Equal(t, e.runtime.InstanceID, r.InstanceID)
		assert.Equal(t, e.runtime.GlobalAccountID, r.GlobalAccountID)
		assert.Equal(t, e.runtime.SubAccountID, r.SubAccountID)
		assert.Equal(t, s.GetName(), r.ShootName)
		assert.Equal(t, s.GetSpecMaintenanceTimeWindowBegin(), r.MaintenanceWindowBegin.Format(maintenanceWindowFormat))
		assert.Equal(t, s.GetSpecMaintenanceTimeWindowEnd(), r.MaintenanceWindowEnd.Format(maintenanceWindowFormat))
	}
}
