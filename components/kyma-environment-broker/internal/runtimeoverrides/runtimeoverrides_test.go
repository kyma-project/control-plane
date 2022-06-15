package runtimeoverrides

import (
	"context"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtimeoverrides/automock"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRuntimeOverrides_Append(t *testing.T) {
	t.Run("Success when there is ConfigMap with overrides for given planID and Kyma version", func(t *testing.T) {
		// GIVEN
		cm := &coreV1.ConfigMap{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      "overrides",
				Namespace: namespace,
				Labels: map[string]string{
					"overrides-version-1.15.1": "true",
					"overrides-plan-foo":       "true",
				},
			},
			Data: map[string]string{"test1": "test1abc"},
		}

		cm2 := &coreV1.ConfigMap{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      "overrides2",
				Namespace: namespace,
				Labels: map[string]string{
					"overrides-version-1.15.1": "true",
					"overrides-account-1234":   "true",
				},
			},
			Data: map[string]string{"test2": "test2abc"},
		}
		sch := runtime.NewScheme()
		require.NoError(t, coreV1.AddToScheme(sch))
		client := fake.NewFakeClientWithScheme(sch, cm, cm2)

		inputAppenderMock := &automock.InputAppender{}
		defer inputAppenderMock.AssertExpectations(t)
		inputAppenderMock.On("AppendGlobalOverrides", []*gqlschema.ConfigEntryInput{
			{Key: "test1", Value: "test1abc"},
			{Key: "test2", Value: "test2abc"},
		}).Return(nil).Once()
		runtimeOverrides := NewRuntimeOverrides(context.TODO(), client)

		// WHEN
		err := runtimeOverrides.Append(inputAppenderMock, "foo", "1.15.1", "1234", "5678")

		// THEN
		require.NoError(t, err)
	})

	t.Run("Success when there are ConfigMap and Secret with overrides for given planID and Kyma version", func(t *testing.T) {
		// GIVEN
		cm := &coreV1.ConfigMap{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      "overrides",
				Namespace: namespace,
				Labels: map[string]string{
					"overrides-version-1.15.1": "true",
					"overrides-plan-foo":       "true",
				},
			},
			Data: map[string]string{"test1": "test1abc"},
		}
		secret := &coreV1.Secret{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      "overrides-secret",
				Namespace: namespace,
				Labels: map[string]string{
					"runtime-override": "true",
				},
			},
			Data: map[string][]byte{"test2": []byte("test2abc")},
		}
		sch := runtime.NewScheme()
		require.NoError(t, coreV1.AddToScheme(sch))
		client := fake.NewFakeClientWithScheme(sch, cm, secret)

		inputAppenderMock := &automock.InputAppender{}
		defer inputAppenderMock.AssertExpectations(t)
		inputAppenderMock.On("AppendGlobalOverrides", []*gqlschema.ConfigEntryInput{
			{
				Key:   "test1",
				Value: "test1abc",
			},
		}).Return(nil).Once()
		inputAppenderMock.On("AppendGlobalOverrides", []*gqlschema.ConfigEntryInput{
			{
				Key:    "test2",
				Value:  "test2abc",
				Secret: ptr.Bool(true),
			},
		}).Return(nil).Once()

		runtimeOverrides := NewRuntimeOverrides(context.TODO(), client)

		// WHEN
		err := runtimeOverrides.Append(inputAppenderMock, "foo", "1.15.1", "1234", "5678")

		// THEN
		require.NoError(t, err)
	})

	t.Run("Success when there is multiple ConfigMaps and Secrets, some with overrides - global and component scoped", func(t *testing.T) {
		// GIVEN
		sch := runtime.NewScheme()
		require.NoError(t, coreV1.AddToScheme(sch))
		client := fake.NewFakeClientWithScheme(sch, fixResources()...)

		inputAppenderMock := &automock.InputAppender{}
		defer inputAppenderMock.AssertExpectations(t)
		inputAppenderMock.On("AppendOverrides", "core", []*gqlschema.ConfigEntryInput{
			{
				Key:    "test1",
				Value:  "test1abc",
				Secret: ptr.Bool(true),
			},
		}).Return(nil).Once()
		inputAppenderMock.On("AppendOverrides", "helm", []*gqlschema.ConfigEntryInput{
			{
				Key:    "test3",
				Value:  "test3abc",
				Secret: ptr.Bool(true),
			},
		}).Return(nil).Once()
		inputAppenderMock.On("AppendGlobalOverrides", []*gqlschema.ConfigEntryInput{
			{
				Key:    "test4",
				Value:  "test4abc",
				Secret: ptr.Bool(true),
			},
		}).Return(nil).Once()
		inputAppenderMock.On("AppendOverrides", "core", []*gqlschema.ConfigEntryInput{
			{
				Key:   "test5",
				Value: "test5abc",
			},
		}).Return(nil).Once()
		inputAppenderMock.On("AppendGlobalOverrides", []*gqlschema.ConfigEntryInput{
			{Key: "test7", Value: "test7abc"},
		}).Return(nil).Once()

		runtimeOverrides := NewRuntimeOverrides(context.TODO(), client)

		// WHEN
		err := runtimeOverrides.Append(inputAppenderMock, "foo", "1.15.1", "1234", "5678")

		// THEN
		require.NoError(t, err)
	})

	t.Run("Error when there is no ConfigMap with overrides present", func(t *testing.T) {
		// GIVEN
		sch := runtime.NewScheme()
		require.NoError(t, coreV1.AddToScheme(sch))
		client := fake.NewFakeClientWithScheme(sch)

		inputAppenderMock := &automock.InputAppender{}
		defer inputAppenderMock.AssertExpectations(t)

		runtimeOverrides := NewRuntimeOverrides(context.TODO(), client)

		// WHEN
		err := runtimeOverrides.Append(inputAppenderMock, "foo", "1.15.1", "1234", "5678")

		// THEN
		require.Error(t, err, "no global overrides for plan 'foo' and Kyma version '1.15.1'")
	})
}

func fixResources() []runtime.Object {
	var resources []runtime.Object

	resources = append(resources, &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "secret#1",
			Namespace: namespace,
			Labels: map[string]string{
				"runtime-override": "true",
				"component":        "core",
			},
		},
		Data: map[string][]byte{"test1": []byte("test1abc")},
	})
	resources = append(resources, &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "secret#2",
			Namespace: namespace,
			Labels: map[string]string{
				"component": "core",
			},
		},
		Data: map[string][]byte{"test2": []byte("test2abc")},
	})
	resources = append(resources, &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "secret#3",
			Namespace: namespace,
			Labels: map[string]string{
				"runtime-override": "true",
				"component":        "helm",
			},
		},
		Data: map[string][]byte{"test3": []byte("test3abc")},
	})
	resources = append(resources, &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "secret#4",
			Namespace: namespace,
			Labels: map[string]string{
				"runtime-override": "true",
			},
		},
		Data: map[string][]byte{"test4": []byte("test4abc")},
	})
	resources = append(resources, &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "configmap#1",
			Namespace: namespace,
			Labels: map[string]string{
				"overrides-version-1.15.1": "true",
				"overrides-plan-foo":       "true",
				"component":                "core",
			},
		},
		Data: map[string]string{"test5": "test5abc"},
	})
	resources = append(resources, &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "configmap#2",
			Namespace: "default",
			Labels: map[string]string{
				"overrides-version-1.15.1": "true",
				"overrides-plan-foo":       "true",
				"overrides-plan-lite":      "true",
				"component":                "helm",
			},
		},
		Data: map[string]string{"test6": "test6abc"},
	})
	resources = append(resources, &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "configmap#3",
			Namespace: namespace,
			Labels: map[string]string{
				"overrides-version-1.15.1": "true",
				"overrides-plan-foo":       "true",
			},
		},
		Data: map[string]string{"test7": "test7abc"},
	})
	resources = append(resources, &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "configmap#4",
			Namespace: namespace,
			Labels: map[string]string{
				"overrides-version-1.15.0": "true",
				"overrides-plan-foo":       "true",
				"overrides-plan-lite":      "true",
			},
		},
		Data: map[string]string{"test8": "test8abc"},
	})
	return resources
}
