package gardener

import (
	gclientset "github.com/gardener/gardener/pkg/client/core/clientset/versioned"
	ginformers "github.com/gardener/gardener/pkg/client/core/informers/externalversions"
	shootsinformer "github.com/gardener/gardener/pkg/client/core/informers/externalversions/core/v1beta1"

	"github.com/kyma-project/control-plane/components/metris/internal/log"

	kubeinformers "k8s.io/client-go/informers"
	secretsinformer "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
)

// Client holds the kubernetes and gardener clientset configuration.
type Client struct {
	// Namespace holds the gardener project namespace used by the client.
	Namespace string
	// GClientset holds the gardener clientset.
	GClientset gclientset.Interface
	// KClientset holds the kubernetes clientset.
	KClientset kubernetes.Interface
}

// Controller represent the controller configuration needed to watch for shoots and secrets.
type Controller struct {
	// providertype is the name of the infrastructure provider.
	providertype string
	// client is the gardener client that holds the clientsets for gardener and kubernetes.
	client *Client
	// gardenerInformerFactory
	gardenerInformerFactory ginformers.SharedInformerFactory
	// kubeInformerFactory
	kubeInformerFactory kubeinformers.SharedInformerFactory
	// shootInformer defines the informer and lister for Shoots.
	shootInformer shootsinformer.ShootInformer
	// secretInformer defines the informer and lister for Secrets.
	secretInformer secretsinformer.SecretInformer
	// clusterChannel define the channel to exchange cluster definitions with provider.
	clusterChannel chan<- *Cluster
	// logger is the standard logger for the controller.
	logger log.Logger
}

// Cluster is a representation of a SKR cluster.
type Cluster struct {
	// ProviderType is the name of the infrastructure provider.
	ProviderType string
	// Region is the region name where the Shoot is created.
	Region string
	// TechnicalID is the technical id of the Shoot that is use for tagging all the resources created in the infrastructure.
	TechnicalID string
	// CredentialData is the provider secret data.
	CredentialData map[string][]byte
	// AccountID is the SCP account id link to the cluster.
	AccountID string
	// SubAccountID is the SCP subaccount id link to the cluster.
	SubAccountID string
	// Deleted is a flag that mark the cluster has being destroyed.
	Deleted bool
	// Trial is a flag that tell if the cluster is a trial one or not
	Trial bool
}
