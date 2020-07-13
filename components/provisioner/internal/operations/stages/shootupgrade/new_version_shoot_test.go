package shootupgrade

import (
	"errors"
	"testing"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_types "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/operations"
	gardener_mocks "github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/deprovisioning/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	oldResourceVersion = "oldVersion"
	newResourceVersion = "newVersion"
)

func TestWaitForNewShootClusterVersion_SingleShoot(t *testing.T) {
	clusterName := "shootName"
	runtimeID := "runtimeID"
	tenant := "tenant"
	operationID := "operationID"

	cluster := model.Cluster{
		ID:     runtimeID,
		Tenant: tenant,
		ClusterConfig: model.GardenerConfig{
			Name: clusterName,
		},
	}

	for _, testCase := range []struct {
		description            string
		mockFunc               func(gardenerClient *gardener_mocks.GardenerClient)
		expectedStage          model.OperationStage
		expectedDelay          time.Duration
		initialResourceVersion string
	}{
		{
			description: "should continue waiting if cluster has old resource version",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Get", clusterName, mock.Anything).Return(fixShootOldResourceVersion(clusterName), nil)
			},
			expectedStage:          model.WaitingForShootNewVersion,
			expectedDelay:          5 * time.Second,
			initialResourceVersion: oldResourceVersion,
		},
		{
			description: "should return finished stage if cluster upgrade has succeeded",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Get", clusterName, mock.Anything).Return(fixShootNewResourceVersion(clusterName), nil)
			},
			expectedStage:          model.WaitingForShootUpgrade,
			expectedDelay:          0,
			initialResourceVersion: oldResourceVersion,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardener_mocks.GardenerClient{}

			testCase.mockFunc(gardenerClient)

			waitForShootClusterUpgradeStep := NewWaitForShootNewVersionStep(gardenerClient, model.WaitingForShootUpgrade, time.Minute)
			waitForShootClusterUpgradeStep.addInitialResourceVersionValue(operationID, testCase.initialResourceVersion)
			// when
			result, err := waitForShootClusterUpgradeStep.Run(cluster, model.Operation{ID: operationID}, logrus.New())

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedStage, result.Stage)
			assert.Equal(t, testCase.expectedDelay, result.Delay)
			gardenerClient.AssertExpectations(t)
		})
	}

	for _, testCase := range []struct {
		description        string
		mockFunc           func(gardenerClient *gardener_mocks.GardenerClient)
		unrecoverableError bool
		cluster            model.Cluster
	}{
		{
			description: "should return error if failed to read Shoot",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Get", clusterName, mock.Anything).Return(nil, errors.New("some error"))
			},
			unrecoverableError: false,
			cluster:            cluster,
		},
		{
			description: "should return unrecoverable error if Shoot is in failed state",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Get", clusterName, mock.Anything).Return(fixShootInFailedState(clusterName), nil)
			},
			unrecoverableError: true,
			cluster:            cluster,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardener_mocks.GardenerClient{}

			testCase.mockFunc(gardenerClient)

			waitForClusterCreationStep := NewWaitForShootNewVersionStep(gardenerClient, model.FinishedStage, time.Minute)

			// when
			_, err := waitForClusterCreationStep.Run(testCase.cluster, model.Operation{}, logrus.New())

			// then
			require.Error(t, err)
			nonRecoverable := operations.NonRecoverableError{}
			require.Equal(t, testCase.unrecoverableError, errors.As(err, &nonRecoverable))
			gardenerClient.AssertExpectations(t)
		})
	}
}

func TestWaitForNewShootClusterVersion_ParallelExecution(t *testing.T) {
	cluster1 := model.Cluster{ClusterConfig: model.GardenerConfig{Name: "shoot1"}}
	cluster2 := model.Cluster{ClusterConfig: model.GardenerConfig{Name: "shoot2"}}

	for _, testCase := range []struct {
		description                 string
		mockFunc                    func(gardenerClient *gardener_mocks.GardenerClient)
		cluster1                    model.Cluster
		cluster2                    model.Cluster
		operationsInProgress        initialResourceVersions
		expectedResourceVersionsMap initialResourceVersions
	}{
		{
			description: "should add new resource version to map",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Get", cluster2.ClusterConfig.Name, mock.Anything).Return(fixShootOldResourceVersion("shoot2"), nil)
			},
			cluster1: cluster1,
			cluster2: cluster2,
			operationsInProgress: initialResourceVersions{versions: map[string]string{
				"operation1": "oldVersion",
			}},
			expectedResourceVersionsMap: initialResourceVersions{versions: map[string]string{
				"operation1": "oldVersion",
				"operation2": "oldVersion",
			}},
		},
		{
			description: "should remove resource version when step finished",
			mockFunc: func(gardenerClient *gardener_mocks.GardenerClient) {
				gardenerClient.On("Get", cluster2.ClusterConfig.Name, mock.Anything).Return(fixShootNewResourceVersion("shoot2"), nil)
			},
			cluster1: cluster1,
			cluster2: cluster2,
			operationsInProgress: initialResourceVersions{versions: map[string]string{
				"operation1": "oldVersion",
				"operation2": "oldVersion",
			}},
			expectedResourceVersionsMap: initialResourceVersions{versions: map[string]string{
				"operation1": "oldVersion",
			}},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			// given
			gardenerClient := &gardener_mocks.GardenerClient{}

			testCase.mockFunc(gardenerClient)

			waitForClusterCreationStep := NewWaitForShootNewVersionStep(gardenerClient, model.FinishedStage, time.Minute)
			waitForClusterCreationStep.initialResourceVersions = testCase.operationsInProgress

			// when
			_, err := waitForClusterCreationStep.Run(testCase.cluster2, model.Operation{ID: "operation2"}, logrus.New())

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedResourceVersionsMap, waitForClusterCreationStep.initialResourceVersions)
			gardenerClient.AssertExpectations(t)
		})
	}
}

func fixShootOldResourceVersion(name string) *gardener_types.Shoot {
	return fixShootWithResourceVersion(name, oldResourceVersion, &gardener_types.LastOperation{
		State: gardencorev1beta1.LastOperationStateSucceeded,
	})
}

func fixShootNewResourceVersion(name string) *gardener_types.Shoot {
	return fixShootWithResourceVersion(name, newResourceVersion, &gardener_types.LastOperation{
		State: gardencorev1beta1.LastOperationStateSucceeded,
	})
}

func fixShootWithResourceVersion(name string, version string, lastOperation *gardener_types.LastOperation) *gardener_types.Shoot {
	return &gardener_types.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			ResourceVersion: version,
		},
		Status: gardener_types.ShootStatus{
			LastOperation: lastOperation,
		},
	}
}
