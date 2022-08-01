package input

import (
	"testing"

	"github.com/google/uuid"
	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input/automock"
	cloudProvider "github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var emptyVersion = internal.RuntimeVersionData{}

func TestShouldEnableComponents(t *testing.T) {
	t.Run("When creating ProvisionRuntimeInput", func(t *testing.T) {
		// given

		// One base component: dex
		// Two optional components: Kiali and Tracing
		// The test checks, if EnableOptionalComponent method adds an optional component
		optionalComponentsDisablers := runtime.ComponentsDisablers{
			components.Kiali:   runtime.NewGenericComponentDisabler(components.Kiali),
			components.Tracing: runtime.NewGenericComponentDisabler(components.Tracing),
		}
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).
			Return([]internal.KymaComponent{
				{Name: components.Kiali},
				{Name: components.Tracing},
				{Name: "dex"},
			}, nil)

		builder, err := NewInputBuilderFactory(runtime.NewOptionalComponentsService(optionalComponentsDisablers), runtime.NewDisabledComponentsProvider(),
			componentsProvider, Config{}, "not-important", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)

		pp := fixProvisioningParameters(broker.AzurePlanID, "")
		creator, err := builder.CreateProvisionInput(pp, internal.RuntimeVersionData{Version: "1.1.0", Origin: internal.Defaults})
		require.NoError(t, err)

		// when
		creator.EnableOptionalComponent(components.Kiali)
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)

		// then
		assertComponentExists(t, input.KymaConfig.Components, gqlschema.ComponentConfigurationInput{
			Component: components.Kiali,
		})
		assertComponentExists(t, input.KymaConfig.Components, gqlschema.ComponentConfigurationInput{
			Component: "dex",
		})
		assert.Len(t, input.KymaConfig.Components, 2)
	})

	t.Run("When creating UpgradeRuntimeInput", func(t *testing.T) {
		// given

		// One base component: dex
		// Two optional components: Kiali and Tracing
		// The test checks, if EnableOptionalComponent method adds an optional component
		optionalComponentsDisablers := runtime.ComponentsDisablers{
			components.Kiali:   runtime.NewGenericComponentDisabler(components.Kiali),
			components.Tracing: runtime.NewGenericComponentDisabler(components.Tracing),
		}
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).
			Return([]internal.KymaComponent{
				{Name: components.Kiali},
				{Name: components.Tracing},
				{Name: "dex"},
			}, nil)

		builder, err := NewInputBuilderFactory(runtime.NewOptionalComponentsService(optionalComponentsDisablers), runtime.NewDisabledComponentsProvider(),
			componentsProvider, Config{}, "not-important", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)

		pp := fixProvisioningParameters(broker.AzurePlanID, "1.14.0")
		creator, err := builder.CreateUpgradeInput(pp, internal.RuntimeVersionData{Version: "1.14.0", Origin: internal.Defaults})
		require.NoError(t, err)

		// when
		creator.EnableOptionalComponent(components.Kiali)
		input, err := creator.CreateUpgradeRuntimeInput()
		require.NoError(t, err)

		// then
		assertComponentExists(t, input.KymaConfig.Components, gqlschema.ComponentConfigurationInput{
			Component: components.Kiali,
		})
		assertComponentExists(t, input.KymaConfig.Components, gqlschema.ComponentConfigurationInput{
			Component: "dex",
		})
		assert.Len(t, input.KymaConfig.Components, 2)
	})
}

func fixTrialProviders() []string {
	return []string{"azure", "aws"}
}

func TestShouldDisableComponents(t *testing.T) {
	t.Run("When creating ProvisionRuntimeInput", func(t *testing.T) {
		// given
		pp := fixProvisioningParameters(broker.AzurePlanID, "")

		optionalComponentsDisablers := runtime.ComponentsDisablers{}
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).
			Return([]internal.KymaComponent{
				{Name: components.Kiali},
				{Name: components.Tracing},
				{Name: components.Backup},
			}, nil)

		builder, err := NewInputBuilderFactory(runtime.NewOptionalComponentsService(optionalComponentsDisablers), runtime.NewDisabledComponentsProvider(),
			componentsProvider, Config{}, "not-important", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)
		creator, err := builder.CreateProvisionInput(pp, internal.RuntimeVersionData{Version: "1.10.0", Origin: internal.Defaults})
		require.NoError(t, err)

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)

		// then
		assertComponentExists(t, input.KymaConfig.Components, gqlschema.ComponentConfigurationInput{
			Component: components.Tracing,
		})
		assertComponentExists(t, input.KymaConfig.Components, gqlschema.ComponentConfigurationInput{
			Component: components.Kiali,
		})
		assert.Len(t, input.KymaConfig.Components, 2)
	})

	t.Run("When creating UpgradeRuntimeInput", func(t *testing.T) {
		// given
		pp := fixProvisioningParameters(broker.AzurePlanID, "1.14.0")

		optionalComponentsDisablers := runtime.ComponentsDisablers{}
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).
			Return([]internal.KymaComponent{
				{Name: components.Kiali},
				{Name: components.Tracing},
				{Name: components.Backup},
			}, nil)

		builder, err := NewInputBuilderFactory(runtime.NewOptionalComponentsService(optionalComponentsDisablers), runtime.NewDisabledComponentsProvider(),
			componentsProvider, Config{}, "not-important", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)
		creator, err := builder.CreateUpgradeInput(pp, internal.RuntimeVersionData{Version: "1.14.0", Origin: internal.Defaults})
		require.NoError(t, err)

		// when
		input, err := creator.CreateUpgradeRuntimeInput()
		require.NoError(t, err)

		// then
		assertComponentExists(t, input.KymaConfig.Components, gqlschema.ComponentConfigurationInput{
			Component: components.Tracing,
		})
		assertComponentExists(t, input.KymaConfig.Components, gqlschema.ComponentConfigurationInput{
			Component: components.Kiali,
		})
		assert.Len(t, input.KymaConfig.Components, 2)
	})
}

func TestDisabledComponentsForPlanNotExist(t *testing.T) {
	// given
	pp := fixProvisioningParameters("invalid-plan", "")

	optionalComponentsDisablers := runtime.ComponentsDisablers{}
	componentsProvider := &automock.ComponentListProvider{}
	componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).
		Return([]internal.KymaComponent{
			{Name: components.Kiali},
			{Name: components.Tracing},
			{Name: components.Backup},
		}, nil)

	builder, err := NewInputBuilderFactory(runtime.NewOptionalComponentsService(optionalComponentsDisablers), runtime.NewDisabledComponentsProvider(),
		componentsProvider, Config{}, "not-important", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
	assert.NoError(t, err)
	// when
	_, err = builder.CreateProvisionInput(pp, emptyVersion)
	require.Error(t, err)
}

func TestInputBuilderFactoryOverrides(t *testing.T) {
	t.Run("should append overrides for the same components multiple times", func(t *testing.T) {
		// given
		var (
			dummyOptComponentsSvc = dummyOptionalComponentServiceMock(fixKymaComponentList())

			overridesA1 = []*gqlschema.ConfigEntryInput{
				{Key: "key-1", Value: "pico"},
				{Key: "key-2", Value: "bello"},
			}
			overridesA2 = []*gqlschema.ConfigEntryInput{
				{Key: "key-3", Value: "hakuna"},
				{Key: "key-4", Value: "matata", Secret: ptr.Bool(true)},
			}
		)

		pp := fixProvisioningParameters(broker.AzurePlanID, "")
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		builder, err := NewInputBuilderFactory(dummyOptComponentsSvc, runtime.NewDisabledComponentsProvider(),
			componentsProvider, Config{}, "not-important", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)
		creator, err := builder.CreateProvisionInput(pp, internal.RuntimeVersionData{Version: "1.10.0", Origin: internal.Defaults})
		require.NoError(t, err)

		// when
		creator.
			AppendOverrides("keb", overridesA1).
			AppendOverrides("keb", overridesA2)

		// then
		out, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)

		overriddenComponent, found := find(out.KymaConfig.Components, "keb")
		require.True(t, found)

		assertContainsAllOverrides(t, overriddenComponent.Configuration, overridesA1, overridesA2)
	})

	t.Run("should append global overrides for ProvisionRuntimeInput", func(t *testing.T) {
		// given
		var (
			optComponentsSvc = dummyOptionalComponentServiceMock(fixKymaComponentList())

			overridesA1 = []*gqlschema.ConfigEntryInput{
				{Key: "key-1", Value: "pico"},
				{Key: "key-2", Value: "bello"},
				{Key: "key-true", Value: "true"},
			}
			overridesA2 = []*gqlschema.ConfigEntryInput{
				{Key: "key-3", Value: "hakuna"},
				{Key: "key-4", Value: "matata", Secret: ptr.Bool(true)},
				{Key: "key-false", Value: "false"},
			}
		)
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		pp := fixProvisioningParameters(broker.AzurePlanID, "")
		builder, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(),
			componentsProvider, Config{}, "not-important", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)
		creator, err := builder.CreateProvisionInput(pp, internal.RuntimeVersionData{Version: "1.10.0", Origin: internal.Defaults})
		require.NoError(t, err)

		// when
		creator.
			AppendGlobalOverrides(overridesA1).
			AppendGlobalOverrides(overridesA2)

		// then
		out, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)

		assertContainsAllOverrides(t, out.KymaConfig.Configuration, overridesA1, overridesA2)
	})

	t.Run("should append global overrides for UpgradeRuntimeInput", func(t *testing.T) {
		// given
		var (
			optComponentsSvc = dummyOptionalComponentServiceMock(fixKymaComponentList())

			overridesA1 = []*gqlschema.ConfigEntryInput{
				{Key: "key-1", Value: "pico"},
				{Key: "key-2", Value: "bello"},
			}
			overridesA2 = []*gqlschema.ConfigEntryInput{
				{Key: "key-3", Value: "hakuna"},
				{Key: "key-4", Value: "matata", Secret: ptr.Bool(true)},
			}
		)
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		pp := fixProvisioningParameters(broker.AzurePlanID, "1.14.0")
		builder, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(),
			componentsProvider, Config{}, "not-important", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)
		creator, err := builder.CreateUpgradeInput(pp, internal.RuntimeVersionData{Version: "1.14.0", Origin: internal.Defaults})
		require.NoError(t, err)

		// when
		creator.
			AppendGlobalOverrides(overridesA1).
			AppendGlobalOverrides(overridesA2)

		// then
		out, err := creator.CreateUpgradeRuntimeInput()
		require.NoError(t, err)

		assertContainsAllOverrides(t, out.KymaConfig.Configuration, overridesA1, overridesA2)
	})

	t.Run("should overwrite already applied component and global overrides", func(t *testing.T) {
		// given
		var (
			dummyOptComponentsSvc = dummyOptionalComponentServiceMock(fixKymaComponentList())

			overridesA1 = []*gqlschema.ConfigEntryInput{
				{Key: "key-1", Value: "initial"},
				{Key: "key-2", Value: "bello"},
			}
			overridesA2 = []*gqlschema.ConfigEntryInput{
				{Key: "key-1", Value: "new"},
				{Key: "key-4", Value: "matata", Secret: ptr.Bool(true)},
			}
			globalOverrides1 = []*gqlschema.ConfigEntryInput{
				{Key: "key-g-1", Value: "initial-g"},
				{Key: "key-g-2", Value: "hakuna", Secret: ptr.Bool(true)},
			}
			globalOverrides2 = []*gqlschema.ConfigEntryInput{
				{Key: "key-g-1", Value: "new"},
				{Key: "key-g-4", Value: "matata", Secret: ptr.Bool(true)},
			}
		)

		pp := fixProvisioningParameters(broker.AzurePlanID, "")
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		builder, err := NewInputBuilderFactory(dummyOptComponentsSvc, runtime.NewDisabledComponentsProvider(),
			componentsProvider, Config{}, "not-important", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)
		creator, err := builder.CreateProvisionInput(pp, internal.RuntimeVersionData{Version: "1.10.0", Origin: internal.Defaults})
		require.NoError(t, err)

		// when
		creator.
			AppendOverrides("keb", overridesA1).
			AppendOverrides("keb", overridesA2).
			AppendGlobalOverrides(globalOverrides1).
			AppendGlobalOverrides(globalOverrides2)

		// then
		out, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)

		overriddenComponent, found := find(out.KymaConfig.Components, "keb")
		require.True(t, found)

		// assert component overrides
		assertContainsAllOverrides(t, overriddenComponent.Configuration, []*gqlschema.ConfigEntryInput{
			{Key: "key-1", Value: "new"},
			{Key: "key-2", Value: "bello"},
			{Key: "key-4", Value: "matata", Secret: ptr.Bool(true)},
		})

		// assert global overrides
		assertContainsAllOverrides(t, out.KymaConfig.Configuration, []*gqlschema.ConfigEntryInput{
			{Key: "key-g-1", Value: "new"},
			{Key: "key-g-2", Value: "hakuna", Secret: ptr.Bool(true)},
			{Key: "key-g-4", Value: "matata", Secret: ptr.Bool(true)},
		})
	})
}

func TestInputBuilderFactoryForAzurePlan(t *testing.T) {
	// given
	var (
		inputComponentList  = fixKymaComponentList()
		mappedComponentList = mapToGQLComponentConfigurationInput(inputComponentList)
		toDisableComponents = []string{"kiali"}
		kebOverrides        = []*gqlschema.ConfigEntryInput{
			{Key: "key-1", Value: "pico"},
			{Key: "key-2", Value: "bello", Secret: ptr.Bool(true)},
		}
	)

	optComponentsSvc := &automock.OptionalComponentService{}
	defer optComponentsSvc.AssertExpectations(t)
	optComponentsSvc.On("ComputeComponentsToDisable", []string{}).Return(toDisableComponents)
	optComponentsSvc.On("ExecuteDisablers", mappedComponentList, toDisableComponents[0]).Return(mappedComponentList, nil)

	config := Config{
		URL: "",
	}
	componentsProvider := &automock.ComponentListProvider{}
	componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(inputComponentList, nil)
	defer componentsProvider.AssertExpectations(t)

	factory, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(),
		componentsProvider, config, "1.10.0", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
	assert.NoError(t, err)
	pp := fixProvisioningParameters(broker.AzurePlanID, "")

	// when
	builder, err := factory.CreateProvisionInput(pp, internal.RuntimeVersionData{Version: "1.10.0", Origin: internal.Defaults})

	// then
	require.NoError(t, err)

	// when
	shootName := "c-51bcc12"
	builder.
		SetProvisioningParameters(internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				Name:         "azure-cluster",
				TargetSecret: ptr.String("azure-secret"),
				Purpose:      ptr.String("development"),
			},
		}).
		SetShootName(shootName).
		SetLabel("label1", "value1").
		SetShootDomain("shoot.domain.sap").
		AppendOverrides("keb", kebOverrides)
	input, err := builder.CreateProvisionRuntimeInput()
	require.NoError(t, err)
	clusterInput, err := builder.CreateProvisionClusterInput()
	require.NoError(t, err)

	// then
	assert.Equal(t, input.ClusterConfig, clusterInput.ClusterConfig)
	assert.Equal(t, input.RuntimeInput, clusterInput.RuntimeInput)
	assert.Nil(t, clusterInput.KymaConfig)
	assert.EqualValues(t, mappedComponentList, input.KymaConfig.Components)
	assert.Contains(t, input.RuntimeInput.Name, "azure-cluster")
	assert.Equal(t, "azure", input.ClusterConfig.GardenerConfig.Provider)
	assert.Equal(t, "azure-secret", input.ClusterConfig.GardenerConfig.TargetSecret)
	require.NotNil(t, input.ClusterConfig.GardenerConfig.Purpose)
	assert.Equal(t, "development", *input.ClusterConfig.GardenerConfig.Purpose)
	assert.Nil(t, input.ClusterConfig.GardenerConfig.LicenceType)
	assert.EqualValues(t, mappedComponentList, input.KymaConfig.Components)
	assert.Equal(t, shootName, input.ClusterConfig.GardenerConfig.Name)
	assert.NotNil(t, input.ClusterConfig.Administrators)
	assert.Equal(t, gqlschema.Labels{
		"label1": "value1",
	}, input.RuntimeInput.Labels)

	assertOverrides(t, "keb", input.KymaConfig.Components, kebOverrides)
}

func TestShouldAdjustRuntimeName(t *testing.T) {
	for name, tc := range map[string]struct {
		runtimeName               string
		expectedNameWithoutSuffix string
	}{
		"regular string": {
			runtimeName:               "test",
			expectedNameWithoutSuffix: "test",
		},
		"too long string": {
			runtimeName:               "this-string-is-too-long-because-it-has-more-than-36-chars",
			expectedNameWithoutSuffix: "this-string-is-too-long-becaus",
		},
		"string with forbidden chars": {
			runtimeName:               "CLUSTER-?name_123@!",
			expectedNameWithoutSuffix: "cluster-name123",
		},
		"too long string with forbidden chars": {
			runtimeName:               "ThisStringIsTooLongBecauseItHasMoreThan36Chars",
			expectedNameWithoutSuffix: "thisstringistoolongbecauseitha",
		},
	} {
		t.Run(name, func(t *testing.T) {
			// given
			optComponentsSvc := dummyOptionalComponentServiceMock(fixKymaComponentList())
			componentsProvider := &automock.ComponentListProvider{}
			componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

			builder, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(),
				componentsProvider, Config{TrialNodesNumber: 0}, "not-important", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
			assert.NoError(t, err)

			pp := fixProvisioningParameters(broker.TrialPlanID, "")
			pp.Parameters.Name = tc.runtimeName

			creator, err := builder.CreateProvisionInput(pp, internal.RuntimeVersionData{Version: "1.1.0", Origin: internal.Defaults})
			require.NoError(t, err)
			creator.SetProvisioningParameters(pp)

			// when
			input, err := creator.CreateProvisionRuntimeInput()
			require.NoError(t, err)
			clusterInput, err := creator.CreateProvisionClusterInput()
			require.NoError(t, err)

			// then
			assert.NotEqual(t, pp.Parameters.Name, input.RuntimeInput.Name)
			assert.LessOrEqual(t, len(input.RuntimeInput.Name), 36)
			assert.Equal(t, tc.expectedNameWithoutSuffix, input.RuntimeInput.Name[:len(input.RuntimeInput.Name)-6])
			assert.Equal(t, 1, input.ClusterConfig.GardenerConfig.AutoScalerMin)
			assert.Equal(t, 1, input.ClusterConfig.GardenerConfig.AutoScalerMax)
			assert.Equal(t, tc.expectedNameWithoutSuffix, clusterInput.RuntimeInput.Name[:len(input.RuntimeInput.Name)-6])
			assert.Equal(t, 1, clusterInput.ClusterConfig.GardenerConfig.AutoScalerMin)
			assert.Equal(t, 1, clusterInput.ClusterConfig.GardenerConfig.AutoScalerMax)
		})
	}
}

func TestShouldSetNumberOfNodesForTrialPlan(t *testing.T) {
	// given
	optComponentsSvc := dummyOptionalComponentServiceMock(fixKymaComponentList())
	componentsProvider := &automock.ComponentListProvider{}
	componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

	builder, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(),
		componentsProvider, Config{TrialNodesNumber: 2}, "not-important", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
	assert.NoError(t, err)

	pp := fixProvisioningParameters(broker.TrialPlanID, "")

	creator, err := builder.CreateProvisionInput(pp, internal.RuntimeVersionData{Version: "1.17.0", Origin: internal.Defaults})
	require.NoError(t, err)
	creator.SetProvisioningParameters(pp)

	// when
	input, err := creator.CreateProvisionRuntimeInput()
	require.NoError(t, err)
	clusterInput, err := creator.CreateProvisionClusterInput()
	require.NoError(t, err)

	// then
	assert.Equal(t, 2, input.ClusterConfig.GardenerConfig.AutoScalerMin)
	assert.Equal(t, 2, clusterInput.ClusterConfig.GardenerConfig.AutoScalerMax)
}

func TestShouldSetGlobalConfiguration(t *testing.T) {
	t.Run("When creating ProvisionRuntimeInput", func(t *testing.T) {
		// given
		optComponentsSvc := dummyOptionalComponentServiceMock(fixKymaComponentList())
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		builder, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(),
			componentsProvider, Config{}, "", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)

		pp := fixProvisioningParameters(broker.TrialPlanID, "")

		creator, err := builder.CreateProvisionInput(pp, internal.RuntimeVersionData{Version: "", Origin: internal.Defaults})
		require.NoError(t, err)
		creator.SetProvisioningParameters(pp)

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)

		// then
		expectedStrategy := gqlschema.ConflictStrategyReplace
		assert.Equal(t, &expectedStrategy, input.KymaConfig.ConflictStrategy)
	})

	t.Run("When creating UpgradeRuntimeInput", func(t *testing.T) {
		// given
		optComponentsSvc := dummyOptionalComponentServiceMock(fixKymaComponentList())
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		builder, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(),
			componentsProvider, Config{}, "", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)

		pp := fixProvisioningParameters(broker.TrialPlanID, "")

		creator, err := builder.CreateUpgradeInput(pp, internal.RuntimeVersionData{Version: "1.21.0", Origin: internal.Defaults})
		require.NoError(t, err)
		creator.SetProvisioningParameters(pp)

		// when
		input, err := creator.CreateUpgradeRuntimeInput()
		require.NoError(t, err)

		// then
		expectedStrategy := gqlschema.ConflictStrategyReplace
		assert.Equal(t, &expectedStrategy, input.KymaConfig.ConflictStrategy)
	})
}

func TestCreateProvisionRuntimeInput_ConfigureDNS(t *testing.T) {

	t.Run("should apply provided DNS Providers values", func(t *testing.T) {
		// given
		expectedDnsValues := &gqlschema.DNSConfigInput{
			Domain: "shoot-name.domain.sap",
			Providers: []*gqlschema.DNSProviderInput{
				&gqlschema.DNSProviderInput{
					DomainsInclude: []string{"devtest.kyma.ondemand.com"},
					Primary:        true,
					SecretName:     "aws_dns_domain_secrets_test_incustom",
					Type:           "route53_type_test",
				},
			},
		}

		id := uuid.New().String()

		optComponentsSvc := dummyOptionalComponentServiceMock(fixKymaComponentList())
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		inputBuilder, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.24.4", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)

		creator, err := inputBuilder.CreateProvisionInput(provisioningParams, internal.RuntimeVersionData{Version: "", Origin: internal.Defaults})
		require.NoError(t, err)
		setRuntimeProperties(creator)

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)
		clusterInput, err := creator.CreateProvisionClusterInput()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedDnsValues, input.ClusterConfig.GardenerConfig.DNSConfig)
		assert.Equal(t, expectedDnsValues, clusterInput.ClusterConfig.GardenerConfig.DNSConfig)
	})

	t.Run("should apply the DNS Providers values while DNS providers is empty", func(t *testing.T) {
		// given
		expectedDnsValues := &gqlschema.DNSConfigInput{
			Domain: "shoot-name.domain.sap",
		}

		id := uuid.New().String()

		optComponentsSvc := dummyOptionalComponentServiceMock(fixKymaComponentList())
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		inputBuilder, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.24.4", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)

		creator, err := inputBuilder.CreateProvisionInput(provisioningParams, internal.RuntimeVersionData{Version: "", Origin: internal.Defaults})
		require.NoError(t, err)
		setRuntimeProperties(creator)
		creator.SetShootDNSProviders(gardener.DNSProvidersData{})

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)
		clusterInput, err := creator.CreateProvisionClusterInput()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedDnsValues, input.ClusterConfig.GardenerConfig.DNSConfig)
		assert.Equal(t, expectedDnsValues, clusterInput.ClusterConfig.GardenerConfig.DNSConfig)
	})

}

func TestCreateProvisionRuntimeInput_ConfigureOIDC(t *testing.T) {

	t.Run("should apply default OIDC values when OIDC is nil", func(t *testing.T) {
		// given
		expectedOidcValues := &gqlschema.OIDCConfigInput{
			ClientID:       "9bd05ed7-a930-44e6-8c79-e6defeb7dec9",
			GroupsClaim:    "groups",
			IssuerURL:      "https://kymatest.accounts400.ondemand.com",
			SigningAlgs:    []string{"RS256"},
			UsernameClaim:  "sub",
			UsernamePrefix: "-",
		}

		id := uuid.New().String()

		optComponentsSvc := dummyOptionalComponentServiceMock(fixKymaComponentList())
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		inputBuilder, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.24.0", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)

		creator, err := inputBuilder.CreateProvisionInput(provisioningParams, internal.RuntimeVersionData{Version: "", Origin: internal.Defaults})
		require.NoError(t, err)

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)
		clusterInput, err := creator.CreateProvisionClusterInput()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedOidcValues, input.ClusterConfig.GardenerConfig.OidcConfig)
		assert.Equal(t, expectedOidcValues, clusterInput.ClusterConfig.GardenerConfig.OidcConfig)
	})

	t.Run("should apply default OIDC values when all OIDC fields are empty", func(t *testing.T) {
		// given
		expectedOidcValues := &gqlschema.OIDCConfigInput{
			ClientID:       "9bd05ed7-a930-44e6-8c79-e6defeb7dec9",
			GroupsClaim:    "groups",
			IssuerURL:      "https://kymatest.accounts400.ondemand.com",
			SigningAlgs:    []string{"RS256"},
			UsernameClaim:  "sub",
			UsernamePrefix: "-",
		}

		id := uuid.New().String()

		optComponentsSvc := dummyOptionalComponentServiceMock(fixKymaComponentList())
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		inputBuilder, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.24.0", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)
		provisioningParams.Parameters.OIDC = &internal.OIDCConfigDTO{}

		creator, err := inputBuilder.CreateProvisionInput(provisioningParams, internal.RuntimeVersionData{Version: "", Origin: internal.Defaults})
		require.NoError(t, err)

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)
		clusterInput, err := creator.CreateProvisionClusterInput()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedOidcValues, input.ClusterConfig.GardenerConfig.OidcConfig)
		assert.Equal(t, expectedOidcValues, clusterInput.ClusterConfig.GardenerConfig.OidcConfig)
	})

	t.Run("should apply provided OIDC values", func(t *testing.T) {
		// given
		expectedOidcValues := &gqlschema.OIDCConfigInput{
			ClientID:       "provided-id",
			GroupsClaim:    "fake-groups-claim",
			IssuerURL:      "https://test.domain.local",
			SigningAlgs:    []string{"RS256", "HS256"},
			UsernameClaim:  "usernameClaim",
			UsernamePrefix: "<<",
		}

		id := uuid.New().String()

		optComponentsSvc := dummyOptionalComponentServiceMock(fixKymaComponentList())
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		inputBuilder, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.24.0", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)
		provisioningParams.Parameters.OIDC = &internal.OIDCConfigDTO{
			ClientID:       "provided-id",
			GroupsClaim:    "fake-groups-claim",
			IssuerURL:      "https://test.domain.local",
			SigningAlgs:    []string{"RS256", "HS256"},
			UsernameClaim:  "usernameClaim",
			UsernamePrefix: "<<",
		}

		creator, err := inputBuilder.CreateProvisionInput(provisioningParams, internal.RuntimeVersionData{Version: "", Origin: internal.Defaults})
		require.NoError(t, err)

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)
		clusterInput, err := creator.CreateProvisionClusterInput()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedOidcValues, input.ClusterConfig.GardenerConfig.OidcConfig)
		assert.Equal(t, expectedOidcValues, clusterInput.ClusterConfig.GardenerConfig.OidcConfig)
	})

	t.Run("should normalize provided issuerURL", func(t *testing.T) {
		// given
		expectedOidcValues := &gqlschema.OIDCConfigInput{
			ClientID:       "provided-id",
			GroupsClaim:    "fake-groups-claim",
			IssuerURL:      "https://test.domain.local",
			SigningAlgs:    []string{"RS256", "HS256"},
			UsernameClaim:  "usernameClaim",
			UsernamePrefix: "<<",
		}

		id := uuid.New().String()

		optComponentsSvc := dummyOptionalComponentServiceMock(fixKymaComponentList())
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		inputBuilder, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.24.0", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)
		provisioningParams.Parameters.OIDC = &internal.OIDCConfigDTO{
			ClientID:       "provided-id",
			GroupsClaim:    "fake-groups-claim",
			IssuerURL:      "https://test.domain.local/",
			SigningAlgs:    []string{"RS256", "HS256"},
			UsernameClaim:  "usernameClaim",
			UsernamePrefix: "<<",
		}

		creator, err := inputBuilder.CreateProvisionInput(provisioningParams, internal.RuntimeVersionData{Version: "", Origin: internal.Defaults})
		require.NoError(t, err)

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)
		clusterInput, err := creator.CreateProvisionClusterInput()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedOidcValues, input.ClusterConfig.GardenerConfig.OidcConfig)
		assert.Equal(t, expectedOidcValues, clusterInput.ClusterConfig.GardenerConfig.OidcConfig)
	})
}

func TestCreateClusterConfiguration_Overrides(t *testing.T) {
	t.Run("Should apply component and global overrides with proper types", func(t *testing.T) {
		// given
		id := uuid.New().String()

		componentList := []internal.KymaComponent{
			{Name: "dex", Namespace: "kyma-system"},
			{Name: "ory", Namespace: "kyma-system"},
			{
				Name:      "custom",
				Namespace: "kyma-system",
				Source:    &internal.ComponentSource{URL: "http://source.url"},
			},
		}

		optComponentsSvc := dummyOptionalComponentServiceMock(componentList)
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(componentList, nil)
		inputBuilder, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.24.0", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)
		creator, err := inputBuilder.CreateProvisionInput(provisioningParams, internal.RuntimeVersionData{Version: "", Origin: internal.Defaults})
		require.NoError(t, err)
		setRuntimeProperties(creator)
		creator.AppendOverrides("dex", []*gqlschema.ConfigEntryInput{
			{Key: "key-1", Value: "pico"},
			{Key: "key-false", Value: "false"},
			{Key: "key-true", Value: "true"},
			{Key: "key-secret", Value: "classified", Secret: ptr.Bool(true)},
		})
		creator.AppendGlobalOverrides([]*gqlschema.ConfigEntryInput{
			{Key: "global-key-string", Value: "global-pico"},
			{Key: "global-key-false", Value: "false"},
			{Key: "global-key-true", Value: "true"},
			{Key: "global-key-secret", Value: "global-classified", Secret: ptr.Bool(true)},
		})

		// when
		inventoryInput, err := creator.CreateClusterConfiguration()
		require.NoError(t, err)

		// then
		assertAllConfigsContainsGlobals(t, inventoryInput.KymaConfig.Components, "shoot-name.domain.sap")
		assert.Equal(t, reconcilerApi.Component{
			URL:       "",
			Component: "dex",
			Namespace: "kyma-system",
			Configuration: []reconcilerApi.Configuration{
				{Key: "global.domainName", Value: "shoot-name.domain.sap", Secret: false},
				{Key: "global-key-string", Value: "global-pico", Secret: false},
				{Key: "global-key-false", Value: false, Secret: false},
				{Key: "global-key-true", Value: true, Secret: false},
				{Key: "global-key-secret", Value: "global-classified", Secret: true},
				{Key: "key-1", Value: "pico", Secret: false},
				{Key: "key-false", Value: false, Secret: false},
				{Key: "key-true", Value: true, Secret: false},
				{Key: "key-secret", Value: "classified", Secret: true},
			},
		}, inventoryInput.KymaConfig.Components[0])

		// check custom source URL
		for _, component := range inventoryInput.KymaConfig.Components {
			if component.Component == "custom" {
				assert.Equal(t, "http://source.url", component.URL)
			}
		}
	})

	t.Run("should overwrite already existing component and global overrides", func(t *testing.T) {
		// given
		var (
			dummyOptComponentsSvc = dummyOptionalComponentServiceMock(fixKymaComponentList())

			overridesA1 = []*gqlschema.ConfigEntryInput{
				{Key: "key-1", Value: "initial"},
				{Key: "key-2", Value: "bello"},
			}
			overridesA2 = []*gqlschema.ConfigEntryInput{
				{Key: "key-1", Value: "new"},
				{Key: "key-4", Value: "matata", Secret: ptr.Bool(true)},
			}
			globalOverrides1 = []*gqlschema.ConfigEntryInput{
				{Key: "key-g-1", Value: "initial-g"},
				{Key: "key-g-2", Value: "hakuna", Secret: ptr.Bool(true)},
			}
			globalOverrides2 = []*gqlschema.ConfigEntryInput{
				{Key: "key-g-1", Value: "new"},
				{Key: "key-g-4", Value: "matata", Secret: ptr.Bool(true)},
			}
		)

		pp := fixProvisioningParameters(broker.AzurePlanID, "2.0.0-rc6")
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		builder, err := NewInputBuilderFactory(dummyOptComponentsSvc, runtime.NewDisabledComponentsProvider(),
			componentsProvider, Config{}, "not-important", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)
		creator, err := builder.CreateProvisionInput(pp, internal.RuntimeVersionData{Version: "1.10.0", Origin: internal.Defaults})
		require.NoError(t, err)
		setRuntimeProperties(creator)

		// when
		creator.
			AppendOverrides("keb", overridesA1).
			AppendOverrides("keb", overridesA2).
			AppendGlobalOverrides(globalOverrides1).
			AppendGlobalOverrides(globalOverrides2)

		// then
		out, err := creator.CreateClusterConfiguration()
		require.NoError(t, err)
		t.Logf("out %+v\n", out)

		overriddenComponent, found := findForReconciler(out.KymaConfig.Components, "keb")
		require.True(t, found)
		t.Logf("overriddenComponent %+v\n", overriddenComponent)

		assertAllConfigsContainsGlobals(t, []reconcilerApi.Component{overriddenComponent}, "shoot-name.domain.sap")
		// assert component and global overrides
		assertContainsAllOverridesForReconciler(t, overriddenComponent.Configuration, []*gqlschema.ConfigEntryInput{
			{Key: "global.domainName", Value: "shoot-name.domain.sap"},
			{Key: "key-1", Value: "new"},
			{Key: "key-2", Value: "bello"},
			{Key: "key-4", Value: "matata", Secret: ptr.Bool(true)},
			{Key: "key-g-1", Value: "new"},
			{Key: "key-g-2", Value: "hakuna", Secret: ptr.Bool(true)},
			{Key: "key-g-4", Value: "matata", Secret: ptr.Bool(true)},
		})
	})
}

func TestCreateProvisionRuntimeInput_ConfigureAdmins(t *testing.T) {
	t.Run("should apply default admin from user_id field", func(t *testing.T) {
		// given
		expectedAdmins := []string{"fake-user-id"}

		id := uuid.New().String()

		optComponentsSvc := dummyOptionalComponentServiceMock(fixKymaComponentList())
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		inputBuilder, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.24.0", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)
		provisioningParams.ErsContext.UserID = expectedAdmins[0]

		creator, err := inputBuilder.CreateProvisionInput(provisioningParams, internal.RuntimeVersionData{Version: "", Origin: internal.Defaults})
		require.NoError(t, err)
		setRuntimeProperties(creator)

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)
		inventoryInput, err := creator.CreateClusterConfiguration()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedAdmins, input.ClusterConfig.Administrators)
		assert.Equal(t, expectedAdmins, inventoryInput.KymaConfig.Administrators)
	})

	t.Run("should apply new admin list", func(t *testing.T) {
		// given
		expectedAdmins := []string{"newAdmin1@test.com", "newAdmin2@test.com"}

		id := uuid.New().String()

		optComponentsSvc := dummyOptionalComponentServiceMock(fixKymaComponentList())
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		inputBuilder, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.24.0", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)
		provisioningParams.Parameters.RuntimeAdministrators = expectedAdmins

		creator, err := inputBuilder.CreateProvisionInput(provisioningParams, internal.RuntimeVersionData{Version: "", Origin: internal.Defaults})
		require.NoError(t, err)
		setRuntimeProperties(creator)

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)
		inventoryInput, err := creator.CreateClusterConfiguration()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedAdmins, input.ClusterConfig.Administrators)
		assert.Equal(t, expectedAdmins, inventoryInput.KymaConfig.Administrators)
	})
}

func assertAllConfigsContainsGlobals(t *testing.T, components []reconcilerApi.Component, domainName string) {
	for _, cmp := range components {
		found := false
		for _, cfg := range cmp.Configuration {
			if cfg.Key == "global.domainName" {
				assert.Equal(t, domainName, cfg.Value)
				found = true
			}
		}
		assert.True(t, found, "Component %s must contain `global.domainName` config", cmp.Component)
	}
}

func setRuntimeProperties(creator internal.ProvisionerInputCreator) {
	creator.SetKubeconfig("example kubeconfig payload")
	creator.SetRuntimeID("runtimeID")
	creator.SetInstanceID("instanceID")
	creator.SetShootName("shoot-name")
	creator.SetShootDomain("shoot-name.domain.sap")
	creator.SetShootDNSProviders(fixture.FixDNSProvidersConfig())
}

func TestCreateUpgradeRuntimeInput_ConfigureAdmins(t *testing.T) {
	t.Run("should not overwrite default admin (from user_id)", func(t *testing.T) {
		// given
		expectedAdmins := []string{"fake-user-id"}

		id := uuid.New().String()

		optComponentsSvc := dummyOptionalComponentServiceMock(fixKymaComponentList())
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		inputBuilder, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.24.0", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)
		provisioningParams.ErsContext.UserID = expectedAdmins[0]

		creator, err := inputBuilder.CreateUpgradeShootInput(provisioningParams)
		require.NoError(t, err)

		// when
		creator.SetProvisioningParameters(provisioningParams)
		input, err := creator.CreateUpgradeShootInput()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedAdmins, input.Administrators)
	})

	t.Run("should overwrite default admin with new admins list", func(t *testing.T) {
		// given
		userId := "fake-user-id"
		expectedAdmins := []string{"newAdmin1@test.com", "newAdmin2@test.com"}

		id := uuid.New().String()

		optComponentsSvc := dummyOptionalComponentServiceMock(fixKymaComponentList())
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		inputBuilder, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.24.0", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)
		provisioningParams.ErsContext.UserID = userId
		provisioningParams.Parameters.RuntimeAdministrators = expectedAdmins

		creator, err := inputBuilder.CreateUpgradeShootInput(provisioningParams)
		require.NoError(t, err)

		// when
		creator.SetProvisioningParameters(provisioningParams)
		input, err := creator.CreateUpgradeShootInput()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedAdmins, input.Administrators)
	})
}

func TestCreateUpgradeShootInput_ConfigureAutoscalerParams(t *testing.T) {
	t.Run("should apply CreateUpgradeShootInput with provisioning autoscaler parameters", func(t *testing.T) {
		// given
		optComponentsSvc := dummyOptionalComponentServiceMock(fixKymaComponentList())
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		ibf, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.24.0", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)

		//ar provider HyperscalerInputProvider

		pp := fixProvisioningParameters(broker.GCPPlanID, "")
		//provider = &cloudProvider.GcpInput{} // for broker.GCPPlanID

		rtinput, err := ibf.CreateUpgradeShootInput(pp)

		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, rtinput)

		rtinput = rtinput.SetProvisioningParameters(pp)
		input, err := rtinput.CreateUpgradeShootInput()
		assert.NoError(t, err)

		expectAutoscalerMin := *pp.Parameters.AutoScalerMin
		expectAutoscalerMax := *pp.Parameters.AutoScalerMax
		expectMaxSurge := *pp.Parameters.MaxSurge
		expectMaxUnavailable := *pp.Parameters.MaxUnavailable
		t.Logf("%v, %v, %v, %v", expectAutoscalerMin, expectAutoscalerMax, expectMaxSurge, expectMaxUnavailable)

		assert.Equal(t, expectAutoscalerMin, *input.GardenerConfig.AutoScalerMin)
		assert.Equal(t, expectAutoscalerMax, *input.GardenerConfig.AutoScalerMax)
		assert.Equal(t, expectMaxSurge, *input.GardenerConfig.MaxSurge)
		assert.Equal(t, expectMaxUnavailable, *input.GardenerConfig.MaxUnavailable)
	})

	t.Run("should apply CreateUpgradeShootInput with provider autoscaler parameters", func(t *testing.T) {
		// given
		optComponentsSvc := dummyOptionalComponentServiceMock(fixKymaComponentList())
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("internal.RuntimeVersionData"), mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		ibf, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.24.0", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)

		pp := fixProvisioningParameters(broker.GCPPlanID, "")
		pp.Parameters.AutoScalerMin = nil
		pp.Parameters.AutoScalerMax = nil
		pp.Parameters.MaxSurge = nil
		pp.Parameters.MaxUnavailable = nil

		provider := &cloudProvider.GcpInput{} // for broker.GCPPlanID

		rtinput, err := ibf.CreateUpgradeShootInput(pp)

		assert.NoError(t, err)
		require.IsType(t, &RuntimeInput{}, rtinput)

		rtinput = rtinput.SetProvisioningParameters(pp)
		input, err := rtinput.CreateUpgradeShootInput()
		assert.NoError(t, err)

		expectAutoscalerMin := provider.Defaults().GardenerConfig.AutoScalerMin
		expectAutoscalerMax := provider.Defaults().GardenerConfig.AutoScalerMax
		expectMaxSurge := provider.Defaults().GardenerConfig.MaxSurge
		expectMaxUnavailable := provider.Defaults().GardenerConfig.MaxUnavailable

		assert.Equal(t, expectAutoscalerMin, *input.GardenerConfig.AutoScalerMin)
		assert.Equal(t, expectAutoscalerMax, *input.GardenerConfig.AutoScalerMax)
		assert.Equal(t, expectMaxSurge, *input.GardenerConfig.MaxSurge)
		assert.Equal(t, expectMaxUnavailable, *input.GardenerConfig.MaxUnavailable)
	})
}

func assertOverrides(t *testing.T, componentName string, components internal.ComponentConfigurationInputList, overrides []*gqlschema.ConfigEntryInput) {
	overriddenComponent, found := find(components, componentName)
	require.True(t, found)

	assert.Equal(t, overriddenComponent.Configuration, overrides)
}

func find(in internal.ComponentConfigurationInputList, name string) (*gqlschema.ComponentConfigurationInput, bool) {
	for _, c := range in {
		if c.Component == name {
			return c, true
		}
	}
	return nil, false
}

func findForReconciler(in []reconcilerApi.Component, name string) (reconcilerApi.Component, bool) {
	for _, c := range in {
		if c.Component == name {
			return c, true
		}
	}
	return reconcilerApi.Component{}, false
}

func fixKymaComponentList() []internal.KymaComponent {
	return []internal.KymaComponent{
		{Name: "dex", Namespace: "kyma-system"},
		{Name: "ory", Namespace: "kyma-system"},
		{Name: "keb", Namespace: "kyma-system"},
	}
}

func dummyOptionalComponentServiceMock(inputComponentList []internal.KymaComponent) *automock.OptionalComponentService {
	mappedComponentList := mapToGQLComponentConfigurationInput(inputComponentList)

	optComponentsSvc := &automock.OptionalComponentService{}
	optComponentsSvc.On("ComputeComponentsToDisable", []string{}).Return([]string{})
	optComponentsSvc.On("ExecuteDisablers", mappedComponentList).Return(mappedComponentList, nil)
	return optComponentsSvc
}

func assertContainsAllOverrides(t *testing.T, gotOverrides []*gqlschema.ConfigEntryInput, expOverrides ...[]*gqlschema.ConfigEntryInput) {
	var expected []*gqlschema.ConfigEntryInput
	for _, o := range expOverrides {
		expected = append(expected, o...)
	}

	require.Len(t, gotOverrides, len(expected))
	for _, o := range expected {
		assert.Contains(t, gotOverrides, o)
	}
}

func assertContainsAllOverridesForReconciler(t *testing.T, gotOverrides []reconcilerApi.Configuration, expOverrides []*gqlschema.ConfigEntryInput) {
	var expected []reconcilerApi.Configuration
	for _, o := range expOverrides {
		expected = append(expected, reconcilerApi.Configuration{
			Key:    o.Key,
			Value:  o.Value,
			Secret: falseIfNil(o.Secret),
		})
	}

	require.Len(t, gotOverrides, len(expected))
	for _, o := range expected {
		assert.Contains(t, gotOverrides, o)
	}
}

func assertComponentExists(t *testing.T,
	components []*gqlschema.ComponentConfigurationInput,
	expected gqlschema.ComponentConfigurationInput) {

	for _, component := range components {
		if component.Component == expected.Component {
			return
		}
	}
	assert.Failf(t, "component list does not contain %s", expected.Component)
}
