package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/runtime"
	"github.com/sirupsen/logrus"
	coreV1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	namespace                 = "kcp-system"
	runtimeVersionLabelPrefix = "runtime-version-"
	kebConfigLabel            = "keb-config"
	defaultConfigKey          = "default"
)

type ConfigReader struct {
	ctx       context.Context
	k8sClient client.Client
	logger    logrus.FieldLogger
}

type ConfigForPlan struct {
	AdditionalComponents []runtime.KymaComponent `json:"additional-components"`
}

func NewConfigReader(ctx context.Context, k8sClient client.Client, logger logrus.FieldLogger) *ConfigReader {
	return &ConfigReader{
		ctx:       ctx,
		k8sClient: k8sClient,
		logger:    logger,
	}
}

func (r *ConfigReader) ReadConfig(kymaVersion, planName string) (string, error) {
	cfgMapList, err := r.getConfigMapList(kymaVersion)
	if err != nil {
		return "", err
	}

	if err = r.verifyConfigMapExistence(cfgMapList); err != nil {
		return "", fmt.Errorf("while verifying configuration configmap existence for Kyma version %v: %w",
			kymaVersion, err)
	}
	r.logger.Infof("found configmap with configuration for Kyma version: %v. Checking plan existence...",
		kymaVersion)

	cfgMap := cfgMapList.Items[0]
	cfgString, err := r.getRawConfigForPlanOrDefaults(&cfgMap, planName)
	if err != nil {
		return "", fmt.Errorf("while getting configuration for Kyma version %v and plan %v"+
			": %w", kymaVersion, planName, err)
	}

	return cfgString, nil
}

func (r *ConfigReader) getConfigMapList(kymaVersion string) (*coreV1.ConfigMapList, error) {
	cfgMapList := &coreV1.ConfigMapList{}
	listOptions := configMapListOptions(kymaVersion)
	if err := r.k8sClient.List(r.ctx, cfgMapList, listOptions...); err != nil {
		return nil, fmt.Errorf("while fetching configmaps with configuration for Kyma version %v: %w",
			kymaVersion, err)
	}
	return cfgMapList, nil
}

func configMapListOptions(version string) []client.ListOption {
	versionLabel := runtimeVersionLabelPrefix + strings.ToLower(version)

	labels := map[string]string{
		versionLabel:   "true",
		kebConfigLabel: "true",
	}

	return []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(labels),
	}
}

func (r *ConfigReader) verifyConfigMapExistence(cfgMapList *coreV1.ConfigMapList) error {
	switch n := len(cfgMapList.Items); n {
	case 1:
		return nil
	case 0:
		return fmt.Errorf("configmap with configuration does not exist")
	default:
		return fmt.Errorf("allowed number of configuration configmaps: 1, found: %d", n)
	}
}

func (r *ConfigReader) getRawConfigForPlanOrDefaults(cfgMap *coreV1.ConfigMap, planName string) (string, error) {
	cfgString, exists := cfgMap.Data[planName]
	if !exists {
		r.logger.Infof("configuration for plan %v does not exist. Using default values", planName)
		cfgString, exists = cfgMap.Data[defaultConfigKey]
		if !exists {
			return "", fmt.Errorf("default configuration does not exist")
		}
	}
	r.logger.Infof("found configuration for plan %v", planName)
	return cfgString, nil
}
