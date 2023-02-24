package runtime_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	kymaVersion                 = "2.2.0"
	requiredComponentsURLFormat = "https://storage.googleapis.com/kyma-prow-artifacts/%s/kyma-components.yaml"
)

func TestComponentsProviderSuccessFlow(t *testing.T) {
	t.Run("should fetch required components and additional components", func(t *testing.T) {
		// given
		cfg := fixConfigForPlan()
		componentsProvider := runtime.NewFakeComponentsProvider()
		expectedPrerequisiteComponent := internal.KymaComponent{
			Name:      "cluster-essentials",
			Namespace: "kyma-system",
		}
		expectedRequiredComponent := internal.KymaComponent{
			Name:      "serverless",
			Namespace: "kyma-system",
		}
		expectedAdditionalComponent := internal.KymaComponent{
			Name:      "new-component1",
			Namespace: "kyma-system",
			Source: &internal.ComponentSource{
				URL: "https://local.test/kyma-additional-components/new-component1.tgz"},
		}
		unexpectedAdditionalComponent := internal.KymaComponent{
			Name:      "test-component1",
			Namespace: "kyma-system",
		}

		// when
		allComponents, err := componentsProvider.AllComponents(internal.RuntimeVersionData{
			Version:      kymaVersion,
			Origin:       internal.Parameters,
			MajorVersion: 2,
		}, cfg)

		// then
		require.NoError(t, err)
		assert.NotEmpty(t, allComponents)
		assert.Contains(t, allComponents, expectedPrerequisiteComponent)
		assert.Contains(t, allComponents, expectedRequiredComponent)
		assert.Contains(t, allComponents, expectedAdditionalComponent)
		assert.NotContains(t, allComponents, unexpectedAdditionalComponent)
	})
}

func TestComponentsProviderErrors(t *testing.T) {
	t.Run("should return 404 Not Found when required components are not available for given Kyma version", func(t *testing.T) {
		// given
		cfg := fixConfigForPlan()
		componentsProvider := runtime.NewFakeComponentsProvider()

		wrongVer := "test-2.2.0"

		errMsg := fmt.Sprintf("while checking response status code for Kyma components list: "+
			"got unexpected status code, want %d, got %d, url: %s, body: %s",
			http.StatusOK, 404, fmt.Sprintf(requiredComponentsURLFormat, wrongVer), "")
		expectedErr := throwError(errMsg)

		// when
		_, err := componentsProvider.AllComponents(internal.RuntimeVersionData{
			Version:      wrongVer,
			Origin:       internal.Parameters,
			MajorVersion: 2,
		}, cfg)

		// then
		require.Error(t, err)
		assert.Equal(t, expectedErr, errors.Unwrap(err))
	})
}

func throwError(message string) error {
	return fmt.Errorf(message)
}

func fixConfigForPlan() *internal.ConfigForPlan {
	return &internal.ConfigForPlan{AdditionalComponents: []internal.KymaComponent{
		{
			Name:      "compass-runtime-agent",
			Namespace: "kyma-system",
		},
		{
			Name:      "new-component1",
			Namespace: "kyma-system",
			Source:    &internal.ComponentSource{URL: "https://local.test/kyma-additional-components/new-component1.tgz"},
		},
		{
			Name:      "new-component2",
			Namespace: "kyma-system",
			Source:    &internal.ComponentSource{URL: "https://local.test/kyma-additional-components/new-component2.tgz"},
		},
		{
			Name:      "new-component3",
			Namespace: "kyma-system",
		},
	}}
}
