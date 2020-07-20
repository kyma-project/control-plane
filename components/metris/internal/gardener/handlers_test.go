package gardener

import (
	"testing"
	"time"

	gcorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/control-plane/components/metris/internal/log"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

var (
	defaultNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "test-ns",
			Labels: map[string]string{"project.gardener.cloud/name": "test-project"},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "core.gardener.cloud/v1beta1",
					Kind:       "Project",
					Name:       "test-project",
					UID:        "62857ea7-7d83-4e94-b4dc-d40784e307bd",
				},
			},
		},
	}

	defaultSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: defaultNamespace.Name,
			Labels:    map[string]string{labelHyperscalerType: "azure"},
		},
		Data: map[string][]byte{
			"clientID":       []byte("fakeclientidhere"),
			"clientSecret":   []byte("fakeclientsecrethere"),
			"subscriptionID": []byte("fakesubscriptionidhere"),
			"tenantID":       []byte("faketenantidhere"),
		},
	}

	defaultShoot = &gcorev1beta1.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-shoot",
			Namespace: defaultNamespace.Name,
			Labels: map[string]string{
				labelAccountID:    "accountid-test",
				labelSubAccountID: "subaccountid-test",
			},
		},
		Spec: gcorev1beta1.ShootSpec{
			CloudProfileName:  "az",
			SecretBindingName: "test-secret",
		},
	}

	clusterChannel = make(chan *Cluster, 1)
	ctrl           *Controller
)

func TestShootAddHandlerFunc(t *testing.T) {
	asserts := assert.New(t)

	testSetup(t)

	t.Run("adding a normal shoot", func(t *testing.T) {
		ctrl.shootAddHandlerFunc(defaultShoot)

		select {
		case cluster := <-clusterChannel:
			technicalID, err := ctrl.getTechnicalID(defaultShoot)
			asserts.NoErrorf(err, "error getting technicalid %s", err)
			asserts.Equal(technicalID, cluster.TechnicalID, "cluster should have technical id %s but got %s", technicalID, cluster.TechnicalID)

		case <-time.After(500 * time.Millisecond):
			asserts.Fail("timed out, did not get the cluster")
		}
	})

	t.Run("adding a shoot with missing label", func(t *testing.T) {
		newshoot := defaultShoot.DeepCopy()
		newshoot.ObjectMeta.Name = "new-test-shoot-account"
		delete(newshoot.ObjectMeta.Labels, labelAccountID)

		if _, err := ctrl.client.GClientset.CoreV1beta1().Shoots(defaultNamespace.Name).Create(newshoot); err != nil {
			asserts.Failf("error adding shoot", "%v", err)
		}

		ctrl.shootAddHandlerFunc(newshoot)

		select {
		case <-clusterChannel:
			asserts.Fail("should not have receive a new cluster")
		case <-time.After(500 * time.Millisecond):
		}
	})

	t.Run("adding a shoot in another namespace", func(t *testing.T) {
		newshoot := defaultShoot.DeepCopy()
		newshoot.ObjectMeta.Name = "test-shoot-2"
		newshoot.ObjectMeta.Namespace = "default"

		if _, err := ctrl.client.GClientset.CoreV1beta1().Shoots("default").Create(newshoot); err != nil {
			asserts.Failf("error adding shoot", "%v", err)
		}

		ctrl.shootAddHandlerFunc(newshoot)

		select {
		case <-clusterChannel:
			asserts.Fail("should not have receive a new cluster")
		case <-time.After(500 * time.Millisecond):
		}
	})

	testTeardown(t)
}

func TestShootUpdateHandlerFunc(t *testing.T) {
	asserts := assert.New(t)

	testSetup(t)

	t.Run("updating shoot with no change", func(t *testing.T) {
		ctrl.shootUpdateHandlerFunc(defaultShoot, defaultShoot)

		select {
		case <-clusterChannel:
			asserts.Fail("should not get cluster update")
		case <-time.After(500 * time.Millisecond):
		}
	})

	t.Run("updating shoot with no change but new version", func(t *testing.T) {
		newshoot := defaultShoot.DeepCopy()
		newshoot.ResourceVersion = "2"

		if _, err := ctrl.client.GClientset.CoreV1beta1().Shoots(defaultNamespace.Name).Update(newshoot); err != nil {
			asserts.Failf("error updating shoot", "%v", err)
		}

		ctrl.shootUpdateHandlerFunc(defaultShoot, newshoot)

		select {
		case <-clusterChannel:
			asserts.Fail("should not get cluster update")
		case <-time.After(500 * time.Millisecond):
		}
	})

	t.Run("updating shoot subaccountid", func(t *testing.T) {
		newshoot := defaultShoot.DeepCopy()
		newshoot.ResourceVersion = "3"
		newshoot.ObjectMeta.Labels[labelSubAccountID] = "new-subaccountid"

		if _, err := ctrl.client.GClientset.CoreV1beta1().Shoots(defaultNamespace.Name).Update(newshoot); err != nil {
			asserts.Failf("error updating shoot", "%v", err)
		}

		ctrl.shootUpdateHandlerFunc(defaultShoot, newshoot)

		select {
		case cluster := <-clusterChannel:
			asserts.Equal(newshoot.ObjectMeta.Labels[labelSubAccountID], cluster.SubAccountID, "cluster should have SubAccountID id %s but got %s", newshoot.ObjectMeta.Labels[labelSubAccountID], cluster.SubAccountID)
		case <-time.After(500 * time.Millisecond):
			asserts.Fail("should get cluster update")
		}
	})

	t.Run("updating shoot with bad new shoot", func(t *testing.T) {
		ctrl.shootUpdateHandlerFunc(defaultShoot, &Cluster{})

		select {
		case <-clusterChannel:
			asserts.Fail("should not get cluster update")
		case <-time.After(500 * time.Millisecond):
		}
	})

	testTeardown(t)
}

func TestShootDeleteHandlerFunc(t *testing.T) {
	asserts := assert.New(t)

	testSetup(t)

	t.Run("delete shoot normally", func(t *testing.T) {
		ctrl.shootDeleteHandlerFunc(defaultShoot)

		select {
		case cluster := <-clusterChannel:
			asserts.True(cluster.Deleted, "cluster should have parameter Deleted set to true")

		case <-time.After(500 * time.Millisecond):
			asserts.Fail("timed out, did not get the cluster")
		}
	})

	t.Run("delete shoot with recovered obj", func(t *testing.T) {
		technicalID, err := ctrl.getTechnicalID(defaultShoot)
		if err != nil {
			asserts.Failf("error getting technicalid", "%s", err)
		}

		deletedShoot := cache.DeletedFinalStateUnknown{
			Key: technicalID,
			Obj: defaultShoot,
		}

		ctrl.shootDeleteHandlerFunc(deletedShoot)

		select {
		case cluster := <-clusterChannel:
			asserts.True(cluster.Deleted, "cluster should have parameter Deleted set to true")

		case <-time.After(500 * time.Millisecond):
			asserts.Fail("timed out, did not get the cluster")
		}
	})

	t.Run("delete unknown object", func(t *testing.T) {
		ctrl.shootDeleteHandlerFunc(&Cluster{})

		select {
		case <-clusterChannel:
			asserts.Fail("should not get cluster update")
		case <-time.After(500 * time.Millisecond):
		}
	})

	t.Run("delete unknown cache object", func(t *testing.T) {
		deletedShoot := cache.DeletedFinalStateUnknown{
			Key: "test",
			Obj: &Cluster{},
		}

		ctrl.shootDeleteHandlerFunc(deletedShoot)

		select {
		case <-clusterChannel:
			asserts.Fail("should not get cluster update")
		case <-time.After(500 * time.Millisecond):
		}
	})

	testTeardown(t)
}

func TestSecretUpdateHandlerFunc(t *testing.T) {
	asserts := assert.New(t)

	testSetup(t)

	t.Run("update secret", func(t *testing.T) {
		newsecret := defaultSecret.DeepCopy()
		newsecret.ResourceVersion = "2"
		newsecret.Data["clientID"] = []byte("new-new-clientid")

		if _, err := ctrl.client.KClientset.CoreV1().Secrets(defaultNamespace.Name).Update(newsecret); err != nil {
			asserts.Failf("error creating secret", "%v", err)
		}

		ctrl.secretUpdateHandlerFunc(defaultSecret, newsecret)

		select {
		case cluster := <-clusterChannel:
			expected := string(newsecret.Data["clientID"])
			got := string(cluster.CredentialData["clientID"])
			asserts.Equalf(expected, got, "should have got new clientid %s but got %s", expected, got)

		case <-time.After(500 * time.Millisecond):
			asserts.Fail("should get cluster update")
		}
	})

	testTeardown(t)
}

func testSetup(t *testing.T) {
	t.Helper()

	var (
		err     error
		asserts = assert.New(t)
		// logger, _     = zap.NewDevelopment()
		defaultLogger = log.NewNoopLogger()
	)

	ctrl, err = NewController(newFakeClient(t), "az", clusterChannel, defaultLogger)
	if err != nil {
		asserts.FailNowf("error creating controller", "%v", err)
	}

	if _, err := ctrl.client.KClientset.CoreV1().Namespaces().Create(defaultNamespace); err != nil {
		asserts.FailNowf("error creating namespace", "%v", err)
	}

	if _, err := ctrl.client.KClientset.CoreV1().Secrets(defaultNamespace.Name).Create(defaultSecret); err != nil {
		asserts.FailNowf("error creating secret", "%v", err)
	}

	if _, err := ctrl.client.GClientset.CoreV1beta1().Shoots(defaultNamespace.Name).Create(defaultShoot); err != nil {
		asserts.FailNowf("error creating shoot", "%v", err)
	}
}

func testTeardown(t *testing.T) {
	t.Helper()

	err := ctrl.client.KClientset.CoreV1().Namespaces().Delete(defaultNamespace.Name, &metav1.DeleteOptions{})
	if err != nil {
		assert.FailNowf(t, "error tearing down test", "%s", err)
	}
}
