package config_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const wrongConfigPlan = "wrong"

func TestConfigProvider(t *testing.T) {
	// setup
	ctx := context.TODO()
	cfgMap, err := fixConfigMap()
	require.NoError(t, err)

	fakeK8sClient := fake.NewClientBuilder().WithRuntimeObjects(cfgMap).Build()
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	cfgReader := config.NewConfigMapReader(ctx, fakeK8sClient, logger, kymaVersion)
	cfgValidator := config.NewConfigMapKeysValidator()
	cfgConverter := config.NewConfigMapConverter()
	cfgProvider := config.NewConfigProvider(cfgReader, cfgValidator, cfgConverter)

	t.Run("should provide config for Kyma 2.4.0 azure plan", func(t *testing.T) {
		// given
		expectedCfg := fixAzureConfig()
		// when
		cfg, err := cfgProvider.ProvideForGivenVersionAndPlan(kymaVersion, broker.AzurePlanName)

		// then
		require.NoError(t, err)
		assert.Len(t, cfg.AdditionalComponents, len(expectedCfg.AdditionalComponents))
		assert.ObjectsAreEqual(expectedCfg, cfg)
	})

	t.Run("should provide config for a default", func(t *testing.T) {
		// given
		expectedCfg := fixDefault()
		// when
		cfg, err := cfgProvider.ProvideForGivenVersionAndPlan(kymaVersion, broker.AWSPlanName)

		// then
		require.NoError(t, err)
		assert.Len(t, cfg.AdditionalComponents, len(expectedCfg.AdditionalComponents))
		assert.ObjectsAreEqual(expectedCfg, cfg)
	})

	t.Run("should provide config for default Kyma version and azure plan when PR-* Kyma version is passed", func(t *testing.T) {
		// given
		expectedCfg := fixAzureConfig()
		customKymaVer := "PR-1234"
		// when
		cfg, err := cfgProvider.ProvideForGivenVersionAndPlan(customKymaVer, broker.AzurePlanName)

		// then
		require.NoError(t, err)
		assert.Len(t, cfg.AdditionalComponents, len(expectedCfg.AdditionalComponents))
		assert.ObjectsAreEqual(expectedCfg, cfg)
	})

	t.Run("should provide config for default Kyma version and azure plan when main-* Kyma version is passed", func(t *testing.T) {
		// given
		expectedCfg := fixAzureConfig()
		customKymaVer := "main-fffff"
		// when
		cfg, err := cfgProvider.ProvideForGivenVersionAndPlan(customKymaVer, broker.AzurePlanName)

		// then
		require.NoError(t, err)
		assert.Len(t, cfg.AdditionalComponents, len(expectedCfg.AdditionalComponents))
		assert.ObjectsAreEqual(expectedCfg, cfg)
	})

	t.Run("validator should return error indicating missing required fields", func(t *testing.T) {
		// given
		expectedMissingConfigKeys := []string{
			"additional-components",
		}
		expectedErrMsg := fmt.Sprintf("missing required configuration entires: %s", strings.Join(expectedMissingConfigKeys, ","))
		// when
		cfg, err := cfgProvider.ProvideForGivenVersionAndPlan(kymaVersion, wrongConfigPlan)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, expectedErrMsg)
		assert.Nil(t, cfg)
	})

	t.Run("reader should return error indicating missing configmap", func(t *testing.T) {
		// given
		err = fakeK8sClient.Delete(ctx, cfgMap)
		require.NoError(t, err)

		// when
		cfg, err := cfgProvider.ProvideForGivenVersionAndPlan(kymaVersion, broker.AzurePlanName)

		// then
		require.Error(t, err)
		assert.Equal(t, "configmap with configuration does not exist", errors.Unwrap(err).Error())
		assert.Nil(t, cfg)
	})
}

func fixAzureConfig() *internal.ConfigForPlan {
	return &internal.ConfigForPlan{
		AdditionalComponents: []internal.KymaComponent{
			{
				Name:      "additional-component1",
				Namespace: "kyma-system",
			},
			{
				Name:      "additional-component2",
				Namespace: "test-system",
			},
			{
				Name:      "azure-component",
				Namespace: "azure-system",
				Source:    &internal.ComponentSource{URL: "https://azure.domain/component/azure-component.git"},
			},
		}}
}

func fixDefault() *internal.ConfigForPlan {
	return &internal.ConfigForPlan{
		KymaTemplate: `apiVersion: operator.kyma-project.io/v1beta2
kind: Kyma
metadata:
  name: my-kyma1
  namespace: kyma-system
spec:
  channel: stable
  modules:
  - name: istio`,
		AdditionalComponents: []internal.KymaComponent{
			{
				Name:      "additional-component1",
				Namespace: "kyma-system",
			},
			{
				Name:      "additional-component2",
				Namespace: "test-system",
			},
			{
				Name:      "azure-component",
				Namespace: "azure-system",
				Source:    &internal.ComponentSource{URL: "https://azure.domain/component/azure-component.git"},
			},
		}}
}
