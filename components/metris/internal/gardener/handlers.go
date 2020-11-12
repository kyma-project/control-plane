package gardener

import (
	"bytes"
	"context"
	"fmt"

	gcorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	commonpkg "github.com/gardener/gardener/pkg/operation/common"
	shootpkg "github.com/gardener/gardener/pkg/operation/shoot"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

// secretUpdateHandlerFunc is notification function which handle secret updates.
func (c *Controller) secretUpdateHandlerFunc(oldObj, newObj interface{}) {
	newSecret, ok := newObj.(*corev1.Secret)
	if !ok {
		c.logger.Error("error decoding new secret, invalid type")
		return
	}

	oldSecret, ok := oldObj.(*corev1.Secret)
	if !ok {
		c.logger.Error("error decoding old secret, invalid type")
		return
	}

	if newSecret.ResourceVersion == oldSecret.ResourceVersion {
		// Periodic resync will send update events for all known Secrets, so if the
		// resource version did not change we skip
		return
	}

	// check all shoots with that secret and update
	fselector := fields.SelectorFromSet(
		fields.Set{
			fieldSecretBindingName: newSecret.Name,
			fieldCloudProfileName:  c.providertype,
		}).String()

	shootlist, err := c.client.GClientset.CoreV1beta1().Shoots(newSecret.Namespace).List(context.TODO(), metav1.ListOptions{FieldSelector: fselector})
	if err != nil {
		c.logger.With("error", err).Error("error retrieving shoot")
		return
	}

	if len(shootlist.Items) == 0 {
		c.logger.With("secret", newSecret.Name).Debugf("no shoots found with this secret binding")
		return
	}

	for i := range shootlist.Items {
		shoot := shootlist.Items[i]
		c.logger.With("shoot", shoot.Name).With("secret", newSecret.Name).Debug("received a shoot secret update event")
		c.shootAddHandlerFunc(&shoot)
	}
}

// shootDeleteHandlerFunc is notification function which handle deleted shoots.
func (c *Controller) shootDeleteHandlerFunc(obj interface{}) {
	var technicalid string

	shoot, ok := obj.(*gcorev1beta1.Shoot)
	if !ok {
		// try to recover deleted obj
		delobj, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			c.logger.Error("error trying to recover deleted object, invalid type")
			return
		}

		shoot, ok = delobj.Obj.(*gcorev1beta1.Shoot)
		if !ok {
			c.logger.Error("error trying to recover deleted shoot")
			return
		}

		technicalid = delobj.Key
	}

	cluster, err := c.newCluster(shoot)

	logger := c.logger.With("account", cluster.AccountID).With("subaccount", cluster.SubAccountID).With("shoot", shoot.Name)

	if cluster.TechnicalID == "" {
		// check if we can recover the key from the deleted cache object
		if technicalid == "" {
			logger.With("error", err).Error("received a shoot delete event, but could not found shoot in cache")
			return
		}

		cluster.TechnicalID = technicalid
	}

	cluster.Deleted = true

	logger.With("technicalid", cluster.TechnicalID).Debug("received a shoot delete event, removing it from the cache")

	c.clusterChannel <- cluster
}

// shootAddHandlerFunc is notification function which handle new shoots.
func (c *Controller) shootAddHandlerFunc(obj interface{}) {
	shoot, ok := obj.(*gcorev1beta1.Shoot)
	if !ok {
		c.logger.Errorf("error decoding shoot object")
		return
	}

	cluster, err := c.newCluster(shoot)

	logger := c.logger.With("account", cluster.AccountID).With("subaccount", cluster.SubAccountID).With("shoot", shoot.Name).With("technicalid", cluster.TechnicalID)

	if err != nil {
		logger.With("error", err).Error("received a shoot add event, but there was missing informations")
		return
	}

	logger.Debug("received a shoot add event")
	c.clusterChannel <- cluster
}

// shootUpdateHandlerFunc is notification function which handle shoot changes.
func (c *Controller) shootUpdateHandlerFunc(oldObj, newObj interface{}) {
	oldShoot, ok := oldObj.(*gcorev1beta1.Shoot)
	if !ok {
		c.logger.Error("error decoding old shoot, invalid type")
		return
	}

	newShoot, ok := newObj.(*gcorev1beta1.Shoot)
	if !ok {
		c.logger.Error("error decoding new shoot, invalid type")
		return
	}

	if newShoot.ResourceVersion == oldShoot.ResourceVersion {
		// Periodic resync will send update events for all known Shoot, so if the
		// resource version did not change we skip
		return
	}

	c1, err := c.newCluster(oldShoot)
	if err != nil {
		c.logger.With("account", c1.AccountID).With("subaccount", c1.SubAccountID).With("shoot", oldShoot.Name).With("technicalid", c1.TechnicalID).Error(err)
		return
	}

	c2, err := c.newCluster(newShoot)
	if err != nil {
		c.logger.With("account", c2.AccountID).With("subaccount", c2.SubAccountID).With("shoot", newShoot.Name).With("technicalid", c2.TechnicalID).Error(err)
		return
	}

	if clustersEqual(c1, c2) {
		return
	}

	logger := c.logger.With("account", c2.AccountID).With("subaccount", c2.SubAccountID).With("shoot", newShoot.Name).With("technicalid", c2.TechnicalID)
	logger.Debug("received a shoot update event")

	c.shootAddHandlerFunc(newShoot)
}

// getTechnicalID determines the technical id of a Shoot which is used for tagging resources created in the infrastructure.
func (c *Controller) getTechnicalID(shoot *gcorev1beta1.Shoot) (string, error) {
	ns, err := c.client.KClientset.CoreV1().Namespaces().Get(context.TODO(), shoot.Namespace, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	projectname := commonpkg.ProjectNameForNamespace(ns)

	return shootpkg.ComputeTechnicalID(projectname, shoot), nil
}

// newCluster creates a Cluster definition based on the shoot information.
func (c *Controller) newCluster(shoot *gcorev1beta1.Shoot) (*Cluster, error) {
	var (
		cluster = &Cluster{ProviderType: c.providertype}
		err     error
		ok      bool
	)

	cluster.Region = shoot.Spec.Region

	cluster.TechnicalID, err = c.getTechnicalID(shoot)
	if err != nil {
		err = fmt.Errorf("could not find technical id")

		clusterSyncErrorVec.WithLabelValues("technicalid").Inc()

		return cluster, err
	}

	cluster.AccountID, ok = shoot.GetLabels()[labelAccountID]
	if !ok || cluster.AccountID == "" {
		err = fmt.Errorf("could not find label '%s'", labelAccountID)

		clusterSyncErrorVec.WithLabelValues("accountid").Inc()

		return cluster, err
	}

	cluster.SubAccountID, ok = shoot.GetLabels()[labelSubAccountID]
	if !ok || cluster.SubAccountID == "" {
		err = fmt.Errorf("could not find label '%s'", labelSubAccountID)

		clusterSyncErrorVec.WithLabelValues("subaccountid").Inc()

		return cluster, err
	}

	secret, err := c.client.KClientset.CoreV1().Secrets(shoot.Namespace).Get(context.TODO(), shoot.Spec.SecretBindingName, metav1.GetOptions{})
	if err != nil {
		err = fmt.Errorf("error getting shoot secret: %s", err)

		clusterSyncErrorVec.WithLabelValues("secret").Inc()

		return cluster, err
	}

	cluster.CredentialData = secret.Data

	return cluster, nil
}

// clustersEqual return true if two Cluster are equal.
func clustersEqual(c1, c2 *Cluster) bool {
	if c1 == nil || c2 == nil {
		return c1 == c2
	}

	if c1.TechnicalID != c2.TechnicalID {
		return false
	}

	if c1.ProviderType != c2.ProviderType {
		return false
	}

	if c1.AccountID != c2.AccountID {
		return false
	}

	if c1.SubAccountID != c2.SubAccountID {
		return false
	}

	for k1, v1 := range c1.CredentialData {
		v2, ok := c2.CredentialData[k1]
		if !ok {
			return false
		}

		if !bytes.Equal(v1, v2) {
			return false
		}
	}

	return true
}
