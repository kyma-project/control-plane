package cls

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	smautomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager/automock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestDeprovisionFailsIfFindQueryFails(t *testing.T) {
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
	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		SKRInstanceID: fakeSKRInstanceID,
		Instance:      fakeInstance,
	})

	require.Error(t, err)
}

func TestDeprovisionReturnsEarlyIfCLSNotReferenced(t *testing.T) {
	fakeSKRInstanceID := "fake-skr-instance-id"
	fakeInstance := servicemanager.InstanceKey{
		BrokerID:   "fake-broker-id",
		ServiceID:  "fake-service-id",
		PlanID:     "fake-plan-id",
		InstanceID: "fake-instance-id",
	}

	storageMock := &automock.DeprovisionerStorage{}
	found := &internal.CLSInstance{
		ID:                       fakeInstance.InstanceID,
		Version:                  42,
		ReferencedSKRInstanceIDs: []string{"other-fake-skr-instance-id-1", "other-fake-skr-instance-id-2"},
	}
	storageMock.On("FindByID", fakeInstance.InstanceID).Return(found, true, nil)

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
	}

	smClientMock := &smautomock.Client{}
	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		SKRInstanceID: fakeSKRInstanceID,
		Instance:      fakeInstance,
	})

	require.NoError(t, err)
}

func TestDeprovisionUnreferencesIfNotLastReference(t *testing.T) {
	fakeSKRInstanceID := "fake-skr-instance-id"
	fakeInstance := servicemanager.InstanceKey{
		BrokerID:   "fake-broker-id",
		ServiceID:  "fake-service-id",
		PlanID:     "fake-plan-id",
		InstanceID: "fake-instance-id",
	}

	storageMock := &automock.DeprovisionerStorage{}
	found := &internal.CLSInstance{
		ID:                       fakeInstance.InstanceID,
		Version:                  42,
		ReferencedSKRInstanceIDs: []string{fakeSKRInstanceID, "other-fake-skr-instance-id"},
	}
	storageMock.On("FindByID", fakeInstance.InstanceID).Return(found, true, nil)
	storageMock.On("Unreference", found.Version, fakeInstance.InstanceID, fakeSKRInstanceID).Return(nil)

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
	}

	smClientMock := &smautomock.Client{}
	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		SKRInstanceID: fakeSKRInstanceID,
		Instance:      fakeInstance,
	})

	require.NoError(t, err)
	storageMock.AssertNumberOfCalls(t, "Unreference", 1)
}

func TestDeprovisionFailsIfUnreferenceQueryFails(t *testing.T) {
	fakeSKRInstanceID := "fake-skr-instance-id"
	fakeInstance := servicemanager.InstanceKey{
		BrokerID:   "fake-broker-id",
		ServiceID:  "fake-service-id",
		PlanID:     "fake-plan-id",
		InstanceID: "fake-instance-id",
	}

	storageMock := &automock.DeprovisionerStorage{}
	found := &internal.CLSInstance{
		ID:                       fakeInstance.InstanceID,
		Version:                  42,
		ReferencedSKRInstanceIDs: []string{fakeSKRInstanceID, "other-fake-skr-instance-id"},
	}
	storageMock.On("FindByID", fakeInstance.InstanceID).Return(found, true, nil)
	storageMock.On("Unreference", found.Version, fakeInstance.InstanceID, fakeSKRInstanceID).Return(errors.New("unable to connect"))

	smClientMock := &smautomock.Client{}
	removerMock := &automock.InstanceRemover{}
	removerMock.On("RemoveInstance", smClientMock, fakeInstance).Return(nil)

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
		remover: removerMock,
	}

	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		SKRInstanceID: fakeSKRInstanceID,
		Instance:      fakeInstance,
	})

	require.Error(t, err)
	removerMock.AssertNumberOfCalls(t, "RemoveInstance", 0)
}

func TestDeprovisionMarksAsBeingRemovedIfLastReference(t *testing.T) {
	fakeSKRInstanceID := "fake-skr-instance-id"
	fakeInstance := servicemanager.InstanceKey{
		BrokerID:   "fake-broker-id",
		ServiceID:  "fake-service-id",
		PlanID:     "fake-plan-id",
		InstanceID: "fake-instance-id",
	}

	storageMock := &automock.DeprovisionerStorage{}
	found := &internal.CLSInstance{
		ID:                       fakeInstance.InstanceID,
		Version:                  42,
		ReferencedSKRInstanceIDs: []string{fakeSKRInstanceID},
	}
	storageMock.On("FindByID", fakeInstance.InstanceID).Return(found, true, nil)
	storageMock.On("MarkAsBeingRemoved", found.Version, fakeInstance.InstanceID, fakeSKRInstanceID).Return(nil)
	storageMock.On("Remove", fakeInstance.InstanceID).Return(nil)

	smClientMock := &smautomock.Client{}
	removerMock := &automock.InstanceRemover{}
	removerMock.On("RemoveInstance", smClientMock, fakeInstance).Return(nil)

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
		remover: removerMock,
	}

	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		SKRInstanceID: fakeSKRInstanceID,
		Instance:      fakeInstance,
	})

	require.NoError(t, err)
	storageMock.AssertNumberOfCalls(t, "MarkAsBeingRemoved", 1)
	storageMock.AssertNumberOfCalls(t, "Remove", 1)
}

func TestDeprovisionFailsIfMarkingQueryFails(t *testing.T) {
	fakeSKRInstanceID := "fake-skr-instance-id"
	fakeInstance := servicemanager.InstanceKey{
		BrokerID:   "fake-broker-id",
		ServiceID:  "fake-service-id",
		PlanID:     "fake-plan-id",
		InstanceID: "fake-instance-id",
	}

	storageMock := &automock.DeprovisionerStorage{}
	found := &internal.CLSInstance{
		ID:                       fakeInstance.InstanceID,
		Version:                  42,
		ReferencedSKRInstanceIDs: []string{fakeSKRInstanceID},
	}
	storageMock.On("FindByID", fakeInstance.InstanceID).Return(found, true, nil)
	storageMock.On("MarkAsBeingRemoved", found.Version, fakeInstance.InstanceID, fakeSKRInstanceID).Return(errors.New("unable to connect"))

	smClientMock := &smautomock.Client{}
	removerMock := &automock.InstanceRemover{}
	removerMock.On("RemoveInstance", smClientMock, fakeInstance).Return(nil)

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
		remover: removerMock,
	}

	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		SKRInstanceID: fakeSKRInstanceID,
		Instance:      fakeInstance,
	})

	require.Error(t, err)
	removerMock.AssertNumberOfCalls(t, "RemoveInstance", 0)
}

func TestDeprovisionRemovesIfLastReference(t *testing.T) {
	fakeSKRInstanceID := "fake-skr-instance-id"
	fakeInstance := servicemanager.InstanceKey{
		BrokerID:   "fake-broker-id",
		ServiceID:  "fake-service-id",
		PlanID:     "fake-plan-id",
		InstanceID: "fake-instance-id",
	}

	storageMock := &automock.DeprovisionerStorage{}
	found := &internal.CLSInstance{
		ID:                       fakeInstance.InstanceID,
		Version:                  42,
		ReferencedSKRInstanceIDs: []string{fakeSKRInstanceID},
	}
	storageMock.On("FindByID", fakeInstance.InstanceID).Return(found, true, nil)
	storageMock.On("MarkAsBeingRemoved", found.Version, fakeInstance.InstanceID, fakeSKRInstanceID).Return(nil)
	storageMock.On("Remove", fakeInstance.InstanceID).Return(nil)

	smClientMock := &smautomock.Client{}
	removerMock := &automock.InstanceRemover{}
	removerMock.On("RemoveInstance", smClientMock, fakeInstance).Return(nil)

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
		remover: removerMock,
	}

	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		SKRInstanceID: fakeSKRInstanceID,
		Instance:      fakeInstance,
	})

	require.NoError(t, err)
	removerMock.AssertNumberOfCalls(t, "RemoveInstance", 1)
}
