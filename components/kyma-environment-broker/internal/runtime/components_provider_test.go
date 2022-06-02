package runtime_test

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	kymaVersion              = "2.2.0"
	additionalComponentsYaml = "additional-runtime-components.yaml"
)

func TestComponentsProviderSuccessFlow(t *testing.T) {
	t.Run("should fetch required components and default additional components", func(t *testing.T) {
		// given
		ctx := context.TODO()
		k8sClient := fake.NewClientBuilder().Build()
		planNameHolder := runtime.GetPlanNameHolderInstance()
		yamlPath := path.Join("testdata", additionalComponentsYaml)
		componentsProvider := runtime.NewFakeComponentsProvider(ctx, k8sClient, planNameHolder, yamlPath)
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

		// when
		allComponents, err := componentsProvider.AllComponents(internal.RuntimeVersionData{
			Version:      kymaVersion,
			Origin:       internal.Parameters,
			MajorVersion: 2,
		})
		require.NoError(t, err)

		// then
		assert.NotEmpty(t, allComponents)
		assert.Contains(t, allComponents, expectedPrerequisiteComponent)
		assert.Contains(t, allComponents, expectedRequiredComponent)
		assert.Contains(t, allComponents, expectedAdditionalComponent)
	})
}

func fixK8sResources() []k8sruntime.Object {
	var resources []k8sruntime.Object
	type additionalComponentData struct {
		name, namespace, sourceURL string
	}
	additionalComponentsData := []additionalComponentData{
		additionalComponentData{
			name:      "test-component1",
			namespace: "kyma-system",
			sourceURL: "",
		},
		additionalComponentData{
			name:      "test-component2",
			namespace: "compass-system",
			sourceURL: "https://test.local/test-component2.tgz",
		},
	}
	for _, cmp := range additionalComponentsData {
		configMap := &coreV1.ConfigMap{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      fmt.Sprintf("additional-component-%s", cmp.name),
				Namespace: "kcp-system",
				Labels: map[string]string{
					"add-cmp-plan-azure":    "true",
					"add-cmp-version-2.2.0": "true",
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
