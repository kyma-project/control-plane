package input

import (
	"testing"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/process/input/automock"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime/components"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
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
		componentsProvider.On("AllComponents", mock.AnythingOfType("string")).
			Return([]v1alpha1.KymaComponent{
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
		componentsProvider.On("AllComponents", mock.AnythingOfType("string")).
			Return([]v1alpha1.KymaComponent{
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
		componentsProvider.On("AllComponents", mock.AnythingOfType("string")).
			Return([]v1alpha1.KymaComponent{
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
		componentsProvider.On("AllComponents", mock.AnythingOfType("string")).
			Return([]v1alpha1.KymaComponent{
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
	componentsProvider.On("AllComponents", mock.AnythingOfType("string")).
		Return([]v1alpha1.KymaComponent{
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
		componentsProvider.On("AllComponents", mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

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

		assertContainsAllOverrides(t, overriddenComponent.Configuration, overridesA1, overridesA1)
	})

	t.Run("should append global overrides for ProvisionRuntimeInput", func(t *testing.T) {
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
		componentsProvider.On("AllComponents", mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

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

		assertContainsAllOverrides(t, out.KymaConfig.Configuration, overridesA1, overridesA1)
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
		componentsProvider.On("AllComponents", mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

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

		assertContainsAllOverrides(t, out.KymaConfig.Configuration, overridesA1, overridesA1)
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
	componentsProvider.On("AllComponents", mock.AnythingOfType("string")).Return(inputComponentList, nil)
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
	input, err := builder.
		SetProvisioningParameters(internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				Name:         "azure-cluster",
				TargetSecret: ptr.String("azure-secret"),
				Purpose:      ptr.String("development"),
			},
		}).
		SetShootName(shootName).
		SetLabel("label1", "value1").
		AppendOverrides("keb", kebOverrides).CreateProvisionRuntimeInput()

	// then
	require.NoError(t, err)
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
	assert.Equal(t, &gqlschema.Labels{
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
			componentsProvider.On("AllComponents", mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

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

			// then
			assert.NotEqual(t, pp.Parameters.Name, input.RuntimeInput.Name)
			assert.LessOrEqual(t, len(input.RuntimeInput.Name), 36)
			assert.Equal(t, tc.expectedNameWithoutSuffix, input.RuntimeInput.Name[:len(input.RuntimeInput.Name)-6])
			assert.Equal(t, 1, input.ClusterConfig.GardenerConfig.AutoScalerMin)
			assert.Equal(t, 1, input.ClusterConfig.GardenerConfig.AutoScalerMax)
		})
	}
}

func TestShouldSetNumberOfNodesForTrialPlan(t *testing.T) {
	// given
	optComponentsSvc := dummyOptionalComponentServiceMock(fixKymaComponentList())
	componentsProvider := &automock.ComponentListProvider{}
	componentsProvider.On("AllComponents", mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

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

	// then
	assert.Equal(t, 2, input.ClusterConfig.GardenerConfig.AutoScalerMin)
	assert.Equal(t, 2, input.ClusterConfig.GardenerConfig.AutoScalerMax)
}

func TestShouldSetGlobalConfiguration(t *testing.T) {
	t.Run("When creating ProvisionRuntimeInput", func(t *testing.T) {
		// given
		optComponentsSvc := dummyOptionalComponentServiceMock(fixKymaComponentList())
		componentsProvider := &automock.ComponentListProvider{}
		componentsProvider.On("AllComponents", mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

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
		componentsProvider.On("AllComponents", mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

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
		componentsProvider.On("AllComponents", mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

		inputBuilder, err := NewInputBuilderFactory(optComponentsSvc, runtime.NewDisabledComponentsProvider(), componentsProvider,
			Config{}, "1.24.0", fixTrialRegionMapping(), fixTrialProviders(), fixture.FixOIDCConfigDTO())
		assert.NoError(t, err)

		provisioningParams := fixture.FixProvisioningParameters(id)

		creator, err := inputBuilder.CreateProvisionInput(provisioningParams, internal.RuntimeVersionData{Version: "", Origin: internal.Defaults})
		require.NoError(t, err)

		// when
		input, err := creator.CreateProvisionRuntimeInput()
		require.NoError(t, err)

		// then
		assert.Equal(t, expectedOidcValues, input.ClusterConfig.GardenerConfig.OidcConfig)
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
		componentsProvider.On("AllComponents", mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

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

		// then
		assert.Equal(t, expectedOidcValues, input.ClusterConfig.GardenerConfig.OidcConfig)
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
		componentsProvider.On("AllComponents", mock.AnythingOfType("string")).Return(fixKymaComponentList(), nil)

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

		// then
		assert.Equal(t, expectedOidcValues, input.ClusterConfig.GardenerConfig.OidcConfig)
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

func fixKymaComponentList() []v1alpha1.KymaComponent {
	return []v1alpha1.KymaComponent{
		{Name: "dex", Namespace: "kyma-system"},
		{Name: "ory", Namespace: "kyma-system"},
		{Name: "keb", Namespace: "kyma-system"},
	}
}

func dummyOptionalComponentServiceMock(inputComponentList []v1alpha1.KymaComponent) *automock.OptionalComponentService {
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
