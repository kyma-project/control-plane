package environmentscleanup

import (
	"bytes"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	mocks "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/environmentscleanup/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	fixInstanceID1 = "instance-1"
	fixInstanceID2 = "instance-2"
	fixInstanceID3 = "instance-3"
	fixRuntimeID1  = "runtime-1"
	fixRuntimeID2  = "runtime-2"
	fixRuntimeID3  = "rntime-3"
	fixOperationID = "operation-id"

	fixAccountID       = "account-id"
	maxShootAge        = 24 * time.Hour
	shootLabelSelector = "owner.do-not-delete!=true"
)

func TestService_PerformCleanup(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		gcMock := &mocks.GardenerClient{}
		gcMock.On("List", mock.AnythingOfType("v1.ListOptions")).Return(fixShootList(), nil)
		bcMock := &mocks.BrokerClient{}
		bcMock.On("Deprovision", mock.AnythingOfType("internal.Instance")).Return(fixOperationID, nil)
		pMock := &mocks.ProvisionerClient{}
		pMock.On("DeprovisionRuntime", fixAccountID, fixRuntimeID3).Return("", nil)

		memoryStorage := storage.NewMemoryStorage()
		memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID1,
			RuntimeID:  fixRuntimeID1,
		})
		memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID2,
			RuntimeID:  fixRuntimeID2,
		})
		logger := logrus.New()

		svc := NewService(gcMock, bcMock, pMock, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

		// when
		err := svc.PerformCleanup()

		// then
		bcMock.AssertExpectations(t)
		gcMock.AssertExpectations(t)
		pMock.AssertExpectations(t)
		assert.NoError(t, err)
	})

	t.Run("should fail when unable to fetch shoots from gardener", func(t *testing.T) {
		// given
		gcMock := &mocks.GardenerClient{}
		gcMock.On("List", mock.AnythingOfType("v1.ListOptions")).Return(&unstructured.UnstructuredList{}, errors.New("failed to reach gardener"))
		bcMock := &mocks.BrokerClient{}
		pMock := &mocks.ProvisionerClient{}

		memoryStorage := storage.NewMemoryStorage()
		logger := logrus.New()

		svc := NewService(gcMock, bcMock, pMock, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

		// when
		err := svc.PerformCleanup()

		// then
		bcMock.AssertExpectations(t)
		gcMock.AssertExpectations(t)
		assert.Error(t, err)
	})

	t.Run("should return error when unable to find instance in db", func(t *testing.T) {
		// given
		gcMock := &mocks.GardenerClient{}
		gcMock.On("List", mock.AnythingOfType("v1.ListOptions")).Return(fixShootList(), nil)
		bcMock := &mocks.BrokerClient{}
		pMock := &mocks.ProvisionerClient{}

		memoryStorage := storage.NewMemoryStorage()
		memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: "some-instance-id",
			RuntimeID:  "not-matching-id",
		})
		logger := logrus.New()

		svc := NewService(gcMock, bcMock, pMock, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

		// when
		err := svc.PerformCleanup()

		// then
		bcMock.AssertExpectations(t)
		gcMock.AssertExpectations(t)
		assert.Error(t, err)
	})

	t.Run("should return error on KEB deprovision call failure", func(t *testing.T) {
		// given
		gcMock := &mocks.GardenerClient{}
		gcMock.On("List", mock.AnythingOfType("v1.ListOptions")).Return(fixShootList(), nil)
		bcMock := &mocks.BrokerClient{}
		bcMock.On("Deprovision", mock.AnythingOfType("internal.Instance")).Return("", errors.New("failed to deprovision instance"))
		pMock := &mocks.ProvisionerClient{}

		memoryStorage := storage.NewMemoryStorage()
		memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID1,
			RuntimeID:  fixRuntimeID1,
		})
		memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID2,
			RuntimeID:  fixRuntimeID2,
		})

		memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID3,
			RuntimeID:  fixRuntimeID3,
		})

		logger := logrus.New()

		svc := NewService(gcMock, bcMock, pMock, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

		// when
		err := svc.PerformCleanup()

		// then
		bcMock.AssertExpectations(t)
		gcMock.AssertExpectations(t)
		pMock.AssertExpectations(t)
		assert.Error(t, err)
	})

	t.Run("should return error on Provisioner deprovision call failure", func(t *testing.T) {
		// given
		gcMock := &mocks.GardenerClient{}
		gcMock.On("List", mock.AnythingOfType("v1.ListOptions")).Return(fixShootList(), nil)
		bcMock := &mocks.BrokerClient{}
		pMock := &mocks.ProvisionerClient{}
		bcMock.On("Deprovision", mock.AnythingOfType("internal.Instance")).Return("", nil)
		pMock.On("DeprovisionRuntime", fixAccountID, fixRuntimeID2).Return("", errors.New("some error"))
		pMock.On("DeprovisionRuntime", fixAccountID, fixRuntimeID3).Return("", errors.New("some other error"))

		memoryStorage := storage.NewMemoryStorage()
		memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID1,
			RuntimeID:  fixRuntimeID1,
		})

		logger := logrus.New()

		svc := NewService(gcMock, bcMock, pMock, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

		// when
		err := svc.PerformCleanup()

		// then
		bcMock.AssertExpectations(t)
		gcMock.AssertExpectations(t)
		pMock.AssertExpectations(t)
		assert.Error(t, err)
	})

	t.Run("should pass when shoot has no runtime id annotation or account label", func(t *testing.T) {
		// given
		gcMock := &mocks.GardenerClient{}
		creationTime, parseErr := time.Parse(time.RFC3339, "2020-01-02T10:00:00-05:00")
		require.NoError(t, parseErr)
		unl := unstructured.UnstructuredList{
			Items: []unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{
							"name":              "az-1234",
							"creationTimestamp": creationTime,
							"annotations": map[string]interface{}{
								shootAnnotationRuntimeId: fixRuntimeID1,
							},
							"clusterName": "cluster-one",
						},
						"spec": map[string]interface{}{
							"cloudProfileName": "az",
						},
					},
				},
				{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{
							"name":              "az-1234",
							"creationTimestamp": creationTime,
							"clusterName":       "cluster-one",
						},
						"spec": map[string]interface{}{
							"cloudProfileName": "az",
						},
					},
				},
			},
		}
		gcMock.On("List", mock.AnythingOfType("v1.ListOptions")).Return(&unl, nil)
		bcMock := &mocks.BrokerClient{}
		pMock := &mocks.ProvisionerClient{}

		memoryStorage := storage.NewMemoryStorage()
		memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID1,
			RuntimeID:  fixRuntimeID1,
		})

		var actualLog bytes.Buffer
		logger := logrus.New()
		logger.SetFormatter(&logrus.TextFormatter{
			DisableTimestamp: true,
		})
		logger.SetOutput(&actualLog)
		shouldContain := "has no runtime-id annotation"

		svc := NewService(gcMock, bcMock, pMock, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

		// when
		err := svc.PerformCleanup()

		// then
		bcMock.AssertExpectations(t)
		gcMock.AssertExpectations(t)
		assert.Contains(t, actualLog.String(), shouldContain)
		assert.NoError(t, err)
	})
}

func fixShootList() *unstructured.UnstructuredList {
	return &unstructured.UnstructuredList{
		Items: fixShootListItems(),
	}
}

func fixShootListItems() []unstructured.Unstructured {
	creationTime, _ := time.Parse(time.RFC3339, "2020-01-02T10:00:00-05:00")
	unl := unstructured.UnstructuredList{
		Items: []unstructured.Unstructured{
			{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":              "az-1234",
						"creationTimestamp": creationTime,
						"labels": map[string]interface{}{
							"should-be-deleted": "true",
							shootLabelAccountId: fixAccountID,
						},
						"annotations": map[string]interface{}{
							shootAnnotationRuntimeId: fixRuntimeID1,
						},
						"clusterName": "cluster-one",
					},
					"spec": map[string]interface{}{
						"cloudProfileName": "az",
					},
				},
			},
			{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":              "gcp-1234",
						"creationTimestamp": creationTime,
						"labels": map[string]interface{}{
							"should-be-deleted": "true",
							shootLabelAccountId: fixAccountID,
						},
						"annotations": map[string]interface{}{
							shootAnnotationRuntimeId: fixRuntimeID2,
						},
						"clusterName": "cluster-two",
					},
					"spec": map[string]interface{}{
						"cloudProfileName": "gcp",
					},
				},
			},
			{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":              "az-4567",
						"creationTimestamp": creationTime,
						"labels": map[string]interface{}{
							"should-be-deleted-by-provisioner": "true",
							shootLabelAccountId:                fixAccountID,
						},
						"annotations": map[string]interface{}{
							shootAnnotationRuntimeId: fixRuntimeID3,
						},
						"clusterName": "cluster-one",
					},
					"spec": map[string]interface{}{
						"cloudProfileName": "az",
					},
				},
			},
		},
	}
	return unl.Items
}
