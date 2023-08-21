package provisioning

import (
	"testing"
	"time"

	"github.com/pkg/errors"

	provisioning_mocks "github.com/kyma-project/control-plane/components/provisioner/internal/operations/stages/provisioning/mocks"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"

	"github.com/kyma-project/control-plane/components/provisioner/internal/model"
	"github.com/kyma-project/control-plane/components/provisioner/internal/runtime/mocks"
	"github.com/kyma-project/control-plane/components/provisioner/internal/util"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	nextStageName model.OperationStage = "NextStage"
)

func TestConnectAgentStep_Run(t *testing.T) {

	cluster := model.Cluster{
		Kubeconfig: util.StringPtr("kubeconfig"),
		ClusterConfig: model.GardenerConfig{
			Name: "shoot",
		},
	}
	dynamicKubeconfigProvider := &provisioning_mocks.DynamicKubeconfigProvider{}
	dynamicKubeconfigProvider.On("FetchFromGardener", "shoot").Return([]byte("dynamic_kubeconfig"), nil)

	t.Run("should return next step when finished", func(t *testing.T) {
		// given
		configurator := &mocks.Configurator{}
		configurator.On("ConfigureRuntime", cluster, dynamicKubeconfig).Return(nil)

		stage := NewConnectAgentStep(configurator, dynamicKubeconfigProvider, nextStageName, time.Minute)

		// when
		result, err := stage.Run(cluster, model.Operation{}, &logrus.Entry{})

		// then
		require.NoError(t, err)
		assert.Equal(t, nextStageName, result.Stage)
		assert.Equal(t, time.Duration(0), result.Delay)
	})

	t.Run("should return error when failed to get dynamic kubeconfig", func(t *testing.T) {
		// given
		dynamicKubeconfigProvider := &provisioning_mocks.DynamicKubeconfigProvider{}
		dynamicKubeconfigProvider.On("FetchFromGardener", "shoot").Return(nil, errors.New("some error"))

		configurator := &mocks.Configurator{}

		stage := NewConnectAgentStep(configurator, dynamicKubeconfigProvider, nextStageName, time.Minute)

		// when
		_, err := stage.Run(cluster, model.Operation{}, &logrus.Entry{})

		// then
		require.Error(t, err)
	})

	t.Run("should return error when failed to configure cluster", func(t *testing.T) {
		// given
		configurator := &mocks.Configurator{}
		configurator.On("ConfigureRuntime", cluster, dynamicKubeconfig).Return(apperrors.Internal("error"))

		stage := NewConnectAgentStep(configurator, dynamicKubeconfigProvider, nextStageName, time.Minute)

		// when
		_, err := stage.Run(cluster, model.Operation{}, &logrus.Entry{})

		// then
		require.Error(t, err)
	})

}
