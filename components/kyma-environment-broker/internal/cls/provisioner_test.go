package cls

import (
	"errors"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	smautomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager/automock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestProvisionCreatesNewInstanceIfNoneFoundInDB(t *testing.T) {
	const (
		fakeGlobalAccountID = "fake-global-account-id"
		fakeSubAccountID    = "fake-sub-account-id"
		fakeBrokerID        = "fake-broker-id"
		fakeServiceID       = "fake-service-id"
		fakePlanID          = "fake-plan-id"
		fakeInstanceID      = "fake-instance-id"
	)

	storageMock := &automock.InstanceStorage{}
	storageMock.On("FindInstance", fakeGlobalAccountID).Return(nil, false, nil)
	storageMock.On("InsertInstance", mock.Anything).Return(nil)

	smClientMock := &smautomock.Client{}
	creatorMock := &automock.InstanceCreator{}
	creatorMock.On("CreateInstance", smClientMock, fakeBrokerID, fakeServiceID, fakePlanID).Return(fakeInstanceID, nil)

	sut := NewProvisioner(storageMock, creatorMock, logger.NewLogDummy())
	result, err := sut.ProvisionIfNoneExists(smClientMock, &ProvisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SubAccountID:    fakeSubAccountID,
		BrokerID:        fakeBrokerID,
		ServiceID:       fakeServiceID,
		PlanID:          fakePlanID,
	})
	require.NotNil(t, result)
	require.NoError(t, err)

	creatorMock.AssertNumberOfCalls(t, "CreateInstance", 1)
}

func TestProvisionDoesNotCreateNewInstanceIfDBQueryFails(t *testing.T) {
	const (
		fakeGlobalAccountID = "fake-global-account-id"
		fakeSubAccountID    = "fake-sub-account-id"
		fakeBrokerID        = "fake-broker-id"
		fakeServiceID       = "fake-service-id"
		fakePlanID          = "fake-plan-id"
		fakeInstanceID      = "fake-instance-id"
	)

	storageMock := &automock.InstanceStorage{}
	storageMock.On("FindInstance", fakeGlobalAccountID).Return(nil, false, errors.New("unable to connect"))

	smClientMock := &smautomock.Client{}
	creatorMock := &automock.InstanceCreator{}

	sut := NewProvisioner(storageMock, creatorMock, logger.NewLogDummy())
	result, err := sut.ProvisionIfNoneExists(smClientMock, &ProvisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SubAccountID:    fakeSubAccountID,
		BrokerID:        fakeBrokerID,
		ServiceID:       fakeServiceID,
		PlanID:          fakePlanID,
	})
	require.Nil(t, result)
	require.Error(t, err)

	creatorMock.AssertNumberOfCalls(t, "CreateInstance", 0)
}

func TestProvisionSavesNewInstanceToDB(t *testing.T) {
	const (
		fakeGlobalAccountID = "fake-global-account-id"
		fakeSubAccountID    = "fake-sub-account-id"
		fakeBrokerID        = "fake-broker-id"
		fakeServiceID       = "fake-service-id"
		fakePlanID          = "fake-plan-id"
		fakeInstanceID      = "fake-instance-id"
	)

	storageMock := &automock.InstanceStorage{}
	storageMock.On("FindInstance", fakeGlobalAccountID).Return(nil, false, nil)
	storageMock.On("InsertInstance", mock.MatchedBy(func(instance internal.CLSInstance) bool {
		return assert.Equal(t, fakeGlobalAccountID, instance.GlobalAccountID) &&
			assert.Equal(t, fakeInstanceID, instance.ID) &&
			assert.Len(t, instance.SubAccountRefs, 1) &&
			assert.Equal(t, fakeSubAccountID, instance.SubAccountRefs[0])
	})).Return(nil).Once()

	smClientMock := &smautomock.Client{}
	creatorMock := &automock.InstanceCreator{}
	creatorMock.On("CreateInstance", smClientMock, fakeBrokerID, fakeServiceID, fakePlanID).Return(fakeInstanceID, nil)

	sut := NewProvisioner(storageMock, creatorMock, logger.NewLogDummy())
	sut.ProvisionIfNoneExists(smClientMock, &ProvisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SubAccountID:    fakeSubAccountID,
		BrokerID:        fakeBrokerID,
		ServiceID:       fakeServiceID,
		PlanID:          fakePlanID,
	})

	storageMock.AssertNumberOfCalls(t, "InsertInstance", 1)
}

func TestProvisionKeepsSavingNewInstanceToDBIfItFails(t *testing.T) {
	const (
		fakeGlobalAccountID = "fake-global-account-id"
		fakeSubAccountID    = "fake-sub-account-id"
		fakeBrokerID        = "fake-broker-id"
		fakeServiceID       = "fake-service-id"
		fakePlanID          = "fake-plan-id"
		fakeInstanceID      = "fake-instance-id"
	)

	storageMock := &automock.InstanceStorage{}
	storageMock.On("FindInstance", fakeGlobalAccountID).Return(nil, false, nil)
	//simulate a DB connection problem that resolves itself in the succeeding call
	storageMock.On("InsertInstance", mock.Anything).Return(errors.New("unable to connect")).Once()
	storageMock.On("InsertInstance", mock.Anything).Return(nil).Once()

	smClientMock := &smautomock.Client{}
	creatorMock := &automock.InstanceCreator{}
	creatorMock.On("CreateInstance", smClientMock, fakeBrokerID, fakeServiceID, fakePlanID).Return(fakeInstanceID, nil)

	sut := NewProvisioner(storageMock, creatorMock, logger.NewLogDummy())
	sut.ProvisionIfNoneExists(smClientMock, &ProvisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SubAccountID:    fakeSubAccountID,
		BrokerID:        fakeBrokerID,
		ServiceID:       fakeServiceID,
		PlanID:          fakePlanID,
	})

	storageMock.AssertNumberOfCalls(t, "InsertInstance", 2)
}

func TestProvisionAddsReferenceIfFoundInDB(t *testing.T) {
	const (
		fakeGlobalAccountID = "fake-global-account-id"
		fakeSubAccountID    = "fake-sub-account-id"
		fakeBrokerID        = "fake-broker-id"
		fakeServiceID       = "fake-service-id"
		fakePlanID          = "fake-plan-id"
		fakeInstanceID      = "fake-instance-id"
	)

	storageMock := &automock.InstanceStorage{}
	storageMock.On("FindInstance", fakeGlobalAccountID).Return(&internal.CLSInstance{
		GlobalAccountID: fakeGlobalAccountID,
		ID:              fakeInstanceID,
	}, true, nil)
	storageMock.On("AddReference", fakeGlobalAccountID, fakeSubAccountID).Return(nil)

	smClientMock := &smautomock.Client{}
	creatorMock := &automock.InstanceCreator{}

	sut := NewProvisioner(storageMock, creatorMock, logger.NewLogDummy())
	result, err := sut.ProvisionIfNoneExists(smClientMock, &ProvisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SubAccountID:    fakeSubAccountID,
		BrokerID:        fakeBrokerID,
		ServiceID:       fakeServiceID,
		PlanID:          fakePlanID,
	})
	require.NotNil(t, result)
	require.NoError(t, err)

	storageMock.AssertNumberOfCalls(t, "AddReference", 1)
	storageMock.AssertNumberOfCalls(t, "InsertInstance", 0)
	creatorMock.AssertNumberOfCalls(t, "CreateInstance", 0)
}
