package config

import (
	"context"
	"fmt"
	"strings"

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

type ConfigMapReader struct {
	ctx       context.Context
	k8sClient client.Client
	logger    logrus.FieldLogger
}

func NewConfigMapReader(ctx context.Context, k8sClient client.Client, logger logrus.FieldLogger) *ConfigMapReader {
	return &ConfigMapReader{
		ctx:       ctx,
		k8sClient: k8sClient,
		logger:    logger,
	}
}

func (r *ConfigMapReader) Read(kymaVersion, planName string) (string, error) {
	r.logger.Infof("getting configuration for Kyma version %v and %v plan", kymaVersion, planName)
	cfgMapList, err := r.getConfigMapList(kymaVersion)
	if err != nil {
		return "", err
	}

	if err = r.verifyConfigMapExistence(cfgMapList); err != nil {
		return "", fmt.Errorf("while verifying configuration configmap existence: %w", err)
	}

	cfgMap := cfgMapList.Items[0]
	cfgString, err := r.getConfigStringForPlanOrDefaults(&cfgMap, planName)
	if err != nil {
		return "", fmt.Errorf("while getting configuration string: %w", err)
	}

	return cfgString, nil
}

func (r *ConfigMapReader) getConfigMapList(kymaVersion string) (*coreV1.ConfigMapList, error) {
	cfgMapList := &coreV1.ConfigMapList{}
	listOptions := configMapListOptions(kymaVersion)
	if err := r.k8sClient.List(r.ctx, cfgMapList, listOptions...); err != nil {
		return nil, fmt.Errorf("while fetching configmap with configuration for Kyma version %v: %w",
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

func (r *ConfigMapReader) verifyConfigMapExistence(cfgMapList *coreV1.ConfigMapList) error {
	switch n := len(cfgMapList.Items); n {
	case 1:
		return nil
	case 0:
		return fmt.Errorf("configmap with configuration does not exist")
	default:
		return fmt.Errorf("allowed number of configuration configmaps: 1, found: %d", n)
	}
}

func (r *ConfigMapReader) getConfigStringForPlanOrDefaults(cfgMap *coreV1.ConfigMap, planName string) (string, error) {
	cfgString, exists := cfgMap.Data[planName]
	if !exists {
		r.logger.Infof("configuration for plan %v does not exist. Using default values", planName)
		cfgString, exists = cfgMap.Data[defaultConfigKey]
		if !exists {
			return "", fmt.Errorf("default configuration does not exist")
		}
	}
	return cfgString, nil
}
