package runtime_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	kymaVersion                 = "2.2.0"
	additionalComponentsYaml    = "additional-runtime-components.yaml"
	requiredComponentsURLFormat = "https://storage.googleapis.com/kyma-prow-artifacts/%s/kyma-components.yaml"
)

func TestComponentsProviderSuccessFlow(t *testing.T) {
	t.Run("should fetch required components and default additional components", func(t *testing.T) {
		// given
		ctx := context.TODO()
		k8sClient := fake.NewClientBuilder().Build()
		yamlPath := path.Join("testdata", additionalComponentsYaml)
		componentsProvider := runtime.NewFakeComponentsProvider(ctx, k8sClient, yamlPath)
		expectedPrerequisiteComponent := runtime.KymaComponent{
			Name:      "cluster-essentials",
			Namespace: "kyma-system",
		}
		expectedRequiredComponent := runtime.KymaComponent{
			Name:      "serverless",
			Namespace: "kyma-system",
		}
		expectedAdditionalComponent := runtime.KymaComponent{
			Name:      "new-component1",
			Namespace: "kyma-system",
			Source: &runtime.ComponentSource{
				URL: "https://local.test/kyma-additional-components/new-component1.tgz"},
		}
		unexpectedAdditionalComponent := runtime.KymaComponent{
			Name:      "test-component1",
			Namespace: "kyma-system",
		}

		// when
		allComponents, err := componentsProvider.AllComponents(internal.RuntimeVersionData{
			Version:      kymaVersion,
			Origin:       internal.Parameters,
			MajorVersion: 2,
		}, "")

		// then
		require.NoError(t, err)
		assert.NotEmpty(t, allComponents)
		assert.Contains(t, allComponents, expectedPrerequisiteComponent)
		assert.Contains(t, allComponents, expectedRequiredComponent)
		assert.Contains(t, allComponents, expectedAdditionalComponent)
		assert.NotContains(t, allComponents, unexpectedAdditionalComponent)
	})

	t.Run("should fetch required components and additional components overrides", func(t *testing.T) {
		// given
		ctx := context.TODO()
		k8sClient := fake.NewClientBuilder().WithRuntimeObjects(fixK8sResources()...).Build()
		yamlPath := path.Join("testdata", additionalComponentsYaml)
		componentsProvider := runtime.NewFakeComponentsProvider(ctx, k8sClient, yamlPath)
		expectedPrerequisiteComponent := runtime.KymaComponent{
			Name:      "cluster-essentials",
			Namespace: "kyma-system",
		}
		expectedRequiredComponent := runtime.KymaComponent{
			Name:      "serverless",
			Namespace: "kyma-system",
		}
		expectedAdditionalComponent1 := runtime.KymaComponent{
			Name:      "test-component1",
			Namespace: "kyma-system",
		}
		expectedAdditionalComponent2 := runtime.KymaComponent{
			Name:      "test-component2",
			Namespace: "compass-system",
			Source: &runtime.ComponentSource{
				URL: "https://test.local/test-component2.tgz"},
		}
		unexpectedAdditionalComponent1 := runtime.KymaComponent{
			Name:      "new-component1",
			Namespace: "kyma-system",
			Source: &runtime.ComponentSource{
				URL: "https://local.test/kyma-additional-components/new-component1.tgz"},
		}
		unexpectedAdditionalComponent2 := runtime.KymaComponent{
			Name:      "test-component3",
			Namespace: "kyma-system",
			Source: &runtime.ComponentSource{
				URL: "https://test.local/test-component3.tgz"},
		}
		unexpectedAdditionalComponent3 := runtime.KymaComponent{
			Name:      "test-component4",
			Namespace: "kyma-system",
			Source: &runtime.ComponentSource{
				URL: "https://test.local/test-component4.tgz"},
		}

		// when
		allComponents, err := componentsProvider.AllComponents(internal.RuntimeVersionData{
			Version:      kymaVersion,
			Origin:       internal.Parameters,
			MajorVersion: 2,
		}, broker.AzurePlanName)

		// then
		require.NoError(t, err)
		assert.NotEmpty(t, allComponents)
		assert.Contains(t, allComponents, expectedPrerequisiteComponent)
		assert.Contains(t, allComponents, expectedRequiredComponent)
		assert.Contains(t, allComponents, expectedAdditionalComponent1)
		assert.Contains(t, allComponents, expectedAdditionalComponent2)
		assert.NotContains(t, allComponents, unexpectedAdditionalComponent1)
		assert.NotContains(t, allComponents, unexpectedAdditionalComponent2)
		assert.NotContains(t, allComponents, unexpectedAdditionalComponent3)
	})
}

func TestComponentsProviderErrors(t *testing.T) {
	t.Run("should return unsupported Kyma version error", func(t *testing.T) {
		// given
		ctx := context.TODO()
		k8sClient := fake.NewClientBuilder().Build()
		yamlPath := path.Join("testdata", additionalComponentsYaml)
		componentsProvider := runtime.NewFakeComponentsProvider(ctx, k8sClient, yamlPath)
		expectedErr := throwError("unsupported Kyma version")

		// when
		_, err := componentsProvider.AllComponents(internal.RuntimeVersionData{
			Version:      "1.1.0",
			Origin:       internal.Parameters,
			MajorVersion: 1,
		}, broker.AzurePlanName)

		// then
		require.Error(t, err)
		assert.Equal(t, expectedErr, errors.Unwrap(err))
	})

	t.Run("should return 404 Not Found when required components are not available for given Kyma version",
		func(t *testing.T) {
			// given
			ctx := context.TODO()
			k8sClient := fake.NewClientBuilder().Build()
			yamlPath := path.Join("testdata", additionalComponentsYaml)
			componentsProvider := runtime.NewFakeComponentsProvider(ctx, k8sClient, yamlPath)

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
			}, broker.AzurePlanName)

			// then
			require.Error(t, err)
			assert.Equal(t, expectedErr, errors.Unwrap(err))
		})
}

func fixK8sResources() []k8sruntime.Object {
	var resources []k8sruntime.Object
	type additionalComponentData struct {
		name, namespace, sourceURL, planForLabel, versionForLabel string
	}
	additionalComponentsData := []additionalComponentData{
		additionalComponentData{
			name:            "test-component1",
			namespace:       "kyma-system",
			sourceURL:       "",
			planForLabel:    broker.AzurePlanName,
			versionForLabel: kymaVersion,
		},
		additionalComponentData{
			name:            "test-component2",
			namespace:       "compass-system",
			sourceURL:       "https://test.local/test-component2.tgz",
			planForLabel:    broker.AzurePlanName,
			versionForLabel: kymaVersion,
		},
		additionalComponentData{
			name:            "test-component3",
			namespace:       "kyma-system",
			sourceURL:       "https://test.local/test-component3.tgz",
			planForLabel:    broker.GCPPlanName,
			versionForLabel: kymaVersion,
		},
		additionalComponentData{
			name:            "test-component4",
			namespace:       "kyma-system",
			sourceURL:       "https://test.local/test-component4.tgz",
			planForLabel:    broker.AzurePlanName,
			versionForLabel: "2.1.0",
		},
	}
	for _, cmp := range additionalComponentsData {
		configMap := &coreV1.ConfigMap{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      fmt.Sprintf("additional-component-%s", cmp.name),
				Namespace: "kcp-system",
				Labels: map[string]string{
					fmt.Sprintf("add-cmp-plan-%s", cmp.planForLabel):       "true",
					fmt.Sprintf("add-cmp-version-%s", cmp.versionForLabel): "true",
				},
			},
			Data: map[string]string{
				"component.name":      cmp.name,
				"component.namespace": cmp.namespace,
				"component.source":    cmp.sourceURL,
			},
		}
		resources = append(resources, configMap)
	}

	return resources
}

func throwError(message string) error {
	return errors.New(message)
}
