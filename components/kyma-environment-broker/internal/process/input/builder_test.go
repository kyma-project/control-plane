package input

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input/automock"

	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
	"github.com/stretchr/testify/assert"
)

// Currently on production only azure is supported

func TestInputBuilderFactory_IsPlanSupport(t *testing.T) {
	// given
	componentsProvider := &automock.ComponentListProvider{}
	componentsProvider.On("AllComponents", "1.10").Return([]v1alpha1.KymaComponent{}, nil)
	defer componentsProvider.AssertExpectations(t)

	ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(), componentsProvider,
		Config{}, "1.10", fixTrialRegionMapping())
	assert.NoError(t, err)

	// when/then
	assert.True(t, ibf.IsPlanSupport(broker.GCPPlanID))
	assert.True(t, ibf.IsPlanSupport(broker.AzurePlanID))
	assert.True(t, ibf.IsPlanSupport(broker.TrialPlanID))
}

func TestInputBuilderFactory_ForPlan(t *testing.T) {
	t.Run("should build RuntimeInput with default version Kyma components and ProvisionRuntimeInput", func(t *testing.T) {
		// given
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", "1.10").Return([]v1alpha1.KymaComponent{}, nil).Once()
		defer componentsProvider.AssertExpectations(t)

		ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.10", fixTrialRegionMapping())
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID, "")

		// when
		input, err := ibf.NewProvisionRuntimeInputCreator(pp)

		// Then
		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, input)

		result := input.(*RuntimeInput)
		assert.NotNil(t, result.provisionRuntimeInput)
		assert.Nil(t, result.upgradeRuntimeInput.KymaConfig)

	})

	t.Run("should build RuntimeInput with default version Kyma components and UpgradeRuntimeInput", func(t *testing.T) {
		// given
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", "1.10").Return([]v1alpha1.KymaComponent{}, nil).Once()
		defer componentsProvider.AssertExpectations(t)

		ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.10", fixTrialRegionMapping())
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID, "")

		// when
		input, err := ibf.NewUpgradeRuntimeInputCreator(pp)

		// Then
		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, input)

		result := input.(*RuntimeInput)
		assert.NotNil(t, result.upgradeRuntimeInput)
		assert.Nil(t, result.provisionRuntimeInput.KymaConfig)
		assert.Nil(t, result.provisionRuntimeInput.RuntimeInput)
		assert.Nil(t, result.provisionRuntimeInput.ClusterConfig)

	})

	t.Run("should build RuntimeInput with set version Kyma components", func(t *testing.T) {
		// given
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", "1.10").Return([]v1alpha1.KymaComponent{}, nil).Once()
		componentsProvider.On("AllComponents", "PR-1").Return([]v1alpha1.KymaComponent{}, nil).Once()
		defer componentsProvider.AssertExpectations(t)

		ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(), componentsProvider, Config{}, "1.10")
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID, "PR-1")

		// when
		input, err := ibf.NewProvisionRuntimeInputCreator(pp)

		// Then
		assert.NoError(t, err)
		assert.IsType(t, &RuntimeInput{}, input)
	})
}

func fixProvisioningParameters(planID, kymaVersion string) internal.ProvisioningParameters {
	return internal.ProvisioningParameters{
		PlanID: planID,
		Parameters: internal.ProvisioningParametersDTO{
			KymaVersion: kymaVersion,
		},
	}
}

func fixTrialRegionMapping() map[string]string {
	return map[string]string{}
}
