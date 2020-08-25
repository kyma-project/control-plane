package cis

import (
	"fmt"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	mocks "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/cis/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// subAccountTestIDs contains test data in form: InstanceID : SubAccountID
var subAccountTestIDs = map[string]string{
	"7b84e7e7-62df-412a-9e09-b4581253efba": "e65a807a-488f-4062-9e02-b39090ec0258",
	"c2cf9bf5-948e-4c65-baed-6b92f9da9c8e": "c9200e87-e43a-4988-b058-ab99c73e20cb",
	"ab0addc7-de76-4f2e-a6a7-bfcbd3d4eefe": "b45f63bd-750c-4ede-920f-e2b7c86ca15f",
	"eb3508ca-eec1-4ae1-8968-e78a45daa741": "33cfc39a-a974-4d8d-964f-5be7b1907fbb",
	"07d368f2-c294-47e7-8d66-20b73ef46342": "967f897e-29a0-4576-8bfa-98e532ba04a7",
	"d7348ffa-a7d3-41cf-8a7f-17c92c7b2794": "c5090aad-af2e-406b-bddc-ece72bfa0b7a",
	"cb6034fc-2bae-466f-a98d-5af492edeca2": "f1672806-136e-4b40-9e71-d93a04391274",
	"9e3cbd53-5c5a-410f-8eaa-1c2d75345814": "c249745a-d52f-4be1-ab02-6d59834e50d4",
	"2c448504-6c8a-4d82-bdc2-fa4bd0534f25": "95490855-cf3c-4333-884c-4084f591aa27",
	"ad6af000-e647-44ea-a3bb-db8672d5bc7e": "8664da71-8e3c-47bb-9c29-128038f8a959",
}

func TestSubAccountCleanupService_Run(t *testing.T) {
	t.Run("all instances should be deprovisioned", func(t *testing.T) {
		// Given
		cisClient := &mocks.CisClient{}
		cisClient.On("FetchSubAccountsToDelete").Return(fixSubAccountIDs(), nil)
		defer cisClient.AssertExpectations(t)

		brokerClient := &mocks.BrokerClient{}
		for _, instance := range fixInstances() {
			brokerClient.On("Deprovision", instance).Return("<operationUUID>", nil).Once()
		}
		defer brokerClient.AssertExpectations(t)

		memoryStorage := storage.NewMemoryStorage()
		for _, instance := range fixInstances() {
			err := memoryStorage.Instances().Insert(instance)
			assert.NoError(t, err)
		}

		service := NewSubAccountCleanupService(cisClient, brokerClient, memoryStorage.Instances(), logrus.New())
		service.chunksAmount = 2

		// When
		err := service.Run()

		// Then
		assert.NoError(t, err)
	})

	t.Run("some deprovisioning should failed and warnings should be displayed", func(t *testing.T) {
		// Given
		brokenInstanceIDOne := "07d368f2-c294-47e7-8d66-20b73ef46342"
		brokenInstanceIDTwo := "ad6af000-e647-44ea-a3bb-db8672d5bc7e"

		cisClient := &mocks.CisClient{}
		cisClient.On("FetchSubAccountsToDelete").Return(fixSubAccountIDs(), nil)
		defer cisClient.AssertExpectations(t)

		brokerClient := &mocks.BrokerClient{}
		for _, instance := range fixInstances() {
			if instance.InstanceID == brokenInstanceIDOne || instance.InstanceID == brokenInstanceIDTwo {
				brokerClient.On("Deprovision", instance).Return("", fmt.Errorf("cannot deprovision")).Once()
			} else {
				brokerClient.On("Deprovision", instance).Return("<operationUUID>", nil).Once()
			}
		}
		defer brokerClient.AssertExpectations(t)

		memoryStorage := storage.NewMemoryStorage()
		for _, instance := range fixInstances() {
			err := memoryStorage.Instances().Insert(instance)
			assert.NoError(t, err)
		}

		log := logger.NewLogSpy()
		service := NewSubAccountCleanupService(cisClient, brokerClient, memoryStorage.Instances(), log.Logger)
		service.chunksAmount = 5

		// When
		err := service.Run()

		// Then
		assert.NoError(t, err)
		log.AssertLogged(t, logrus.WarnLevel, "part of deprovisioning process failed with error: error occurred during deprovisioning instance with ID ad6af000-e647-44ea-a3bb-db8672d5bc7e: cannot deprovision")
		log.AssertLogged(t, logrus.WarnLevel, "part of deprovisioning process failed with error: error occurred during deprovisioning instance with ID 07d368f2-c294-47e7-8d66-20b73ef46342: cannot deprovision")
	})

	t.Run("process should return with error", func(t *testing.T) {
		// Given
		cisClient := &mocks.CisClient{}
		cisClient.On("FetchSubAccountsToDelete").Return([]string{}, fmt.Errorf("cannot fetch subaccounts"))
		defer cisClient.AssertExpectations(t)

		brokerClient := &mocks.BrokerClient{}
		memoryStorage := storage.NewMemoryStorage()

		service := NewSubAccountCleanupService(cisClient, brokerClient, memoryStorage.Instances(), logrus.New())
		service.chunksAmount = 7

		// When
		err := service.Run()

		// Then
		assert.Error(t, err)
	})
}

func fixSubAccountIDs() []string {
	subAccountIDs := make([]string, 0)

	for _, subAccountID := range subAccountTestIDs {
		subAccountIDs = append(subAccountIDs, subAccountID)
	}

	return subAccountIDs
}

func fixInstances() []internal.Instance {
	instances := make([]internal.Instance, 0)

	for instanceID, subAccountTestID := range subAccountTestIDs {
		instances = append(instances, internal.Instance{
			InstanceID:   instanceID,
			SubAccountID: subAccountTestID,
		})
	}

	return instances
}
