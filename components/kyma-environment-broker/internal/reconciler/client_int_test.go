// +build reconciler_integration

package reconciler

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"
)

var createClusterJSONPayload = filepath.Join(".", "test", "createCluster.json")

/**
Those tests perform operation on the Reconciler inventory using the client. Before running any test set the following envs:
 - RECONCILER_URL - url to locally running instance of reconciler mothership for example "http://localhost:8080"

and add one-line (with \n escapes) KUBECONFIG for the cluster you want to reconcile to test/createCluster.json "kubeconfig" field
*/

// Running: go test -v -tags=reconciler_integration ./internal/reconciler/... -run TestClient_ReconcileCluster
func TestClient_ReconcileCluster(t *testing.T) {
	// given
	client := newClient(t)
	reqPayload := fixPayload(t)

	// when
	response, err := client.RegisterCluster(*reqPayload)

	// then
	if err != nil {
		t.Error(err)
	}
	t.Logf("%#v", response)

	// then

	if err != nil {
		t.Error(err)
	}
	wait.PollImmediate(10*time.Second, 2*time.Minute, func() (done bool, err error) {
		response, err := client.GetLatestCluster(reqPayload.Cluster)
		if err != nil {
			return false, err
		}
		t.Logf("got status: %s", response.Status)
		if response.Status == "ready" {
			return true, nil
		}
		return false, nil
	})

}

func newClient(t *testing.T) Client {
	t.Helper()

	reconcilerURL := os.Getenv("RECONCILER_URL")
	cfg := &Config{reconcilerURL: reconcilerURL}
	client := NewReconcilerClient(http.DefaultClient, logrus.WithField("test-client", "reconciler"), cfg)

	return client
}

func fixPayload(t *testing.T) *Cluster {
	cluster := &Cluster{}
	data, err := ioutil.ReadFile(createClusterJSONPayload)
	require.NoError(t, err)
	err = json.Unmarshal(data, cluster)
	require.NoError(t, err)

	//TODO: load kubeconfig from file
	//cluster.Kubeconfig = os.Getenv("TEST_KUBECONFIG")

	return cluster
}
