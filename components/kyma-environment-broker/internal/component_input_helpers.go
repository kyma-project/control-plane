package internal

import (
	"context"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	reconcilerApi "github.com/kyma-incubator/reconciler/pkg/keb"
)

const (
	SCMigrationComponentName          = "sc-migration"
	BTPOperatorComponentName          = "btp-operator"
	HelmBrokerComponentName           = "helm-broker"
	ServiceCatalogComponentName       = "service-catalog"
	ServiceCatalogAddonsComponentName = "service-catalog-addons"
	ServiceManagerComponentName       = "service-manager-proxy"
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

func getBTPOperatorProvisioningOverrides(creds *ServiceManagerOperatorCredentials) []*gqlschema.ConfigEntryInput {
	return []*gqlschema.ConfigEntryInput{
		{
			Key:    "manager.secret.clientid",
			Value:  creds.ClientID,
			Secret: ptr.Bool(true),
		},
		{
			Key:    "manager.secret.clientsecret",
			Value:  creds.ClientSecret,
			Secret: ptr.Bool(true),
		},
		{
			Key:   "manager.secret.url",
			Value: creds.ServiceManagerURL,
		},
		{
			Key:   "manager.secret.tokenurl",
			Value: creds.URL,
		},
	}
}

func getBTPOperatorUpdateOverrides(creds *ServiceManagerOperatorCredentials, clusterId string) []*gqlschema.ConfigEntryInput {
	return []*gqlschema.ConfigEntryInput{
		{
			Key:   "cluster.id",
			Value: clusterId,
		},
	}
}

func GetBTPOperatorReconcilerOverrides(creds *ServiceManagerOperatorCredentials, clusterIdGetter ClusterIDGetter) ([]reconcilerApi.Configuration, error) {
	id, err := clusterIdGetter()
	if err != nil {
		return nil, err
	}
	provisioning := getBTPOperatorProvisioningOverrides(creds)
	update := getBTPOperatorUpdateOverrides(creds, id)
	all := append(provisioning, update...)
	var config []reconcilerApi.Configuration
	for _, c := range all {
		secret := false
		if c.Secret != nil {
			secret = *c.Secret
		}
		rc := reconcilerApi.Configuration{Key: c.Key, Value: c.Value, Secret: secret}
		config = append(config, rc)
	}
	return config, nil
}

func CreateBTPOperatorProvisionInput(r ProvisionerInputCreator, creds *ServiceManagerOperatorCredentials) {
	overrides := getBTPOperatorProvisioningOverrides(creds)
	r.AppendOverrides(BTPOperatorComponentName, overrides)
}

func CreateBTPOperatorUpdateInput(r ProvisionerInputCreator, creds *ServiceManagerOperatorCredentials, clusterIdGetter ClusterIDGetter) error {
	id, err := clusterIdGetter()
	if err != nil {
		return err
	}
	provisioning := getBTPOperatorProvisioningOverrides(creds)
	update := getBTPOperatorUpdateOverrides(creds, id)
	r.AppendOverrides(BTPOperatorComponentName, provisioning)
	r.AppendOverrides(BTPOperatorComponentName, update)
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
		if err != nil {
			return "", err
		}
		return cm.Data["id"], nil
	}
}
