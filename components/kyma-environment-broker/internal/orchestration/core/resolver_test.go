package core

import (
	"database/sql"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	gardenerapi "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardenerclient_fake "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1/fake"
	brokerapi "github.com/pivotal-cf/brokerapi/v7/domain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/orchestration/core/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbsession/dbmodel"
)

const (
	shootNamespace = "garden-kyma"

	runtime1GlobalAccountID = "f8576376-603b-40a8-9225-0edc65052463"
	runtime1SubAccountID    = "b9ea5e77-c9ba-4af1-83a7-f2cf957353c1"
	runtime1ID              = "b7634bc8-da1f-4343-8513-0241bf81ecb2"
	runtime1InstanceID      = "c27be958-cd7e-4bbc-a3ef-33e81212bfb4"

	runtime2GlobalAccountID = "f8576376-603b-40a8-9225-0edc65052463"
	runtime2SubAccountID    = "15a296bd-da4a-408d-a76e-60fcf4a30014"
	runtime2ID              = "61bd315c-89d0-4462-acc5-174ea3162493"
	runtime2InstanceID      = "6395faea-4dbb-4913-b715-f15c8cb62280"

	runtime3GlobalAccountID = "cb4d9447-8a6c-47d4-a2cd-48fa8121a91e"
	runtime3SubAccountID    = "e08dbc3c-45ac-489a-8963-dc7c75527367"
	runtime3ID              = "7ba68c2a-0f63-4941-967d-30a9d3c30c0a"
	runtime3InstanceID      = "a115079e-3ea4-4a60-8ec8-26f3b9d16583"

	runtime4GlobalAccountID = "51b8c950-7ce3-4382-8670-99ae9549eabf"
	runtime4SubAccountID    = "74cb2f93-4a30-4ff9-9cd0-4242f2d8227d"
	runtime4ID              = "2cc9f953-5979-4efd-a2cd-ca60a125bf4a"
	runtime4InstanceID      = "f89cfee8-2506-472c-ae92-d00697672917"
)

var shoot1 = gardenerapi.Shoot{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "shoot1",
		Namespace: shootNamespace,
		Labels: map[string]string{
			globalAccountLabel: runtime1GlobalAccountID,
			subAccountLabel:    runtime1SubAccountID,
		},
		Annotations: map[string]string{
			runtimeIDAnnotation: runtime1ID,
		},
	},
	Spec: gardenerapi.ShootSpec{
		Region: "westeurope",
		Maintenance: &gardenerapi.Maintenance{
			TimeWindow: &gardenerapi.MaintenanceTimeWindow{
				Begin: "030000+0000",
				End:   "040000+0000",
			},
		},
	},
}

var instance1 = internal.InstanceWithOperation{
	Instance: internal.Instance{
		InstanceID:      runtime1InstanceID,
		RuntimeID:       runtime1ID,
		GlobalAccountID: runtime1GlobalAccountID,
		SubAccountID:    runtime1SubAccountID,
	},
	Type: sql.NullString{
		String: string(dbmodel.OperationTypeProvision),
	},
	State: sql.NullString{
		String: string(brokerapi.Succeeded),
	},
}

var shoot2 = gardenerapi.Shoot{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "shoot2",
		Namespace: shootNamespace,
		Labels: map[string]string{
			globalAccountLabel: runtime2GlobalAccountID,
			subAccountLabel:    runtime2SubAccountID,
		},
		Annotations: map[string]string{
			runtimeIDAnnotation: runtime2ID,
		},
	},
	Spec: gardenerapi.ShootSpec{
		Region: "centralus",
		Maintenance: &gardenerapi.Maintenance{
			TimeWindow: &gardenerapi.MaintenanceTimeWindow{
				Begin: "040000+0000",
				End:   "050000+0000",
			},
		},
	},
}

var instance2 = internal.InstanceWithOperation{
	Instance: internal.Instance{
		InstanceID:      runtime2InstanceID,
		RuntimeID:       runtime2ID,
		GlobalAccountID: runtime2GlobalAccountID,
		SubAccountID:    runtime2SubAccountID,
	},
	Type: sql.NullString{
		String: string(dbmodel.OperationTypeProvision),
	},
	State: sql.NullString{
		String: string(brokerapi.Succeeded),
	},
}

var shoot3 = gardenerapi.Shoot{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "shoot3",
		Namespace: shootNamespace,
		Labels: map[string]string{
			globalAccountLabel: runtime3GlobalAccountID,
			subAccountLabel:    runtime3SubAccountID,
		},
		Annotations: map[string]string{
			runtimeIDAnnotation: runtime3ID,
		},
	},
	Spec: gardenerapi.ShootSpec{
		Region: "uksouth",
		Maintenance: &gardenerapi.Maintenance{
			TimeWindow: &gardenerapi.MaintenanceTimeWindow{
				Begin: "150000+0000",
				End:   "160000+0000",
			},
		},
	},
}

var instance3 = internal.InstanceWithOperation{
	Instance: internal.Instance{
		InstanceID:      runtime3InstanceID,
		RuntimeID:       runtime3ID,
		GlobalAccountID: runtime3GlobalAccountID,
		SubAccountID:    runtime3SubAccountID,
	},
	Type: sql.NullString{
		String: string(dbmodel.OperationTypeProvision),
	},
	State: sql.NullString{
		String: string(brokerapi.Succeeded),
	},
}

var shoot4 = gardenerapi.Shoot{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "shoot4",
		Namespace: shootNamespace,
		Labels: map[string]string{
			globalAccountLabel: runtime4GlobalAccountID,
			subAccountLabel:    runtime4SubAccountID,
		},
		Annotations: map[string]string{
			runtimeIDAnnotation: runtime4ID,
		},
	},
	Spec: gardenerapi.ShootSpec{
		Region: "westeurope",
		Maintenance: &gardenerapi.Maintenance{
			TimeWindow: &gardenerapi.MaintenanceTimeWindow{
				Begin: "030000+0000",
				End:   "040000+0000",
			},
		},
	},
}

var instance4 = internal.InstanceWithOperation{
	Instance: internal.Instance{
		InstanceID:      runtime4InstanceID,
		RuntimeID:       runtime4ID,
		GlobalAccountID: runtime4GlobalAccountID,
		SubAccountID:    runtime4SubAccountID,
	},
	Type: sql.NullString{
		String: string(dbmodel.OperationTypeProvision),
	},
	State: sql.NullString{
		String: string(brokerapi.Succeeded),
	},
}

var instance4Deprovisioning = internal.InstanceWithOperation{
	Instance: internal.Instance{
		InstanceID:      runtime4InstanceID,
		RuntimeID:       runtime4ID,
		GlobalAccountID: runtime4GlobalAccountID,
		SubAccountID:    runtime4SubAccountID,
	},
	Type: sql.NullString{
		String: string(dbmodel.OperationTypeDeprovision),
	},
	State: sql.NullString{
		String: string(brokerapi.InProgress),
	},
}

var instance5Failed = internal.InstanceWithOperation{
	Instance: internal.Instance{
		InstanceID:      "bbedf05a-6943-4999-ba45-4895314cf847",
		RuntimeID:       "d80162bf-b8cd-4402-833e-2576f88bc086",
		GlobalAccountID: "6590826c-5a42-4755-ad94-6e26381af2fa",
		SubAccountID:    "ed68cee6-2bd6-47c0-9f45-cf9f97b2f724",
	},
	Type: sql.NullString{
		String: string(dbmodel.OperationTypeProvision),
	},
	State: sql.NullString{
		String: string(brokerapi.Failed),
	},
}

var instance6Provisioning = internal.InstanceWithOperation{
	Instance: internal.Instance{
		InstanceID:      "a6620ef5-a7ff-4673-bbc8-17eeb8fb2d65",
		RuntimeID:       "b662223a-640c-45f9-a29a-173af464d10b",
		GlobalAccountID: "9228d730-98ce-44c5-be2c-f93abcc97e29",
		SubAccountID:    "f6c2602a-7e8a-44d9-9fa1-96b541a931b5",
	},
	Type: sql.NullString{
		String: string(dbmodel.OperationTypeProvision),
	},
	State: sql.NullString{
		String: string(brokerapi.InProgress),
	},
}

var instance7 = internal.InstanceWithOperation{
	Instance: internal.Instance{
		InstanceID:      "e2f20cdd-33a7-49fa-ab03-052bebe3d670",
		RuntimeID:       "42611fd0-ee5b-4c5c-8728-1da08b4332bf",
		GlobalAccountID: "2cf17ad3-5d1e-4fb0-bc4d-cd2481b9103a",
		SubAccountID:    "4dea272f-23bf-4db6-a657-a140b3cdd783",
	},
	Type: sql.NullString{
		String: string(dbmodel.OperationTypeProvision),
	},
	State: sql.NullString{
		String: string(brokerapi.Succeeded),
	},
}

type expectedRuntime struct {
	shoot    *gardenerapi.Shoot
	instance *internal.Instance
}

var expectedRuntime1 = expectedRuntime{
	shoot:    &shoot1,
	instance: &instance1.Instance,
}
var expectedRuntime2 = expectedRuntime{
	shoot:    &shoot2,
	instance: &instance2.Instance,
}
var expectedRuntime3 = expectedRuntime{
	shoot:    &shoot3,
	instance: &instance3.Instance,
}
var expectedRuntime4 = expectedRuntime{
	shoot:    &shoot4,
	instance: &instance4.Instance,
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

func lookupRuntime(runtimeID string, runtimes []Runtime) *Runtime {
	for _, r := range runtimes {
		if r.RuntimeID == runtimeID {
			return &r
		}
	}

	return nil
}

func assertRuntimeTargets(t *testing.T, expectedRuntimes []expectedRuntime, runtimes []Runtime) {
	assert.Equal(t, len(expectedRuntimes), len(runtimes))

	for _, e := range expectedRuntimes {
		r := lookupRuntime(e.instance.RuntimeID, runtimes)
		assert.NotNil(t, r)
		assert.Equal(t, e.instance.InstanceID, r.InstanceID)
		assert.Equal(t, e.instance.GlobalAccountID, r.GlobalAccountID)
		assert.Equal(t, e.instance.SubAccountID, r.SubAccountID)
		assert.Equal(t, e.shoot.Name, r.ShootName)
		assert.Equal(t, e.shoot.Spec.Maintenance.TimeWindow.Begin, r.MaintenanceWindowBegin)
		assert.Equal(t, e.shoot.Spec.Maintenance.TimeWindow.End, r.MaintenanceWindowEnd)
	}
}

func TestResolver_Resolve_IncludeAll(t *testing.T) {
	// given
	client := newFakeGardenerClient()
	lister := newInstanceListerMock()
	defer lister.AssertExpectations(t)
	logger := logger.NewLogDummy()
	resolver := NewGardenerRuntimeResolver(client, shootNamespace, lister, logger)

	// when
	runtimes, err := resolver.Resolve(
		[]RuntimeTarget{
			{
				Target: TargetAll,
			},
		},
		nil,
	)

	// then
	assert.Nil(t, err)
	assertRuntimeTargets(t, []expectedRuntime{expectedRuntime1, expectedRuntime2, expectedRuntime3}, runtimes)
}

func TestResolver_Resolve_IncludeAllExcludeOne(t *testing.T) {
	// given
	client := newFakeGardenerClient()
	lister := newInstanceListerMock()
	defer lister.AssertExpectations(t)
	logger := logger.NewLogDummy()
	resolver := NewGardenerRuntimeResolver(client, shootNamespace, lister, logger)

	// when
	runtimes, err := resolver.Resolve(
		[]RuntimeTarget{
			{
				Target: TargetAll,
			},
		},
		[]RuntimeTarget{
			{
				GlobalAccount: runtime2GlobalAccountID,
				SubAccount:    runtime2SubAccountID,
			},
		},
	)

	// then
	assert.Nil(t, err)
	assertRuntimeTargets(t, []expectedRuntime{expectedRuntime1, expectedRuntime3}, runtimes)
}

func TestResolver_Resolve_ExcludeAll(t *testing.T) {
	// given
	client := newFakeGardenerClient()
	lister := newInstanceListerMock()
	defer lister.AssertExpectations(t)
	logger := logger.NewLogDummy()
	resolver := NewGardenerRuntimeResolver(client, shootNamespace, lister, logger)

	// when
	runtimes, err := resolver.Resolve(
		[]RuntimeTarget{
			{
				Target: TargetAll,
			},
		},
		[]RuntimeTarget{
			{
				Target: TargetAll,
			},
		},
	)

	// then
	assert.Nil(t, err)
	assert.Len(t, runtimes, 0)
}

func TestResolver_Resolve_IncludeOne(t *testing.T) {
	// given
	client := newFakeGardenerClient()
	lister := newInstanceListerMock()
	defer lister.AssertExpectations(t)
	logger := logger.NewLogDummy()
	resolver := NewGardenerRuntimeResolver(client, shootNamespace, lister, logger)

	// when
	runtimes, err := resolver.Resolve(
		[]RuntimeTarget{
			{
				GlobalAccount: runtime2GlobalAccountID,
				SubAccount:    runtime2SubAccountID,
			},
		},
		nil,
	)

	// then
	assert.Nil(t, err)
	assertRuntimeTargets(t, []expectedRuntime{expectedRuntime2}, runtimes)
}

func TestResolver_Resolve_IncludeTenant(t *testing.T) {
	// given
	client := newFakeGardenerClient()
	lister := newInstanceListerMock()
	defer lister.AssertExpectations(t)
	logger := logger.NewLogDummy()
	resolver := NewGardenerRuntimeResolver(client, shootNamespace, lister, logger)

	// when
	runtimes, err := resolver.Resolve(
		[]RuntimeTarget{
			{
				GlobalAccount: runtime1GlobalAccountID,
			},
		},
		nil,
	)

	// then
	assert.Nil(t, err)
	assertRuntimeTargets(t, []expectedRuntime{expectedRuntime1, expectedRuntime2}, runtimes)
}

func TestResolver_Resolve_IncludeRegion(t *testing.T) {
	// given
	client := newFakeGardenerClient()
	lister := newInstanceListerMock()
	defer lister.AssertExpectations(t)
	logger := logger.NewLogDummy()
	resolver := NewGardenerRuntimeResolver(client, shootNamespace, lister, logger)

	// when
	runtimes, err := resolver.Resolve(
		[]RuntimeTarget{
			{
				Region: "europe|eu|uk",
			},
		},
		nil,
	)

	// then
	assert.Nil(t, err)
	assertRuntimeTargets(t, []expectedRuntime{expectedRuntime1, expectedRuntime3}, runtimes)
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
	runtimes, err := resolver.Resolve(
		[]RuntimeTarget{
			{
				Target: TargetAll,
			},
		},
		nil,
	)

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
	runtimes, err := resolver.Resolve(
		[]RuntimeTarget{
			{
				Target: TargetAll,
			},
		},
		nil,
	)

	// then
	assert.NotNil(t, err)
	assert.Len(t, runtimes, 0)
}
