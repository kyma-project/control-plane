package cls

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
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

	deprovisioner := &deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
	}

	err := deprovisioner.Deprovision(&DeprovisionRequest{
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

	deprovisioner := &deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
	}

	err := deprovisioner.Deprovision(&DeprovisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SKRInstanceID:   fakeSKRInstanceID,
		Instance:        fakeInstance,
	})

	require.NoError(t, err)
}

func TestDeprovisionRemovesReferenceIfNotLastReference(t *testing.T) {
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

	deprovisioner := &deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
	}

	err := deprovisioner.Deprovision(&DeprovisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SKRInstanceID:   fakeSKRInstanceID,
		Instance:        fakeInstance,
	})

	require.NoError(t, err)
	storageMock.AssertNumberOfCalls(t, "Unreference", 1)
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

	deprovisioner := &deprovisioner{
		log:     logger.NewLogDummy(),
		storage: storageMock,
	}

	err := deprovisioner.Deprovision(&DeprovisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SKRInstanceID:   fakeSKRInstanceID,
		Instance:        fakeInstance,
	})

	require.NoError(t, err)
	storageMock.AssertNumberOfCalls(t, "MarkAsBeingRemoved", 1)
}
