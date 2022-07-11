package config_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/config"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const kebConfigYaml = "keb-config.yaml"

func TestConfigReaderSuccessFlow(t *testing.T) {
	t.Run("should read KEB config for 2.4.0 runtime version", func(t *testing.T) {
		// given
		ctx := context.TODO()
		fakeK8sClientBuilder := fake.NewClientBuilder()
		cfgMapObj, err := fixConfigMap()
		if err != nil {
			t.Fatal("error while creating configmap from yaml")
		}
		fakeK8sClient := fakeK8sClientBuilder.WithRuntimeObjects(cfgMapObj).Build()
		logger := logrus.New()
		logger.SetFormatter(&logrus.JSONFormatter{})
		_ = config.NewConfigReader(ctx, fakeK8sClient, logger)
	})
}

func fixConfigMap() (runtime.Object, error) {
	yamlFilePath := path.Join("testdata", kebConfigYaml)
	contents, err := os.ReadFile(yamlFilePath)
	if err != nil {
		return nil, fmt.Errorf("while reading configuration yaml")
	}
	cfgMap := &coreV1.ConfigMap{}
	err = yaml.Unmarshal(contents, cfgMap)
	if err != nil {
		return nil, fmt.Errorf("while unmarshalling configuration yaml")
	}
	return cfgMap, nil
}
