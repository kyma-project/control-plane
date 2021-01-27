package provisioner

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/control-plane/tests/provisioner-tests/test/testkit"
	"github.com/kyma-project/control-plane/tests/provisioner-tests/test/testkit/compass/director"
	"github.com/kyma-project/control-plane/tests/provisioner-tests/test/testkit/compass/director/oauth"
	"github.com/kyma-project/control-plane/tests/provisioner-tests/test/testkit/control-plane/provisioner"
	"github.com/kyma-project/control-plane/tests/provisioner-tests/test/testkit/graphql"

	"k8s.io/client-go/kubernetes"
	v1client "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ProvisioningTimeout   = 180 * time.Minute
	UpgradeTimeout        = 90 * time.Minute
	UpgradeShootTimeout   = 35 * time.Minute
	DeprovisioningTimeout = 240 * time.Minute
	HibernationTimeout    = 20 * time.Minute

	checkInterval = 10 * time.Second
)

type TestSuite struct {
	TestID            string
	HttpClient        http.Client
	ProvisionerClient provisioner.Client
	DirectorClient    director.Client

	gardenerProviders []string

	config        testkit.TestConfig
	secretsClient v1client.SecretInterface
}

func NewTestSuite(config testkit.TestConfig) (*TestSuite, error) {
	rand.Seed(time.Now().UnixNano())

	// TODO: Sleep ensures that the Istio Sidecar is up before running the tests. We can consider adding some health endpoint in the service to avoid hardcoded sleep.
	time.Sleep(15 * time.Second)

	httpClient := newHTTPClient(true)
	provisionerClient := provisioner.NewProvisionerClient(config.InternalProvisionerURL, config.Tenant, config.QueryLogging)
	directorClient, err := newDirectorClient(config)
	if err != nil {
		return nil, err
	}

	testId := randStringBytes(8)

	return &TestSuite{
		TestID: testId,

		HttpClient:        httpClient,
		ProvisionerClient: provisionerClient,
		DirectorClient:    directorClient,

		gardenerProviders: config.Gardener.Providers,

		config: config,
	}, nil
}

func newDirectorClient(config testkit.TestConfig) (director.Client, error) {
	k8sConfig, err := restclient.InClusterConfig()
	if err != nil {
		logrus.Warnf("Failed to read in cluster config: %s", err.Error())
		logrus.Info("Trying to initialize with local config")
		home := homedir.HomeDir()
		k8sConfPath := filepath.Join(home, ".kube", "config")
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", k8sConfPath)
		if err != nil {
			return nil, errors.Errorf("failed to read k8s in-cluster configuration, %s", err.Error())
		}
	}
	coreClientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create core clientset")
	}
	secretClient := coreClientset.CoreV1().Secrets(config.DirectorClient.Namespace)
	httpClient := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	graphQLClient := graphql.NewGraphQLClient(config.DirectorClient.URL, true, config.QueryLogging)
	oauthClient := oauth.NewOauthClient(httpClient, secretClient, config.DirectorClient.OauthCredentialsSecretName)

	return director.NewDirectorClient(oauthClient, *graphQLClient, config.Tenant, logrus.WithField("service", "director_client")), nil
}

func newHTTPClient(skipCertVerification bool) http.Client {
	return http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: skipCertVerification},
		},
		Timeout: 30 * time.Second,
	}
}

func (ts *TestSuite) Setup() error {
	logrus.Infof("Setting up environment")

	return nil
}

func (ts *TestSuite) Cleanup() {
	logrus.Infof("Starting cleanup...")
	// TODO: Fetch Provisioner logs if test failed
	logrus.Infof("Cleanup completed.")
}

func (ts *TestSuite) Recover() {
	if r := recover(); r != nil {
		logrus.Warn("Recovered after panic signal: ", r)
	}
}

func (ts *TestSuite) WaitUntilOperationIsFinished(timeout time.Duration, operationID string, log *testkit.Logger) (gqlschema.OperationStatus, error) {
	var operationStatus gqlschema.OperationStatus
	var err error

	err = testkit.WaitForFunction(checkInterval, timeout, func() bool {
		operationStatus, err = ts.ProvisionerClient.RuntimeOperationStatus(operationID)
		if err != nil {
			log.Errorf("Failed to get operation status: %s", err.Error())
			return false
		}

		if operationStatus.State == gqlschema.OperationStateInProgress {
			log.Logf("Operation '%s': %s in progress", operationStatus.Operation, operationID)
			return false
		}

		return true
	})

	return operationStatus, err
}

func (ts *TestSuite) KubernetesClientFromRawConfig(t *testing.T, rawConfig string) *kubernetes.Clientset {
	tempKubeconfigFile, err := ioutil.TempFile("", "kubeconfig")
	require.NoError(t, err)
	defer func() {
		err := os.RemoveAll(tempKubeconfigFile.Name())
		if err != nil {
			logrus.Warnf("Failed to delete temporary Kubeconfig file: %s", err.Error())
		}
	}()

	_, err = tempKubeconfigFile.WriteString(rawConfig)
	require.NoError(t, err)

	kubernetesConfig, err := clientcmd.BuildConfigFromFlags("", tempKubeconfigFile.Name())
	require.NoError(t, err)
	k8sClient, err := kubernetes.NewForConfig(kubernetesConfig)
	require.NoError(t, err)

	return k8sClient
}

func (ts *TestSuite) removeCredentialsSecret(secretName string) error {
	return ts.secretsClient.Delete(context.Background(), secretName, metav1.DeleteOptions{})
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz123456789"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
