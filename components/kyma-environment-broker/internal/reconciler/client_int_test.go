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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"
)

var createClusterJSONPayload = filepath.Join(".", "test", "createCluster.json")
var testKubeconfig = filepath.Join(".", "test", "kubeconfig.yaml")

const reconcilerUrlEnv = "RECONCILER_URL"
const testKubeconfigPathKey = "TEST_KUBECONFIG_PATH"

/**
Those tests perform operation on the Reconciler inventory using the client. Before running any test set the following envs:
 - RECONCILER_URL - url to locally running instance of reconciler mothership for example "http://localhost:8080"
 - TEST_KUBECONFIG_PATH - path to kubeconfig for the cluster to reconcile

EXAMPLE USAGE:
kind create cluster
kind get kubeconfig > $(pwd)/internal/reconciler/test/kindkubeconfig.yaml

then run 'base' reconciler, reconciler-mothership (inventory API) and psql db:

gh repo clone kyma-incubator/reconciler
./scripts/postgres.sh start
make build-darwin

1st terminal:
./bin/reconciler-darwin reconciler start base --server-port=8081 --verbose

2st terminal:
./bin/reconciler-darwin mothership start  --reconcilers configs/component-reconcilers.json --verbose

3rd terminal:
export RECONCILER_URL="http://localhost:8080"
export TEST_KUBECONFIG_PATH=$(pwd)/internal/reconciler/test/kindkubeconfig.yaml
go test -v -tags=reconciler_integration ./internal/reconciler/... -run TestClient_ReconcileCluster
*/
func TestClient_ReconcileCluster(t *testing.T) {
	// given
	client := newClient(t)
	reqPayload := fixPayload(t)

	// when
	response, err := client.ApplyClusterConfig(*reqPayload)

	// then
	if err != nil {
		t.Error(err)
	}
	t.Logf("%#v", response)

	// then

	if err != nil {
		t.Error(err)
	}
	err = wait.PollImmediate(10*time.Second, 2*time.Minute, func() (done bool, err error) {
		response, callErr := client.GetLatestCluster(reqPayload.Cluster)
		if callErr != nil {
			return false, callErr
		}
		t.Logf("got status: %s", response.Status)
		if response.Status == "ready" {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(t, err)

}

func newClient(t *testing.T) Client {
	t.Helper()
	reconcilerURL, ok := os.LookupEnv(reconcilerUrlEnv)
	if !ok {
		reconcilerURL = "http://localhost:8080"
	}

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

	kubeconfigPath, ok := os.LookupEnv(testKubeconfigPathKey)
	if !ok {
		t.Errorf("%s not set", testKubeconfigPathKey)
	}
	kubeconfig, err := ioutil.ReadFile(kubeconfigPath)
	require.NoError(t, err)
	cluster.Kubeconfig = string(kubeconfig)

	return cluster
}
