package provisioning

import (
	"fmt"
	"testing"
	"time"

	"github.com/kyma-project/control-plane/components/provisioner/internal/runtime/mocks"
	"github.com/stretchr/testify/mock"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"

	"github.com/kyma-incubator/compass/components/director/pkg/graphql"

	directorMocks "github.com/kyma-project/control-plane/components/provisioner/internal/director/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	v1alpha12 "github.com/kyma-project/kyma/components/compass-runtime-agent/pkg/apis/compass/v1alpha1"
	"github.com/kyma-project/kyma/components/compass-runtime-agent/pkg/client/clientset/versioned/fake"
	"github.com/kyma-project/kyma/components/compass-runtime-agent/pkg/client/clientset/versioned/typed/compass/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func TestWaitForAgentToConnect(t *testing.T) {

	cluster := model.Cluster{
		ID:         "someID",
		Tenant:     "someTenant",
		Kubeconfig: util.StringPtr(kubeconfig),
	}

	for _, testCase := range []struct {
		state v1alpha12.ConnectionState
	}{
		{
			state: v1alpha12.Synchronized,
		},
		{
			state: v1alpha12.SynchronizationFailed,
		},
		{
			state: v1alpha12.MetadataUpdateFailed,
		},
	} {
		t.Run(fmt.Sprintf("should proceed to next step when Compass connection in state: %s", testCase.state), func(t *testing.T) {
			// given
			clientProvider := newMockClientProvider(&v1alpha12.CompassConnection{
				ObjectMeta: v1.ObjectMeta{Name: defaultCompassConnectionName},
				Status: v1alpha12.CompassConnectionStatus{
					State: testCase.state,
				},
			})

			configurator := &mocks.Configurator{}

			directorClient := &directorMocks.DirectorClient{}
			directorClient.On("SetRuntimeStatusCondition", cluster.ID, graphql.RuntimeStatusConditionConnected, cluster.Tenant).Return(nil)

			waitForAgentToConnectStep := NewWaitForAgentToConnectStep(clientProvider.NewCompassConnectionClient, configurator, nextStageName, 10*time.Minute, directorClient)

			// when
			result, err := waitForAgentToConnectStep.Run(cluster, model.Operation{}, logrus.New())

			// then
			require.NoError(t, err)
			require.Equal(t, nextStageName, result.Stage)
			require.Equal(t, time.Duration(0), result.Delay)
		})

		t.Run(fmt.Sprintf("should retry when SetRuntimeStatusCondition fails and proceed to next step when Compass connection in state: %s", testCase.state), func(t *testing.T) {
			// given
			clientProvider := newMockClientProvider(&v1alpha12.CompassConnection{
				ObjectMeta: v1.ObjectMeta{Name: defaultCompassConnectionName},
				Status: v1alpha12.CompassConnectionStatus{
					State: testCase.state,
				},
			})

			configurator := &mocks.Configurator{}

			directorClient := &directorMocks.DirectorClient{}
			directorClient.On("SetRuntimeStatusCondition", cluster.ID, graphql.RuntimeStatusConditionConnected, cluster.Tenant).Once().Return(apperrors.Internal("runtime status error"))
			directorClient.On("SetRuntimeStatusCondition", cluster.ID, graphql.RuntimeStatusConditionConnected, cluster.Tenant).Once().Return(nil)

			waitForAgentToConnectStep := NewWaitForAgentToConnectStep(clientProvider.NewCompassConnectionClient, configurator, nextStageName, 10*time.Minute, directorClient)

			// when
			result, err := waitForAgentToConnectStep.Run(cluster, model.Operation{}, logrus.New())

			// then
			require.NoError(t, err)
			require.Equal(t, nextStageName, result.Stage)
			require.Equal(t, time.Duration(0), result.Delay)
		})

		t.Run(fmt.Sprintf("should rerun step if failed to update Director when Compass connection in state: %s", testCase.state), func(t *testing.T) {
			// given
			clientProvider := newMockClientProvider(&v1alpha12.CompassConnection{
				ObjectMeta: v1.ObjectMeta{Name: defaultCompassConnectionName},
				Status: v1alpha12.CompassConnectionStatus{
					State: testCase.state,
				},
			})

			configurator := &mocks.Configurator{}

			directorClient := &directorMocks.DirectorClient{}
			directorClient.On("SetRuntimeStatusCondition", cluster.ID, graphql.RuntimeStatusConditionConnected, cluster.Tenant).Return(apperrors.Internal("some error"))

			waitForAgentToConnectStep := NewWaitForAgentToConnectStep(clientProvider.NewCompassConnectionClient, configurator, nextStageName, 10*time.Minute, directorClient)

			// when
			result, err := waitForAgentToConnectStep.Run(cluster, model.Operation{}, logrus.New())

			// then
			require.NoError(t, err)
			require.Equal(t, model.WaitForAgentToConnect, result.Stage)
			require.Equal(t, 2*time.Second, result.Delay)
		})
	}

	t.Run("should proceed to next step when Agent connects", func(t *testing.T) {
		// given
		clientProvider := newMockClientProvider(&v1alpha12.CompassConnection{
			ObjectMeta: v1.ObjectMeta{Name: defaultCompassConnectionName},
			Status: v1alpha12.CompassConnectionStatus{
				State: v1alpha12.MetadataUpdateFailed,
			},
		})

		configurator := &mocks.Configurator{}

		directorClient := &directorMocks.DirectorClient{}
		directorClient.On("SetRuntimeStatusCondition", cluster.ID, graphql.RuntimeStatusConditionConnected, cluster.Tenant).Return(nil)

		waitForAgentToConnectStep := NewWaitForAgentToConnectStep(clientProvider.NewCompassConnectionClient, configurator, nextStageName, 10*time.Minute, directorClient)

		// when
		result, err := waitForAgentToConnectStep.Run(cluster, model.Operation{}, logrus.New())

		// then
		require.NoError(t, err)
		require.Equal(t, nextStageName, result.Stage)
		require.Equal(t, time.Duration(0), result.Delay)
	})

	t.Run("should rerun step if connection not yet synchronize", func(t *testing.T) {
		// given
		clientProvider := newMockClientProvider(&v1alpha12.CompassConnection{
			ObjectMeta: v1.ObjectMeta{Name: defaultCompassConnectionName},
			Status: v1alpha12.CompassConnectionStatus{
				State: v1alpha12.Connected,
			},
		})

		configurator := &mocks.Configurator{}

		directorClient := &directorMocks.DirectorClient{}
		directorClient.On("SetRuntimeStatusCondition", cluster.ID, graphql.RuntimeStatusConditionConnected, cluster.Tenant).Return(nil)

		waitForAgentToConnectStep := NewWaitForAgentToConnectStep(clientProvider.NewCompassConnectionClient, configurator, nextStageName, 10*time.Minute, directorClient)

		// when
		result, err := waitForAgentToConnectStep.Run(cluster, model.Operation{}, logrus.New())

		// then
		require.NoError(t, err)
		require.Equal(t, model.WaitForAgentToConnect, result.Stage)
		require.Equal(t, 2*time.Second, result.Delay)
	})

	t.Run("should rerun step if Compass connection not found", func(t *testing.T) {
		// given
		clientProvider := newMockClientProvider(&v1alpha12.CompassConnection{})
		configurator := &mocks.Configurator{}
		directorClient := &directorMocks.DirectorClient{}
		directorClient.On("SetRuntimeStatusCondition", cluster.ID, graphql.RuntimeStatusConditionConnected, cluster.Tenant).Return(nil)

		waitForAgentToConnectStep := NewWaitForAgentToConnectStep(clientProvider.NewCompassConnectionClient, configurator, nextStageName, 10*time.Minute, directorClient)

		// when
		result, err := waitForAgentToConnectStep.Run(cluster, model.Operation{}, logrus.New())

		// then
		require.NoError(t, err)
		require.Equal(t, model.WaitForAgentToConnect, result.Stage)
		require.Equal(t, 5*time.Second, result.Delay)
	})

	t.Run("should return error if Compass Connection in Connection Failed state and runtime reconfigure fails", func(t *testing.T) {
		// given
		clientProvider := newMockClientProvider(&v1alpha12.CompassConnection{
			ObjectMeta: v1.ObjectMeta{Name: defaultCompassConnectionName},
			Status: v1alpha12.CompassConnectionStatus{
				State: v1alpha12.ConnectionFailed,
			},
		})
		configurator := &mocks.Configurator{}
		configurator.On("ConfigureRuntime", cluster, mock.AnythingOfType("string")).Return(apperrors.Internal("test: runtime reconfigure failure"))

		directorClient := &directorMocks.DirectorClient{}
		directorClient.On("SetRuntimeStatusCondition", cluster.ID, graphql.RuntimeStatusConditionConnected, cluster.Tenant).Return(nil)

		waitForAgentToConnectStep := NewWaitForAgentToConnectStep(clientProvider.NewCompassConnectionClient, configurator, nextStageName, 10*time.Minute, directorClient)

		// when
		_, err := waitForAgentToConnectStep.Run(cluster, model.Operation{}, logrus.New())

		// then
		require.Error(t, err)
	})

	t.Run("should should rerun step if Compass Connection in Connection Failed state and runtime reconfigure is successful", func(t *testing.T) {
		// given
		clientProvider := newMockClientProvider(&v1alpha12.CompassConnection{
			ObjectMeta: v1.ObjectMeta{Name: defaultCompassConnectionName},
			Status: v1alpha12.CompassConnectionStatus{
				State: v1alpha12.ConnectionFailed,
			},
		})
		configurator := &mocks.Configurator{}
		configurator.On("ConfigureRuntime", cluster, mock.AnythingOfType("string")).Return(nil)

		directorClient := &directorMocks.DirectorClient{}
		directorClient.On("SetRuntimeStatusCondition", cluster.ID, graphql.RuntimeStatusConditionConnected, cluster.Tenant).Return(nil)

		waitForAgentToConnectStep := NewWaitForAgentToConnectStep(clientProvider.NewCompassConnectionClient, configurator, nextStageName, 10*time.Minute, directorClient)

		// when
		result, err := waitForAgentToConnectStep.Run(cluster, model.Operation{}, logrus.New())

		// then
		require.NoError(t, err)
		require.Equal(t, model.WaitForAgentToConnect, result.Stage)
		require.Equal(t, 2*time.Minute, result.Delay)
	})
}

type mockClientProvider struct {
	compassConnection *v1alpha12.CompassConnection
}

func newMockClientProvider(compassConnection *v1alpha12.CompassConnection) *mockClientProvider {
	return &mockClientProvider{
		compassConnection: compassConnection,
	}
}

func (m *mockClientProvider) NewCompassConnectionClient(k8sConfig *rest.Config) (v1alpha1.CompassConnectionInterface, error) {
	fakeClient := fake.NewSimpleClientset(m.compassConnection)

	return fakeClient.CompassV1alpha1().CompassConnections(), nil
}
