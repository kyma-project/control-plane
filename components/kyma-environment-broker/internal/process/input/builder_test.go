package input

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input/automock"
	cloudProvider "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Currently on production only azure is supported

func TestInputBuilderFactory_IsPlanSupport(t *testing.T) {
	// given
	componentsProvider := &automock.ComponentListProvider{}
	defer componentsProvider.AssertExpectations(t)

	configProvider := mockConfigProvider()

	ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(), componentsProvider,
		configProvider, Config{}, "1.10", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
	assert.NoError(t, err)

	// when/then
	assert.True(t, ibf.IsPlanSupport(broker.GCPPlanID))
	assert.True(t, ibf.IsPlanSupport(broker.AzurePlanID))
	assert.True(t, ibf.IsPlanSupport(broker.AzureHAPlanID))
	assert.True(t, ibf.IsPlanSupport(broker.TrialPlanID))
}

func TestInputBuilderFactory_ForPlan(t *testing.T) {
	t.Run("should build RuntimeInput with default version Kyma components and ProvisionRuntimeInput", func(t *testing.T) {
		// given
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"),
			mock.AnythingOfType("*internal.ConfigForPlan")).Return([]internal.KymaComponent{}, nil).Once()
		defer componentsProvider.AssertExpectations(t)

		configProvider := mockConfigProvider()

		ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(), componentsProvider,
			configProvider, Config{}, "1.10", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID, "")

		// when
		input, err := ibf.CreateProvisionInput(pp, internal.RuntimeVersionData{
			Version: "1.10",
			Origin:  internal.Defaults,
		})

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
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("*internal.ConfigForPlan")).Return([]internal.
			KymaComponent{}, nil)
		defer componentsProvider.AssertExpectations(t)

		configProvider := mockConfigProvider()

		ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(), componentsProvider,
			configProvider, Config{}, "1.10", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID, "")

		// when
		input, err := ibf.CreateUpgradeInput(pp, internal.RuntimeVersionData{Version: "1.1.0", Origin: internal.Defaults})

		// Then
		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, input)

		result := input.(*RuntimeInput)
		assert.NotNil(t, result.upgradeRuntimeInput)
	})

	t.Run("should build RuntimeInput with GA version Kyma components and UpgradeRuntimeInput", func(t *testing.T) {
		// given
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("*internal.ConfigForPlan")).Return([]internal.
			KymaComponent{}, nil)
		defer componentsProvider.AssertExpectations(t)

		configProvider := mockConfigProvider()

		ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(), componentsProvider,
			configProvider, Config{}, "1.10", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID, "")

		// when
		input, err := ibf.CreateUpgradeInput(pp, internal.RuntimeVersionData{Version: "1.1.0", Origin: internal.AccountMapping})

		// Then
		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, input)

		result := input.(*RuntimeInput)
		assert.NotNil(t, result.upgradeRuntimeInput)
	})

	t.Run("should build RuntimeInput with set version Kyma components", func(t *testing.T) {
		// given
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("*internal.ConfigForPlan")).Return([]internal.KymaComponent{}, nil).Once()
		defer componentsProvider.AssertExpectations(t)

		configProvider := mockConfigProvider()

		ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(), componentsProvider,
			configProvider, Config{}, "1.10", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID, "PR-1")

		// when
		input, err := ibf.CreateProvisionInput(pp, internal.RuntimeVersionData{Version: "PR-1", Origin: internal.Parameters})

		// Then
		assert.NoError(t, err)
		assert.IsType(t, &RuntimeInput{}, input)
	})

	t.Run("should build RuntimeInput with proper plan", func(t *testing.T) {
		// given
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("*internal.ConfigForPlan")).Return([]internal.KymaComponent{}, nil).Once()
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("*internal.ConfigForPlan")).Return([]internal.KymaComponent{}, nil).Once()
		defer componentsProvider.AssertExpectations(t)

		configProvider := mockConfigProvider()

		ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(), componentsProvider,
			configProvider, Config{}, "1.10", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID, "")

		// when
		input, err := ibf.CreateProvisionInput(pp, internal.RuntimeVersionData{Version: "1.1.0", Origin: internal.Defaults})

		// Then
		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, input)

		result := input.(*RuntimeInput)
		assert.Equal(t, gqlschema.KymaProfileProduction, *result.provisionRuntimeInput.KymaConfig.Profile)

		// given
		pp = fixProvisioningParameters(broker.TrialPlanID, "")

		// when
		input, err = ibf.CreateProvisionInput(pp, internal.RuntimeVersionData{Version: "1.1.0", Origin: internal.Defaults})

		// Then
		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, input)

		result = input.(*RuntimeInput)
		assert.Equal(t, gqlschema.KymaProfileEvaluation, *result.provisionRuntimeInput.KymaConfig.Profile)

	})

	t.Run("should build UpgradeRuntimeInput with proper profile", func(t *testing.T) {
		// given
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("*internal.ConfigForPlan")).Return([]internal.KymaComponent{}, nil)
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("*internal.ConfigForPlan")).Return([]internal.KymaComponent{}, nil)
		defer componentsProvider.AssertExpectations(t)

		configProvider := mockConfigProvider()

		ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(), componentsProvider,
			configProvider, Config{}, "1.10", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID, "")

		// when
		input, err := ibf.CreateUpgradeInput(pp, internal.RuntimeVersionData{Version: "1.1.0", Origin: internal.Defaults})

		// Then
		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, input)

		result := input.(*RuntimeInput)
		assert.NotNil(t, result.upgradeRuntimeInput)
		assert.NotNil(t, result.upgradeRuntimeInput.KymaConfig.Profile)
		assert.Equal(t, gqlschema.KymaProfileProduction, *result.upgradeRuntimeInput.KymaConfig.Profile)

		// given
		pp = fixProvisioningParameters(broker.TrialPlanID, "")
		provider := internal.GCP
		pp.Parameters.Provider = &provider
		// when
		input, err = ibf.CreateUpgradeInput(pp, internal.RuntimeVersionData{Version: "1.1.0", Origin: internal.Defaults})

		// Then
		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, input)

		result = input.(*RuntimeInput)
		assert.NotNil(t, result.upgradeRuntimeInput)
		assert.NotNil(t, result.upgradeRuntimeInput.KymaConfig.Profile)
		assert.Equal(t, gqlschema.KymaProfileEvaluation, *result.upgradeRuntimeInput.KymaConfig.Profile)
	})

	t.Run("should build CreateUpgradeShootInput with proper autoscaler parameters", func(t *testing.T) {
		// given
		var provider HyperscalerInputProvider

		componentsProvider := &automock.ComponentListProvider{}
		defer componentsProvider.AssertExpectations(t)

		configProvider := mockConfigProvider()

		ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(), componentsProvider,
			configProvider, Config{}, "1.10", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID, "")
		provider = &cloudProvider.GcpInput{} // for broker.GCPPlanID
		ver := internal.RuntimeVersionData{
			Version: "2.4.0",
			Origin:  internal.Defaults,
		}

		// when
		input, err := ibf.CreateUpgradeShootInput(pp, ver)

		// Then
		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, input)

		result := input.(*RuntimeInput)
		autoscalerMax := *result.upgradeShootInput.GardenerConfig.AutoScalerMax
		autoscalerMin := *result.upgradeShootInput.GardenerConfig.AutoScalerMin
		maxSurge := *result.upgradeShootInput.GardenerConfig.MaxSurge
		maxUnavailable := *result.upgradeShootInput.GardenerConfig.MaxUnavailable

		assert.Equal(t, autoscalerMax, provider.Defaults().GardenerConfig.AutoScalerMax)
		assert.Equal(t, autoscalerMin, provider.Defaults().GardenerConfig.AutoScalerMin)
		assert.Equal(t, maxSurge, provider.Defaults().GardenerConfig.MaxSurge)
		assert.Equal(t, maxUnavailable, provider.Defaults().GardenerConfig.MaxUnavailable)
		t.Logf("%v, %v, %v, %v", autoscalerMax, autoscalerMin, maxSurge, maxUnavailable)
	})

}

func fixProvisioningParameters(planID, kymaVersion string) internal.ProvisioningParameters {
	pp := fixture.FixProvisioningParameters("")
	pp.PlanID = planID
	pp.Parameters.KymaVersion = kymaVersion
	pp.Parameters.AutoScalerMin = ptr.Integer(1)
	pp.Parameters.AutoScalerMax = ptr.Integer(1)

	return pp
}

func fixTrialRegionMapping() map[string]string {
	return map[string]string{}
}
