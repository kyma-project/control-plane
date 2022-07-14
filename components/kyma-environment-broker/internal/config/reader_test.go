package config_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	kebConfigYaml             = "keb-config.yaml"
	namespace                 = "kcp-system"
	runtimeVersionLabelPrefix = "runtime-version-"
	kebConfigLabel            = "keb-config"
	kymaVersion               = "2.4.0"
	defaultConfigKey          = "default"
)

func TestConfigReaderSuccessFlow(t *testing.T) {
	// setup
	ctx := context.TODO()
	cfgMap, err := fixConfigMap()
	require.NoError(t, err)

	fakeK8sClient := fake.NewClientBuilder().WithRuntimeObjects(cfgMap).Build()
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	cfgReader := config.NewConfigMapReader(ctx, fakeK8sClient, logger)

	t.Run("should read default KEB config for Kyma version 2.4.0", func(t *testing.T) {
		// when
		rawCfg, err := cfgReader.Read(kymaVersion, broker.AWSPlanName)

		// then
		require.NoError(t, err)
		assert.Equal(t, cfgMap.Data[defaultConfigKey], rawCfg)
	})

	t.Run("should read KEB config for Kyma version 2.4.0 and azure plan", func(t *testing.T) {
		// when
		rawCfg, err := cfgReader.Read(kymaVersion, broker.AzurePlanName)

		// then
		require.NoError(t, err)
		assert.Equal(t, cfgMap.Data[broker.AzurePlanName], rawCfg)
	})

	t.Run("should read KEB config for Kyma version 2.4.0 and trial plan", func(t *testing.T) {
		// when
		rawCfg, err := cfgReader.Read(kymaVersion, broker.TrialPlanName)

		// then
		require.NoError(t, err)
		assert.Equal(t, cfgMap.Data[broker.TrialPlanName], rawCfg)
	})
}

func TestConfigReaderErrors(t *testing.T) {
	// setup
	ctx := context.TODO()
	redundantCfgMap := &coreV1.ConfigMap{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "redundant-configmap",
			Namespace: namespace,
			Labels: map[string]string{
				fmt.Sprintf("%s%s", runtimeVersionLabelPrefix, kymaVersion): "true",
				kebConfigLabel: "true",
			},
		},
	}
	cfgMap, err := fixConfigMap()
	require.NoError(t, err)

	mockK8sClient := mockK8sClient{}
	fakeK8sClient := fake.NewClientBuilder().Build()
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	t.Run("should return error while fetching configmap on List() of K8s client", func(t *testing.T) {
		// given
		cfgReader := config.NewConfigMapReader(ctx, mockK8sClient, logger)

		// when
		rawCfg, err := cfgReader.Read(kymaVersion, broker.AzurePlanName)

		// then
		require.Error(t, err)
		logger.Error(err)
		assert.Equal(t, "", rawCfg)
	})

	t.Run("should return error while verifying configuration configmap existence", func(t *testing.T) {
		// given
		cfgReader := config.NewConfigMapReader(ctx, fakeK8sClient, logger)

		// when
		rawCfg, err := cfgReader.Read(kymaVersion, broker.AzurePlanName)

		// then
		require.Error(t, err)
		logger.Error(err)
		assert.Equal(t, "", rawCfg)

		// given
		err = fakeK8sClient.Create(ctx, cfgMap)
		require.NoError(t, err)

		err = fakeK8sClient.Create(ctx, redundantCfgMap)
		require.NoError(t, err)

		// when
		rawCfg, err = cfgReader.Read(kymaVersion, broker.AzurePlanName)

		// then
		require.Error(t, err)
		logger.Error(err)
		assert.Equal(t, "", rawCfg)
	})

	t.Run("should return error while getting config string for a plan", func(t *testing.T) {
		// given
		err = fakeK8sClient.Delete(ctx, cfgMap)
		require.NoError(t, err)

		cfgReader := config.NewConfigMapReader(ctx, fakeK8sClient, logger)

		// when
		rawCfg, err := cfgReader.Read(kymaVersion, broker.AzurePlanName)

		// then
		require.Error(t, err)
		logger.Error(err)
		assert.Equal(t, "", rawCfg)
	})
}

func fixConfigMap() (*coreV1.ConfigMap, error) {
	yamlFilePath := path.Join("testdata", kebConfigYaml)
	contents, err := os.ReadFile(yamlFilePath)
	if err != nil {
		return nil, fmt.Errorf("while reading configmap")
	}

	var tempCfgMap tempConfigMap
	err = yaml.Unmarshal(contents, &tempCfgMap)
	if err != nil {
		return nil, fmt.Errorf("while unmarshalling configmap")
	}

	return tempCfgMap.toConfigMap(), nil
}

type tempConfigMap struct {
	APIVersion string            `yaml:"apiVersion,omitempty"`
	Kind       string            `yaml:"kind,omitempty"`
	Metadata   tempMetadata      `yaml:"metadata,omitempty"`
	Data       map[string]string `yaml:"data,omitempty"`
}

type tempMetadata struct {
	Name      string            `yaml:"name,omitempty"`
	Namespace string            `yaml:"namespace,omitempty"`
	Labels    map[string]string `yaml:"labels,omitempty"`
}

func (m *tempConfigMap) toConfigMap() *coreV1.ConfigMap {
	return &coreV1.ConfigMap{
		TypeMeta: metaV1.TypeMeta{
			Kind:       m.Kind,
			APIVersion: m.APIVersion,
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      m.Metadata.Name,
			Namespace: m.Metadata.Namespace,
			Labels:    m.Metadata.Labels,
		},
		Data: m.Data,
	}
}

type mockK8sClient struct{}

func (m mockK8sClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	return fmt.Errorf("not implemented")
}

func (m mockK8sClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return fmt.Errorf("not implemented")
}

func (m mockK8sClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	return fmt.Errorf("not implemented")
}

func (m mockK8sClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return fmt.Errorf("not implemented")
}

func (m mockK8sClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return fmt.Errorf("not implemented")
}

func (m mockK8sClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return fmt.Errorf("not implemented")
}

func (m mockK8sClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return fmt.Errorf("not implemented")
}

func (m mockK8sClient) Status() client.StatusWriter {
	panic("not implemented")
}

func (m mockK8sClient) Scheme() *runtime.Scheme {
	panic("not implemented")
}

func (m mockK8sClient) RESTMapper() meta.RESTMapper {
	panic("not implemented")
}
