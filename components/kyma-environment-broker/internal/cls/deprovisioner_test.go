package cls

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	smautomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager/automock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDeprovisionFailsIfFindQueryFails(t *testing.T) {
	const (
		fakeGlobalAccountID = "fake-global-account-id"
		fakeSKRInstanceID   = "fake-skr-instance-id"
	)

	var (
		fakeInstance = servicemanager.InstanceKey{
			BrokerID:   "fake-broker-id",
			ServiceID:  "fake-service-id",
			PlanID:     "fake-plan-id",
			InstanceID: "fake-instance-id",
		}
	)

	storageMock := &automock.DeprovisionerStorage{}
	storageMock.On("FindInstance", fakeGlobalAccountID).Return(nil, false, errors.New("unable to connect"))

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
	}

	smClientMock := &smautomock.Client{}
	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SKRInstanceID:   fakeSKRInstanceID,
		Instance:        fakeInstance,
	})

	require.Error(t, err)
}

func TestDeprovisionReturnsEarlyIfCLSNotReferenced(t *testing.T) {
	const (
		fakeGlobalAccountID = "fake-global-account-id"
		fakeSKRInstanceID   = "fake-skr-instance-id"
	)

	var (
		fakeInstance = servicemanager.InstanceKey{
			BrokerID:   "fake-broker-id",
			ServiceID:  "fake-service-id",
			PlanID:     "fake-plan-id",
			InstanceID: "fake-instance-id",
		}
	)

	storageMock := &automock.DeprovisionerStorage{}
	found := &internal.CLSInstance{
		Version:       42,
		SKRReferences: []string{"other-fake-skr-instance-id-1", "other-fake-skr-instance-id-2"},
	}
	storageMock.On("FindInstance", mock.Anything).Return(found, true, nil)

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
	}

	smClientMock := &smautomock.Client{}
	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SKRInstanceID:   fakeSKRInstanceID,
		Instance:        fakeInstance,
	})

	require.NoError(t, err)
}

func TestDeprovisionUnreferencesIfNotLastReference(t *testing.T) {
	const (
		fakeGlobalAccountID = "fake-global-account-id"
		fakeSKRInstanceID   = "fake-skr-instance-id"
	)

	var (
		fakeInstance = servicemanager.InstanceKey{
			BrokerID:   "fake-broker-id",
			ServiceID:  "fake-service-id",
			PlanID:     "fake-plan-id",
			InstanceID: "fake-instance-id",
		}
	)

	storageMock := &automock.DeprovisionerStorage{}
	found := &internal.CLSInstance{
		Version:       42,
		SKRReferences: []string{fakeSKRInstanceID, "other-fake-skr-instance-id"},
	}
	storageMock.On("FindInstance", mock.Anything).Return(found, true, nil)
	storageMock.On("Unreference", found.Version, fakeGlobalAccountID, fakeSKRInstanceID).Return(nil)

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
	}

	smClientMock := &smautomock.Client{}
	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SKRInstanceID:   fakeSKRInstanceID,
		Instance:        fakeInstance,
	})

	require.NoError(t, err)
	storageMock.AssertNumberOfCalls(t, "Unreference", 1)
}

func TestDeprovisionFailsIfUnreferenceQueryFails(t *testing.T) {
	const (
		fakeGlobalAccountID = "fake-global-account-id"
		fakeSKRInstanceID   = "fake-skr-instance-id"
	)

	var (
		fakeInstance = servicemanager.InstanceKey{
			BrokerID:   "fake-broker-id",
			ServiceID:  "fake-service-id",
			PlanID:     "fake-plan-id",
			InstanceID: "fake-instance-id",
		}
	)

	storageMock := &automock.DeprovisionerStorage{}
	found := &internal.CLSInstance{
		Version:       42,
		SKRReferences: []string{fakeSKRInstanceID, "other-fake-skr-instance-id"},
	}
	storageMock.On("FindInstance", mock.Anything).Return(found, true, nil)
	storageMock.On("Unreference", found.Version, fakeGlobalAccountID, fakeSKRInstanceID).Return(errors.New("unable to connect"))

	smClientMock := &smautomock.Client{}
	removerMock := &automock.InstanceRemover{}
	removerMock.On("RemoveInstance", smClientMock, fakeInstance).Return(nil)

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
		remover: removerMock,
	}

	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SKRInstanceID:   fakeSKRInstanceID,
		Instance:        fakeInstance,
	})

	require.Error(t, err)
	removerMock.AssertNumberOfCalls(t, "DeleteInstance", 0)
}

func TestDeprovisionMarksAsBeingRemovedIfLastReference(t *testing.T) {
	const (
		fakeGlobalAccountID = "fake-global-account-id"
		fakeSKRInstanceID   = "fake-skr-instance-id"
	)

	var (
		fakeInstance = servicemanager.InstanceKey{
			BrokerID:   "fake-broker-id",
			ServiceID:  "fake-service-id",
			PlanID:     "fake-plan-id",
			InstanceID: "fake-instance-id",
		}
	)

	storageMock := &automock.DeprovisionerStorage{}
	found := &internal.CLSInstance{
		Version:       42,
		SKRReferences: []string{fakeSKRInstanceID},
	}
	storageMock.On("FindInstance", mock.Anything).Return(found, true, nil)
	storageMock.On("MarkAsBeingRemoved", found.Version, fakeGlobalAccountID).Return(nil)

	smClientMock := &smautomock.Client{}
	removerMock := &automock.InstanceRemover{}
	removerMock.On("RemoveInstance", smClientMock, fakeInstance).Return(nil)

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
		remover: removerMock,
	}

	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SKRInstanceID:   fakeSKRInstanceID,
		Instance:        fakeInstance,
	})

	require.NoError(t, err)
	storageMock.AssertNumberOfCalls(t, "MarkAsBeingRemoved", 1)
}

func TestDeprovisionFailsIfMarkingQueryFails(t *testing.T) {
	const (
		fakeGlobalAccountID = "fake-global-account-id"
		fakeSKRInstanceID   = "fake-skr-instance-id"
	)

	var (
		fakeInstance = servicemanager.InstanceKey{
			BrokerID:   "fake-broker-id",
			ServiceID:  "fake-service-id",
			PlanID:     "fake-plan-id",
			InstanceID: "fake-instance-id",
		}
	)

	storageMock := &automock.DeprovisionerStorage{}
	found := &internal.CLSInstance{
		Version:       42,
		SKRReferences: []string{fakeSKRInstanceID},
	}
	storageMock.On("FindInstance", mock.Anything).Return(found, true, nil)
	storageMock.On("MarkAsBeingRemoved", found.Version, fakeGlobalAccountID).Return(errors.New("unable to connect"))

	smClientMock := &smautomock.Client{}
	removerMock := &automock.InstanceRemover{}
	removerMock.On("RemoveInstance", smClientMock, fakeInstance).Return(nil)

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
		remover: removerMock,
	}

	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SKRInstanceID:   fakeSKRInstanceID,
		Instance:        fakeInstance,
	})

	require.Error(t, err)
	removerMock.AssertNumberOfCalls(t, "DeleteInstance", 0)
}

func TestDeprovisionDeletesIfLastReference(t *testing.T) {
	const (
		fakeGlobalAccountID = "fake-global-account-id"
		fakeSKRInstanceID   = "fake-skr-instance-id"
	)

	var (
		fakeInstance = servicemanager.InstanceKey{
			BrokerID:   "fake-broker-id",
			ServiceID:  "fake-service-id",
			PlanID:     "fake-plan-id",
			InstanceID: "fake-instance-id",
		}
	)

	storageMock := &automock.DeprovisionerStorage{}
	found := &internal.CLSInstance{
		Version:       42,
		SKRReferences: []string{fakeSKRInstanceID},
	}
	storageMock.On("FindInstance", mock.Anything).Return(found, true, nil)
	storageMock.On("MarkAsBeingRemoved", found.Version, fakeGlobalAccountID).Return(nil)

	smClientMock := &smautomock.Client{}
	removerMock := &automock.InstanceRemover{}
	removerMock.On("RemoveInstance", smClientMock, fakeInstance).Return(nil)

	deprovisioner := &Deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
		remover: removerMock,
	}

	err := deprovisioner.Deprovision(smClientMock, &DeprovisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SKRInstanceID:   fakeSKRInstanceID,
		Instance:        fakeInstance,
	})

	require.NoError(t, err)
	removerMock.AssertNumberOfCalls(t, "RemoveInstance", 1)
}
