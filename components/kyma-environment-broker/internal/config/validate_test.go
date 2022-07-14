package config_test

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	planWithWrongConfig           = "wrong"
	additionalComponentsConfigKey = "additional-components"
)

func TestValidate(t *testing.T) {
	// setup
	ctx := context.TODO()
	cfgMap, err := fixConfigMap()
	require.NoError(t, err)

	fakeK8sClient := fake.NewClientBuilder().WithRuntimeObjects(cfgMap).Build()
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	cfgReader := config.NewConfigReader(ctx, fakeK8sClient, logger)

	t.Run("should validate whether config contains required fields", func(t *testing.T) {
		// given
		cfgString, err := cfgReader.ReadConfig(kymaVersion, broker.AzurePlanName)
		require.NoError(t, err)

		// when
		err = config.Validate(cfgString)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error indicating missing required fields", func(t *testing.T) {
		// given
		cfgString, err := cfgReader.ReadConfig(kymaVersion, planWithWrongConfig)
		require.NoError(t, err)

		// when
		err = config.Validate(cfgString)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), additionalComponentsConfigKey)
		logger.Error(err)
	})
}
