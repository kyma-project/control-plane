package cls

import (
	"errors"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cls/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager"
	smautomock "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/servicemanager/automock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestProvisionReturnsExistingInstanceIfFoundInDB(t *testing.T) {
	const (
		fakeGlobalAccountID = "fake-global-account-id"
		fakeSKRInstanceID   = "fake-skr-instance-id"
		fakeRegion          = "westmongolia"
		fakeInstanceID      = "fake-instance-id"
		fakeBrokerID        = "fake-broker-id"
		fakeServiceID       = "fake-service-id"
		fakePlanID          = "fake-plan-id"
	)

	storageMock := &automock.ProvisionerStorage{}
	storageMock.On("FindInstance", fakeGlobalAccountID).Return(&internal.CLSInstance{
		ID:              fakeInstanceID,
		GlobalAccountID: fakeGlobalAccountID,
		Region:          fakeRegion,
	}, true, nil)
	storageMock.On("Reference", mock.Anything, fakeGlobalAccountID, fakeSKRInstanceID).Return(nil)

	smClientMock := &smautomock.Client{}
	creatorMock := &automock.InstanceCreator{}

	sut := NewProvisioner(storageMock, creatorMock, logger.NewLogDummy())
	result, err := sut.Provision(smClientMock, &ProvisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SKRInstanceID:   fakeSKRInstanceID,
		BrokerID:        fakeBrokerID,
		ServiceID:       fakeServiceID,
		PlanID:          fakePlanID,
	})
	require.NotNil(t, result)
	require.NoError(t, err)
	require.Equal(t, fakeInstanceID, result.InstanceID)
	require.False(t, result.ProvisioningTriggered)
	require.Equal(t, fakeRegion, result.Region)
}

func TestProvisionCreatesNewInstanceIfNoneFoundInDB(t *testing.T) {
	const (
		fakeGlobalAccountID = "fake-global-account-id"
		fakeSKRInstanceID   = "fake-skr-instance-id"
		fakeBrokerID        = "fake-broker-id"
		fakeServiceID       = "fake-service-id"
		fakePlanID          = "fake-plan-id"
		fakeRegion          = "westmongolia"
	)

	storageMock := &automock.ProvisionerStorage{}
	storageMock.On("FindInstance", fakeGlobalAccountID).Return(nil, false, nil)
	storageMock.On("InsertInstance", mock.Anything).Return(nil)

	smClientMock := &smautomock.Client{}
	creatorMock := &automock.InstanceCreator{}
	creatorMock.On("CreateInstance", smClientMock, mock.MatchedBy(func(instance servicemanager.InstanceKey) bool {
		return assert.Equal(t, fakeBrokerID, instance.BrokerID) &&
			assert.Equal(t, fakeServiceID, instance.ServiceID) &&
			assert.Equal(t, fakePlanID, instance.PlanID) &&
			isValidUUID(instance.InstanceID)
	})).Return(nil)

	sut := NewProvisioner(storageMock, creatorMock, logger.NewLogDummy())
	result, err := sut.Provision(smClientMock, &ProvisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SKRInstanceID:   fakeSKRInstanceID,
		Region:          fakeRegion,
		BrokerID:        fakeBrokerID,
		ServiceID:       fakeServiceID,
		PlanID:          fakePlanID,
	})
	require.NotNil(t, result)
	require.NoError(t, err)
	require.NotEmpty(t, result.InstanceID)
	require.True(t, result.ProvisioningTriggered)
	require.Equal(t, fakeRegion, result.Region)

	creatorMock.AssertNumberOfCalls(t, "CreateInstance", 1)
}

func TestProvisionDoesNotCreateNewInstanceIfFindQueryFails(t *testing.T) {
	const (
		fakeGlobalAccountID = "fake-global-account-id"
		fakeSKRInstanceID   = "fake-skr-instance-id"
		fakeBrokerID        = "fake-broker-id"
		fakeServiceID       = "fake-service-id"
		fakePlanID          = "fake-plan-id"
		fakeInstanceID      = "fake-instance-id"
	)

	storageMock := &automock.ProvisionerStorage{}
	storageMock.On("FindInstance", fakeGlobalAccountID).Return(nil, false, errors.New("unable to connect"))

	smClientMock := &smautomock.Client{}
	creatorMock := &automock.InstanceCreator{}

	sut := NewProvisioner(storageMock, creatorMock, logger.NewLogDummy())
	result, err := sut.Provision(smClientMock, &ProvisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SKRInstanceID:   fakeSKRInstanceID,
		BrokerID:        fakeBrokerID,
		ServiceID:       fakeServiceID,
		PlanID:          fakePlanID,
	})
	require.Nil(t, result)
	require.Error(t, err)

	creatorMock.AssertNumberOfCalls(t, "CreateInstance", 0)
}

func TestProvisionDoesNotCreateNewInstanceIfInsertQueryFails(t *testing.T) {
	const (
		fakeGlobalAccountID = "fake-global-account-id"
		fakeSKRInstanceID   = "fake-skr-instance-id"
		fakeBrokerID        = "fake-broker-id"
		fakeServiceID       = "fake-service-id"
		fakePlanID          = "fake-plan-id"
		fakeInstanceID      = "fake-instance-id"
	)

	storageMock := &automock.ProvisionerStorage{}
	storageMock.On("FindInstance", fakeGlobalAccountID).Return(nil, false, nil)
	storageMock.On("InsertInstance", mock.Anything).Return(errors.New("unable to connect"))

	smClientMock := &smautomock.Client{}
	creatorMock := &automock.InstanceCreator{}

	sut := NewProvisioner(storageMock, creatorMock, logger.NewLogDummy())
	result, err := sut.Provision(smClientMock, &ProvisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SKRInstanceID:   fakeSKRInstanceID,
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
		fakeSKRInstanceID   = "fake-skr-instance-id"
		fakeBrokerID        = "fake-broker-id"
		fakeServiceID       = "fake-service-id"
		fakePlanID          = "fake-plan-id"
	)

	storageMock := &automock.ProvisionerStorage{}
	storageMock.On("FindInstance", fakeGlobalAccountID).Return(nil, false, nil)
	storageMock.On("InsertInstance", mock.MatchedBy(func(instance internal.CLSInstance) bool {
		return assert.Equal(t, fakeGlobalAccountID, instance.GlobalAccountID) &&
			assert.NotEmpty(t, instance.ID) &&
			assert.Len(t, instance.SKRReferences, 1) &&
			assert.Equal(t, fakeSKRInstanceID, instance.SKRReferences[0])
	})).Return(nil).Once()

	smClientMock := &smautomock.Client{}
	creatorMock := &automock.InstanceCreator{}
	creatorMock.On("CreateInstance", smClientMock, mock.Anything).Return(nil)

	sut := NewProvisioner(storageMock, creatorMock, logger.NewLogDummy())
	result, err := sut.Provision(smClientMock, &ProvisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SKRInstanceID:   fakeSKRInstanceID,
		BrokerID:        fakeBrokerID,
		ServiceID:       fakeServiceID,
		PlanID:          fakePlanID,
	})
	require.NotNil(t, result)
	require.NoError(t, err)

	storageMock.AssertNumberOfCalls(t, "InsertInstance", 1)
}

func TestProvisionAddsReferenceIfFoundInDB(t *testing.T) {
	const (
		fakeGlobalAccountID = "fake-global-account-id"
		fakeSKRInstanceID   = "fake-skr-instance-id"
		fakeBrokerID        = "fake-broker-id"
		fakeServiceID       = "fake-service-id"
		fakePlanID          = "fake-plan-id"
		fakeInstanceID      = "fake-instance-id"
	)

	storageMock := &automock.ProvisionerStorage{}
	storageMock.On("FindInstance", fakeGlobalAccountID).Return(&internal.CLSInstance{
		GlobalAccountID: fakeGlobalAccountID,
		ID:              fakeInstanceID,
	}, true, nil)
	storageMock.On("Reference", mock.Anything, fakeGlobalAccountID, fakeSKRInstanceID).Return(nil)

	smClientMock := &smautomock.Client{}
	creatorMock := &automock.InstanceCreator{}

	sut := NewProvisioner(storageMock, creatorMock, logger.NewLogDummy())
	result, err := sut.Provision(smClientMock, &ProvisionRequest{
		GlobalAccountID: fakeGlobalAccountID,
		SKRInstanceID:   fakeSKRInstanceID,
		BrokerID:        fakeBrokerID,
		ServiceID:       fakeServiceID,
		PlanID:          fakePlanID,
	})
	require.NotNil(t, result)
	require.NoError(t, err)

	storageMock.AssertNumberOfCalls(t, "Reference", 1)
	storageMock.AssertNumberOfCalls(t, "InsertInstance", 0)
	creatorMock.AssertNumberOfCalls(t, "CreateInstance", 0)
}
