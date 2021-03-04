package cls

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	smautomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDeprovisionFailsIfFindQueryFails(t *testing.T) {
	// given
	fakeSKRInstanceID := "fake-skr-instance-id"
	fakeInstance := servicemanager.InstanceKey{
		BrokerID:   "fake-broker-id",
		ServiceID:  "fake-service-id",
		PlanID:     "fake-plan-id",
		InstanceID: "fake-instance-id",
	}

	storageMock := &automock.DeprovisionerStorage{}
	storageMock.On("FindByID", fakeInstance.InstanceID).Return(nil, false, errors.New("unable to connect"))

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
	}

	smClientMock := &smautomock.Client{}

	// when
	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		SKRInstanceID: fakeSKRInstanceID,
		Instance:      fakeInstance,
	})

	// then
	require.Error(t, err)
}

func TestDeprovisionReturnsEarlyIfCLSNotReferenced(t *testing.T) {
	// given
	fakeSKRInstanceID := "fake-skr-instance-id"
	fakeInstance := servicemanager.InstanceKey{
		BrokerID:   "fake-broker-id",
		ServiceID:  "fake-service-id",
		PlanID:     "fake-plan-id",
		InstanceID: "fake-instance-id",
	}

	found := internal.NewCLSInstance("fake-global-id", "eu",
		internal.WithID(fakeInstance.InstanceID),
		internal.WithReferences("other-fake-skr-instance-id-1", "other-fake-skr-instance-id-2"))
	fakeStorage := storage.NewMemoryStorage().CLSInstances()
	fakeStorage.Insert(*found)

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: fakeStorage,
	}

	smClientMock := &smautomock.Client{}

	// when
	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		SKRInstanceID: fakeSKRInstanceID,
		Instance:      fakeInstance,
	})

	// then
	require.NoError(t, err)
}

func TestDeprovisionUnreferencesIfNotLastReference(t *testing.T) {
	// given
	firstFakeSKRInstanceID := "fake-skr-instance-id-1"
	secondFakeSKRInstanceID := "fake-skr-instance-id-2"
	fakeInstance := servicemanager.InstanceKey{
		BrokerID:   "fake-broker-id",
		ServiceID:  "fake-service-id",
		PlanID:     "fake-plan-id",
		InstanceID: "fake-instance-id",
	}

	found := internal.NewCLSInstance("fake-global-id", "eu",
		internal.WithID(fakeInstance.InstanceID),
		internal.WithReferences(firstFakeSKRInstanceID, secondFakeSKRInstanceID))
	fakeStorage := storage.NewMemoryStorage().CLSInstances()
	fakeStorage.Insert(*found)

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: fakeStorage,
	}

	smClientMock := &smautomock.Client{}

	// when
	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		SKRInstanceID: secondFakeSKRInstanceID,
		Instance:      fakeInstance,
	})

	// then
	require.NoError(t, err)

	instance, exists, _ := fakeStorage.FindByID(fakeInstance.InstanceID)
	require.True(t, exists)
	require.ElementsMatch(t, instance.References(), []string{firstFakeSKRInstanceID})
}

func TestDeprovisionFailsIfUpdateQueryFailsAfterUnreferencing(t *testing.T) {
	// given
	fakeSKRInstanceID := "fake-skr-instance-id"
	fakeInstance := servicemanager.InstanceKey{
		BrokerID:   "fake-broker-id",
		ServiceID:  "fake-service-id",
		PlanID:     "fake-plan-id",
		InstanceID: "fake-instance-id",
	}

	found := internal.NewCLSInstance("fake-global-id", "eu",
		internal.WithID(fakeInstance.InstanceID),
		internal.WithReferences(fakeSKRInstanceID))
	storageMock := &automock.DeprovisionerStorage{}
	storageMock.On("FindByID", fakeInstance.InstanceID).Return(found, true, nil)
	storageMock.On("Update", mock.Anything).Return(errors.New("unable to connect"))

	smClientMock := &smautomock.Client{}
	removerMock := &automock.InstanceRemover{}
	removerMock.On("RemoveInstance", smClientMock, fakeInstance).Return(nil)

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
		remover: removerMock,
	}

	// when
	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		SKRInstanceID: fakeSKRInstanceID,
		Instance:      fakeInstance,
	})

	// then
	require.Error(t, err)
	removerMock.AssertNumberOfCalls(t, "RemoveInstance", 0)
}

func TestDeprovisionRemovesIfLastReference(t *testing.T) {
	// given
	fakeSKRInstanceID := "fake-skr-instance-id"
	fakeInstance := servicemanager.InstanceKey{
		BrokerID:   "fake-broker-id",
		ServiceID:  "fake-service-id",
		PlanID:     "fake-plan-id",
		InstanceID: "fake-instance-id",
	}

	found := internal.NewCLSInstance("fake-global-id", "eu",
		internal.WithID(fakeInstance.InstanceID),
		internal.WithReferences(fakeSKRInstanceID))
	fakeStorage := storage.NewMemoryStorage().CLSInstances()
	fakeStorage.Insert(*found)

	smClientMock := &smautomock.Client{}
	removerMock := &automock.InstanceRemover{}
	removerMock.On("RemoveInstance", smClientMock, fakeInstance).Return(nil)

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: fakeStorage,
		remover: removerMock,
	}

	// when
	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		SKRInstanceID: fakeSKRInstanceID,
		Instance:      fakeInstance,
	})

	// then
	require.NoError(t, err)

	_, exists, _ := fakeStorage.FindByID(fakeInstance.InstanceID)
	require.False(t, exists)
}

func TestDeprovisionFailsIfUpdateQueryFails(t *testing.T) {
	// given
	fakeSKRInstanceID := "fake-skr-instance-id"
	fakeInstance := servicemanager.InstanceKey{
		BrokerID:   "fake-broker-id",
		ServiceID:  "fake-service-id",
		PlanID:     "fake-plan-id",
		InstanceID: "fake-instance-id",
	}

	found := internal.NewCLSInstance("fake-global-id", "eu",
		internal.WithID(fakeInstance.InstanceID),
		internal.WithReferences(fakeSKRInstanceID))
	storageMock := &automock.DeprovisionerStorage{}
	storageMock.On("FindByID", fakeInstance.InstanceID).Return(found, true, nil)
	storageMock.On("Update", mock.Anything).Return(errors.New("unable to connect"))

	smClientMock := &smautomock.Client{}
	removerMock := &automock.InstanceRemover{}
	removerMock.On("RemoveInstance", smClientMock, fakeInstance).Return(nil)

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
		remover: removerMock,
	}

	// when
	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		SKRInstanceID: fakeSKRInstanceID,
		Instance:      fakeInstance,
	})

	// then
	require.Error(t, err)
	removerMock.AssertNumberOfCalls(t, "RemoveInstance", 0)
}

func TestDeprovisionRemovesInstanceIfLastReference(t *testing.T) {
	// given
	fakeSKRInstanceID := "fake-skr-instance-id"
	fakeInstance := servicemanager.InstanceKey{
		BrokerID:   "fake-broker-id",
		ServiceID:  "fake-service-id",
		PlanID:     "fake-plan-id",
		InstanceID: "fake-instance-id",
	}

	found := internal.NewCLSInstance("fake-global-id", "eu",
		internal.WithID(fakeInstance.InstanceID),
		internal.WithReferences(fakeSKRInstanceID))
	fakeStorage := storage.NewMemoryStorage().CLSInstances()
	fakeStorage.Insert(*found)

	smClientMock := &smautomock.Client{}
	removerMock := &automock.InstanceRemover{}
	removerMock.On("RemoveInstance", smClientMock, fakeInstance).Return(nil)

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: fakeStorage,
		remover: removerMock,
	}

	// when
	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		SKRInstanceID: fakeSKRInstanceID,
		Instance:      fakeInstance,
	})

	// then
	require.NoError(t, err)
	removerMock.AssertNumberOfCalls(t, "RemoveInstance", 1)
}
