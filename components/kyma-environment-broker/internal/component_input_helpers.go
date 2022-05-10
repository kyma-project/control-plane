package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	SCMigrationComponentName          = "sc-migration"
	BTPOperatorComponentName          = "btp-operator"
	HelmBrokerComponentName           = "helm-broker"
	ServiceCatalogComponentName       = "service-catalog"
	ServiceCatalogAddonsComponentName = "service-catalog-addons"
	ServiceManagerComponentName       = "service-manager-proxy"

	// BTP Operator overrides keys
	BTPOperatorClientID     = "manager.secret.clientid"
	BTPOperatorClientSecret = "manager.secret.clientsecret"
	BTPOperatorURL          = "manager.secret.url"    // deprecated, for btp-operator v0.2.0
	BTPOperatorSMURL        = "manager.secret.sm_url" // for btp-operator v0.2.3
	BTPOperatorTokenURL     = "manager.secret.tokenurl"
	BTPOperatorClusterID    = "cluster.id"
)

type ClusterIDGetter func() (string, error)

func DisableServiceManagementComponents(r ProvisionerInputCreator) {
	r.DisableOptionalComponent(SCMigrationComponentName)
	r.DisableOptionalComponent(HelmBrokerComponentName)
	r.DisableOptionalComponent(ServiceCatalogComponentName)
	r.DisableOptionalComponent(ServiceCatalogAddonsComponentName)
	r.DisableOptionalComponent(ServiceManagerComponentName)
	r.DisableOptionalComponent(BTPOperatorComponentName)
}

func getBTPOperatorProvisioningOverrides(creds *ServiceManagerOperatorCredentials, clusterId string) []*gqlschema.ConfigEntryInput {
	return []*gqlschema.ConfigEntryInput{
		{
			Key:    BTPOperatorClientID,
			Value:  creds.ClientID,
			Secret: ptr.Bool(true),
		},
		{
			Key:    BTPOperatorClientSecret,
			Value:  creds.ClientSecret,
			Secret: ptr.Bool(true),
		},
		{
			Key:   BTPOperatorURL,
			Value: creds.ServiceManagerURL,
		},
		{
			Key:   BTPOperatorSMURL,
			Value: creds.ServiceManagerURL,
		},
		{
			Key:   BTPOperatorTokenURL,
			Value: creds.URL,
		},
		{
			Key:   BTPOperatorClusterID,
			Value: clusterId,
		},
	}
}

func GetBTPOperatorReconcilerOverrides(creds *ServiceManagerOperatorCredentials, clusterIdGetter ClusterIDGetter) ([]reconcilerApi.Configuration, error) {
	id, err := clusterIdGetter()
	if err != nil {
		return nil, err
	}
	overrides := getBTPOperatorProvisioningOverrides(creds, id)
	var config []reconcilerApi.Configuration
	for _, c := range overrides {
		secret := false
		if c.Secret != nil {
			secret = *c.Secret
		}
		rc := reconcilerApi.Configuration{Key: c.Key, Value: c.Value, Secret: secret}
		config = append(config, rc)
	}
	return config, nil
}

func CreateBTPOperatorProvisionInput(r ProvisionerInputCreator, creds *ServiceManagerOperatorCredentials, clusterIdGetter ClusterIDGetter) error {
	id, err := clusterIdGetter()
	if err != nil {
		return err
	}
	overrides := getBTPOperatorProvisioningOverrides(creds, id)
	r.AppendOverrides(BTPOperatorComponentName, overrides)
	return nil
}

func GetClusterIDWithKubeconfig(kubeconfig string) ClusterIDGetter {
	return func() (string, error) {
		cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
		if err != nil {
			return "", err
		}
		cs, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return "", err
		}
		cm, err := cs.CoreV1().ConfigMaps("kyma-system").Get(context.Background(), "cluster-info", metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			cm = &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster-info",
					Namespace: "kyma-system",
				},
				Data: map[string]string{
					"id": uuid.NewString(),
				},
			}
			if cm, err = cs.CoreV1().ConfigMaps(cm.Namespace).Create(context.Background(), cm, metav1.CreateOptions{}); err != nil {
				return "", err
			}
			return cm.Data["id"], nil
		}
		if err != nil {
			return "", err
		}
		return cm.Data["id"], nil
	}
}

func CheckBTPCredsValid(clusterConfiguration reconcilerApi.Cluster) error {
	vals := make(map[string]bool)
	requiredKeys := []string{BTPOperatorClientID, BTPOperatorClientSecret, BTPOperatorURL, BTPOperatorSMURL, BTPOperatorTokenURL, BTPOperatorClusterID}
	hasBTPOperator := false
	var errs []string
	for _, c := range clusterConfiguration.KymaConfig.Components {
		if c.Component == BTPOperatorComponentName {
			hasBTPOperator = true
			for _, cfg := range c.Configuration {
				for _, key := range requiredKeys {
					if cfg.Key == key {
						vals[key] = true
						if cfg.Value == nil {
							errs = append(errs, fmt.Sprintf("missing required value for %v", key))
						}
						if val, ok := cfg.Value.(string); !ok || val == "" {
							errs = append(errs, fmt.Sprintf("missing required value for %v", key))
						}
					}
				}
			}
		}
	}
	if hasBTPOperator {
		for _, key := range requiredKeys {
			if !vals[key] {
				errs = append(errs, fmt.Sprintf("missing required key %v", key))
			}
		}
		if len(errs) != 0 {
			return fmt.Errorf("BTP Operator is about to be installed but is missing required configuration: %v", strings.Join(errs, ", "))
		}
	}
	return nil
}
