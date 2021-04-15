package input

import (
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Currently on production only azure is supported

func TestInputBuilderFactory_IsPlanSupport(t *testing.T) {
	// given
	componentsProvider := &automock.ComponentListProvider{}
	componentsProvider.On("AllComponents", "1.10").Return(runtime.ComponentListData{}, nil)
	defer componentsProvider.AssertExpectations(t)

	ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(), componentsProvider,
		Config{}, "1.10", fixTrialRegionMapping(), fixTrialProviders(), &fixture.FakeListDecider{})
	assert.NoError(t, err)

	// when/then
	assert.True(t, ibf.IsPlanSupport(broker.GCPPlanID))
	assert.True(t, ibf.IsPlanSupport(broker.AzurePlanID))
	assert.True(t, ibf.IsPlanSupport(broker.AzureHAPlanID))
	assert.True(t, ibf.IsPlanSupport(broker.AzureLitePlanID))
	assert.True(t, ibf.IsPlanSupport(broker.TrialPlanID))
	assert.True(t, ibf.IsPlanSupport(broker.AWSPlanID))
	assert.True(t, ibf.IsPlanSupport(broker.OpenStackPlanID))
}

func TestInputBuilderFactory_ForPlan(t *testing.T) {
	t.Run("should build RuntimeInput with default version Kyma components and ProvisionRuntimeInput", func(t *testing.T) {
		// given
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", "1.10").Return(runtime.ComponentListData{
			Prerequisites: []runtime.ComponentDefinition{
				{
					Name:      "istio",
					Namespace: "istio",
				},
			},
			Components: []runtime.ComponentDefinition{
				{
					Name:      "kyma-component",
					Namespace: "kyma-system",
				},
				{
					Name:      "kyma-component-with-url",
					Namespace: "kyma-system",
					Source:    &runtime.ComponentSource{URL: "http://kyma.test.com"},
				},
			},
		}, nil).Once()
		defer componentsProvider.AssertExpectations(t)

		ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.10", fixTrialRegionMapping(), fixTrialProviders(), &fixture.FakeListDecider{})
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
		assert.Len(t, result.provisionRuntimeInput.KymaConfig.Components, 3)
		assert.ElementsMatch(t, result.provisionRuntimeInput.KymaConfig.Components, []*gqlschema.ComponentConfigurationInput{
			{
				Component:    "kyma-component-with-url",
				Namespace:    "kyma-system",
				SourceURL:    ptr.String("http://kyma.test.com"),
				Prerequisite: ptr.Bool(false),
			},
			{
				Component:    "kyma-component",
				Namespace:    "kyma-system",
				Prerequisite: ptr.Bool(false),
			},
			{
				Component:    "istio",
				Namespace:    "istio",
				Prerequisite: ptr.Bool(true),
			},
		})
		assert.Equal(t, gqlschema.KymaInstallationMethodKymaOperator, *result.provisionRuntimeInput.KymaConfig.KymaInstaller)
	})

	t.Run("should build RuntimeInput with default version Kyma components and UpgradeRuntimeInput", func(t *testing.T) {
		// given
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", "1.10").Return(runtime.ComponentListData{}, nil).Once()
		defer componentsProvider.AssertExpectations(t)

		ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.10", fixTrialRegionMapping(), fixTrialProviders(), &fixture.FakeListDecider{})
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID, "")

		// when
		input, err := ibf.CreateUpgradeInput(pp, internal.RuntimeVersionData{Version: "1.1.0", Origin: internal.Defaults})

		// Then
		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, input)

		result := input.(*RuntimeInput)
		assert.NotNil(t, result.upgradeRuntimeInput)
		assert.Nil(t, result.provisionRuntimeInput.KymaConfig)
		assert.Nil(t, result.provisionRuntimeInput.RuntimeInput)
		assert.Nil(t, result.provisionRuntimeInput.ClusterConfig)
	})

	t.Run("should build RuntimeInput with GA version Kyma components and UpgradeRuntimeInput", func(t *testing.T) {
		// given
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", "1.10").Return(runtime.ComponentListData{}, nil).Once()
		componentsProvider.On("AllComponents", "1.1.0").Return(runtime.ComponentListData{}, nil).Once()
		defer componentsProvider.AssertExpectations(t)

		ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.10", fixTrialRegionMapping(), fixTrialProviders(), &fixture.FakeListDecider{})
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID, "")

		// when
		input, err := ibf.CreateUpgradeInput(pp, internal.RuntimeVersionData{Version: "1.1.0", Origin: internal.AccountMapping})

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
		componentsProvider.On("AllComponents", "1.10").Return(runtime.ComponentListData{}, nil).Once()
		componentsProvider.On("AllComponents", "PR-1").Return(runtime.ComponentListData{}, nil).Once()
		defer componentsProvider.AssertExpectations(t)

		ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(),
			componentsProvider, Config{}, "1.10", fixTrialRegionMapping(), fixTrialProviders(), &fixture.FakeListDecider{})
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
		componentsProvider.On("AllComponents", "1.10").Return(runtime.ComponentListData{}, nil).Once()
		defer componentsProvider.AssertExpectations(t)

		ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.10", fixTrialRegionMapping(), fixTrialProviders(), &fixture.FakeListDecider{})
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

	t.Run("should build UpgradeRuntimeInput with proper profile and new installation type", func(t *testing.T) {
		// given
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", "1.10").Return(runtime.ComponentListData{}, nil).Once()
		defer componentsProvider.AssertExpectations(t)

		ibf, err := NewInputBuilderFactory(nil, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.10", fixTrialRegionMapping(), fixTrialProviders(), &fixture.FakeListDecider{NewComponentList: true})
		assert.NoError(t, err)
		pp := fixProvisioningParameters(broker.GCPPlanID, "")

		// when
		input, err := ibf.CreateUpgradeInput(pp, internal.RuntimeVersionData{Version: "1.1.0", Origin: internal.Defaults})

		// Then
		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, input)

		result := input.(*RuntimeInput)
		assert.NotNil(t, result.upgradeRuntimeInput)
		assert.Nil(t, result.provisionRuntimeInput.KymaConfig)
		assert.Nil(t, result.provisionRuntimeInput.RuntimeInput)
		assert.Nil(t, result.provisionRuntimeInput.ClusterConfig)
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
		assert.Nil(t, result.provisionRuntimeInput.KymaConfig)
		assert.Nil(t, result.provisionRuntimeInput.RuntimeInput)
		assert.Nil(t, result.provisionRuntimeInput.ClusterConfig)
		assert.NotNil(t, result.upgradeRuntimeInput.KymaConfig.Profile)
		assert.Equal(t, gqlschema.KymaProfileEvaluation, *result.upgradeRuntimeInput.KymaConfig.Profile)
		assert.Equal(t, gqlschema.KymaInstallationMethodParallelInstall, *result.upgradeRuntimeInput.KymaConfig.KymaInstaller)
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
