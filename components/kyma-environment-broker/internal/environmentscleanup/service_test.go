package environmentscleanup

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	run "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	mocks "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/environmentscleanup/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		gcMock.On("List", mock.Anything, mock.AnythingOfType("v1.ListOptions")).Return(fixShootList(), nil)
		bcMock := &mocks.BrokerClient{}
		bcMock.On("Deprovision", mock.AnythingOfType("internal.Instance")).Return(fixOperationID, nil)
		brcMock := &mocks.BrokerRuntimesClient{}
		brcMock.On("ListRuntimes", mock.AnythingOfType("runtime.ListParameters")).Return(run.RuntimesPage{}, nil)
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

		svc := NewService(gcMock, bcMock, brcMock, pMock, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

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
		gcMock.On("List", mock.Anything, mock.AnythingOfType("v1.ListOptions")).Return(&unstructured.
			UnstructuredList{}, fmt.Errorf("failed to reach gardener"))
		brcMock := &mocks.BrokerRuntimesClient{}
		brcMock.On("ListRuntimes", mock.AnythingOfType("runtime.ListParameters")).Return(run.RuntimesPage{}, nil)
		bcMock := &mocks.BrokerClient{}
		pMock := &mocks.ProvisionerClient{}

		memoryStorage := storage.NewMemoryStorage()
		logger := logrus.New()

		svc := NewService(gcMock, bcMock, brcMock, pMock, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

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
		gcMock.On("List", mock.Anything, mock.AnythingOfType("v1.ListOptions")).Return(fixShootList(), nil)
		brcMock := &mocks.BrokerRuntimesClient{}
		brcMock.On("ListRuntimes", mock.AnythingOfType("runtime.ListParameters")).Return(run.RuntimesPage{}, nil)
		bcMock := &mocks.BrokerClient{}
		pMock := &mocks.ProvisionerClient{}

		memoryStorage := storage.NewMemoryStorage()
		memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: "some-instance-id",
			RuntimeID:  "not-matching-id",
		})
		logger := logrus.New()

		svc := NewService(gcMock, bcMock, brcMock, pMock, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

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
		gcMock.On("List", mock.Anything, mock.AnythingOfType("v1.ListOptions")).Return(fixShootList(), nil)
		bcMock := &mocks.BrokerClient{}
		bcMock.On("Deprovision", mock.AnythingOfType("internal.Instance")).Return("",
			fmt.Errorf("failed to deprovision instance"))
		brcMock := &mocks.BrokerRuntimesClient{}
		brcMock.On("ListRuntimes", mock.AnythingOfType("runtime.ListParameters")).Return(run.RuntimesPage{}, nil)
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

		svc := NewService(gcMock, bcMock, brcMock, pMock, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

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
		gcMock.On("List", mock.Anything, mock.AnythingOfType("v1.ListOptions")).Return(fixShootList(), nil)
		bcMock := &mocks.BrokerClient{}
		brcMock := &mocks.BrokerRuntimesClient{}
		brcMock.On("ListRuntimes", mock.AnythingOfType("runtime.ListParameters")).Return(run.RuntimesPage{}, nil)
		pMock := &mocks.ProvisionerClient{}
		bcMock.On("Deprovision", mock.AnythingOfType("internal.Instance")).Return("", nil)
		pMock.On("DeprovisionRuntime", fixAccountID, fixRuntimeID2).Return("", fmt.Errorf("some error"))
		pMock.On("DeprovisionRuntime", fixAccountID, fixRuntimeID3).Return("", fmt.Errorf("some other error"))

		memoryStorage := storage.NewMemoryStorage()
		memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID1,
			RuntimeID:  fixRuntimeID1,
		})

		logger := logrus.New()

		svc := NewService(gcMock, bcMock, brcMock, pMock, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

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
		gcMock.On("List", mock.Anything, mock.AnythingOfType("v1.ListOptions")).Return(&unl, nil)
		bcMock := &mocks.BrokerClient{}
		pMock := &mocks.ProvisionerClient{}
		brcMock := &mocks.BrokerRuntimesClient{}
		brcMock.On("ListRuntimes", mock.AnythingOfType("runtime.ListParameters")).Return(run.RuntimesPage{}, nil)

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

		svc := NewService(gcMock, bcMock, brcMock, pMock, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

		// when
		err := svc.PerformCleanup()

		// then
		bcMock.AssertExpectations(t)
		gcMock.AssertExpectations(t)
		assert.Contains(t, actualLog.String(), shouldContain)
		assert.NoError(t, err)
	})

	t.Run("should remove shoots and runtimes by broker not when runtime found in broker", func(t *testing.T) {
		// given
		gcMock := &mocks.GardenerClient{}
		creationTime, parseErr := time.Parse(time.RFC3339, "2020-01-02T10:00:00-05:00")
		require.NoError(t, parseErr)

		var shootOne = unstructured.Unstructured{
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
		}

		var shootTwo = unstructured.Unstructured{
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":              "az-4567",
					"creationTimestamp": creationTime,
					"annotations": map[string]interface{}{
						shootAnnotationRuntimeId: fixRuntimeID2,
					},
					"clusterName": "cluster-one",
				},
				"spec": map[string]interface{}{
					"cloudProfileName": "az",
				},
			},
		}

		unl := unstructured.UnstructuredList{
			Items: []unstructured.Unstructured{
				shootOne,
				shootTwo,
			},
		}
		gcMock.On("List", mock.Anything, mock.AnythingOfType("v1.ListOptions")).Return(&unl, nil)
		gcMock.On("Get", mock.Anything, "az-4567", v1.GetOptions{}, "shoot").Return(&shootTwo, nil)
		gcMock.On("Get", mock.Anything, "az-1234", v1.GetOptions{}, "shoot").Return(&shootOne, nil)

		bcMock := &mocks.BrokerClient{}
		bcMock.On("Deprovision", mock.AnythingOfType("internal.Instance")).Return(fixOperationID, nil)

		pMock := &mocks.ProvisionerClient{}
		pMock.On("DeprovisionRuntime", fixAccountID, fixRuntimeID1).Return("", nil)
		pMock.On("DeprovisionRuntime", fixAccountID, fixRuntimeID2).Return("", nil)

		brcMock := &mocks.BrokerRuntimesClient{}
		brcMock.On("ListRuntimes", mock.AnythingOfType("runtime.ListParameters")).Return(run.RuntimesPage{
			Data: []run.RuntimeDTO{
				{
					ShootName: "az-1234",
					Status: run.RuntimeStatus{
						Provisioning: &run.Operation{
							CreatedAt: creationTime,
						},
					},
					RuntimeID:    fixRuntimeID1,
					SubAccountID: fixAccountID,
				},
				{
					ShootName: "az-4567",
					Status: run.RuntimeStatus{
						Provisioning: &run.Operation{
							CreatedAt: creationTime,
						},
					},
					RuntimeID:    fixRuntimeID2,
					SubAccountID: fixAccountID,
				},
			},
			Count:      2,
			TotalCount: 2,
		}, nil)

		memoryStorage := storage.NewMemoryStorage()
		memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID1,
			RuntimeID:  fixRuntimeID1,
		})
		memoryStorage.Instances().Insert(internal.Instance{
			InstanceID: fixInstanceID2,
			RuntimeID:  fixRuntimeID2,
		})

		var actualLog bytes.Buffer
		logger := logrus.New()
		logger.SetFormatter(&logrus.TextFormatter{
			DisableTimestamp: true,
		})
		logger.SetOutput(&actualLog)

		svc := NewService(gcMock, bcMock, brcMock, pMock, memoryStorage.Instances(), logger, maxShootAge, shootLabelSelector)

		// when
		err := svc.PerformCleanup()

		// then
		assert.NoError(t, err)

		// assert expectations on mocks

	})

	// TODO: Test pagination

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
