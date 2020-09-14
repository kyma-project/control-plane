package gardener

import (
	"fmt"
	"strings"
	"time"

	"github.com/kyma-project/control-plane/components/metris/internal/log"

	ginformers "github.com/gardener/gardener/pkg/client/core/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

const (
	labelAccountID       = "account"
	labelSubAccountID    = "subaccount"
	labelHyperscalerType = "hyperscalerType"

	fieldSecretBindingName = "spec.secretBindingName"
	fieldCloudProfileName  = "spec.cloudProfileName"

	defaultResyncPeriod = time.Second * 30
)

// NewController return a new controller for watching shoots and secrets.
func NewController(client *Client, provider string, clusterChannel chan<- *Cluster, logger log.Logger) (*Controller, error) {
	gardenerInformerFactory := ginformers.NewSharedInformerFactoryWithOptions(
		client.GClientset,
		defaultResyncPeriod,
		ginformers.WithNamespace(client.Namespace),
		ginformers.WithTweakListOptions(func(opts *metav1.ListOptions) {
			opts.FieldSelector = fields.SelectorFromSet(fields.Set{fieldCloudProfileName: provider}).String()
		}),
	)

	hyperscalertype := provider
	if hyperscalertype == "az" {
		hyperscalertype = "azure"
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(
		client.KClientset,
		defaultResyncPeriod,
		kubeinformers.WithNamespace(client.Namespace),
		kubeinformers.WithTweakListOptions(func(opts *metav1.ListOptions) {
			opts.LabelSelector = labels.SelectorFromSet(labels.Set{labelHyperscalerType: hyperscalertype}).String()
		}),
	)

	shootInformer := gardenerInformerFactory.Core().V1beta1().Shoots()
	secretInformer := kubeInformerFactory.Core().V1().Secrets()

	controller := &Controller{
		providertype:            strings.ToLower(provider),
		client:                  client,
		gardenerInformerFactory: gardenerInformerFactory,
		kubeInformerFactory:     kubeInformerFactory,
		shootInformer:           shootInformer,
		secretInformer:          secretInformer,
		clusterChannel:          clusterChannel,
		logger:                  logger,
	}

	// Set up event handlers for Shoot resources
	shootInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.shootAddHandlerFunc,
		UpdateFunc: controller.shootUpdateHandlerFunc,
		DeleteFunc: controller.shootDeleteHandlerFunc,
	})

	// Set up event handler for Secret resources
	secretInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: controller.secretUpdateHandlerFunc,
	})

	return controller, nil
}

// Run will set up the event handlers for secrets and shoots, as well as syncing informer caches.
func (c *Controller) Run(stop <-chan struct{}) error {
	c.logger.Info("controller started")
	defer c.logger.Info("controller stopped")

	// Start the informer factories to begin populating the informer caches
	c.gardenerInformerFactory.Start(stop)
	c.kubeInformerFactory.Start(stop)

	c.logger.Debug("waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(stop, c.shootInformer.Informer().HasSynced, c.secretInformer.Informer().HasSynced); !ok {
		return fmt.Errorf("error waiting for cache to sync")
	}

	c.logger.Debug("informer caches sync completed")

	// wait for stop signal from the workgroup
	<-stop

	return nil
}
