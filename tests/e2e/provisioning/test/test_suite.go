package test

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/director"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/control-plane/tests/e2e/provisioning/pkg/client/broker"
	"github.com/kyma-project/control-plane/tests/e2e/provisioning/pkg/client/runtime"
	"github.com/kyma-project/control-plane/tests/e2e/provisioning/pkg/client/v1_client"
	"github.com/ory/hydra-maester/api/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrischmann/envconfig"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type Config struct {
	Broker   broker.Config
	Director director.Config
	Gardener gardener.Config

	TenantID             string `default:"d9994f8f-7e46-42a8-b2c1-1bfff8d2fe05"`
	SkipCertVerification bool   `envconfig:"default=true"`

	ProvisionerURL        string        `default:"http://kcp-provisioner.kcp-system.svc.cluster.local:3000/graphql"`
	ProvisionTimeout      time.Duration `default:"3h"`
	DeprovisionTimeout    time.Duration `default:"1h"`
	PreUpgradeKymaVersion string        `envconfig:"optional"`
	ConfigName            string        `default:"e2e-runtime-config"`
	DeployNamespace       string        `default:"kcp-system"`

	UpgradeTest  bool `envconfig:"default=false"`
	DummyTest    bool `default:"false"`
	CleanupPhase bool `default:"false"`

	BusolaURL string
}

// Suite provides set of clients able to provision and test Kyma runtime
type Suite struct {
	t *testing.T

	upgradeSuite *UpgradeSuite

	log             logrus.FieldLogger
	brokerClient    *broker.Client
	runtimeClient   *runtime.Client
	secretClient    v1_client.Secrets
	configMapClient v1_client.ConfigMaps

	PreUpgradeKymaVersion string
	dashboardChecker      *runtime.DashboardChecker

	directorClient *director.Client

	ProvisionTimeout   time.Duration
	DeprovisionTimeout time.Duration

	InstanceID string

	ConfigName      string
	DeployNamespace string

	IsUpgradeTest  bool
	IsDummyTest    bool
	IsCleanupPhase bool

	BusolaURL string
}

const (
	instanceIdKey   = "instanceId"
	dashboardUrlKey = "dashboardUrl"
	kubeconfigKey   = "config"
	subAccountID    = "39ba9a66-2c1a-4fe4-a28e-6e5db434084e"
	userID          = "test@test.com"
)

func newTestSuite(t *testing.T) *Suite {
	ctx := context.Background()
	cfg := &Config{}
	err := envconfig.InitWithPrefix(cfg, "APP")
	require.NoError(t, err)

	log := logrus.New()

	k8sConfig, err := config.GetConfig()
	if err != nil {
		panic(err)
	}

	cli, err := client.New(k8sConfig, client.Options{})
	if err != nil {
		panic(err)
	}

	oAuth2Config, err := createBrokerOAuthConfig(ctx, cli, cfg)
	if err != nil {
		panic(err)
	}

	secretClient := v1_client.NewSecretClient(cli, log)
	configMapClient := v1_client.NewConfigMapClient(cli, log)

	instanceID := uuid.New().String()
	if cfg.CleanupPhase {
		cfgMap, err := configMapClient.Get(cfg.ConfigName, cfg.DeployNamespace)
		require.NoError(t, err)

		instanceID = cfgMap.Data[instanceIdKey]
		log.Infof("using instance ID %s", instanceID)
	}

	httpClient := newHTTPClient(cfg.SkipCertVerification)

	brokerClient := broker.NewClient(ctx, cfg.Broker, cfg.TenantID, instanceID, subAccountID, userID, oAuth2Config, log.WithField("service", "broker_client"))

	directorClient := director.NewDirectorClient(ctx, cfg.Director, log.WithField("service", "director_client"))

	runtimeClient := runtime.NewClient(cfg.ProvisionerURL, cfg.TenantID, instanceID, *httpClient, directorClient, log.WithField("service", "runtime_client"))

	dashboardChecker := runtime.NewDashboardChecker(*httpClient, log.WithField("service", "dashboard_checker"))

	suite := &Suite{
		t:   t,
		log: log,

		dashboardChecker: dashboardChecker,
		brokerClient:     brokerClient,
		runtimeClient:    runtimeClient,
		secretClient:     secretClient,
		configMapClient:  configMapClient,

		directorClient: directorClient,

		InstanceID:            instanceID,
		ProvisionTimeout:      cfg.ProvisionTimeout,
		DeprovisionTimeout:    cfg.DeprovisionTimeout,
		PreUpgradeKymaVersion: cfg.PreUpgradeKymaVersion,

		ConfigName:      cfg.ConfigName,
		DeployNamespace: cfg.DeployNamespace,

		IsUpgradeTest:  cfg.UpgradeTest,
		IsDummyTest:    cfg.DummyTest,
		IsCleanupPhase: cfg.CleanupPhase,

		BusolaURL: cfg.BusolaURL,
	}

	if suite.IsUpgradeTest {
		suite.upgradeSuite = newUpgradeSuite(t, ctx, oAuth2Config, cfg.Broker, log)
	}

	return suite
}

// Cleanup removes all data associated with the test along with runtime
func (ts *Suite) Cleanup() {
	ts.log.Info("Cleaning up...")
	err := ts.cleanupResources()
	assert.NoError(ts.t, err)
	operationID, err := ts.brokerClient.DeprovisionRuntime()
	require.NoError(ts.t, err)
	err = ts.brokerClient.AwaitOperationSucceeded(operationID, ts.DeprovisionTimeout)
	assert.NoError(ts.t, err)
}

// cleanupResources removes secret and config map used to store data about the test
func (ts *Suite) cleanupResources() error {
	ts.log.Infof("removing secret %s", ts.ConfigName)
	err := ts.secretClient.Delete(v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ts.ConfigName,
			Namespace: ts.DeployNamespace,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "while waiting for secret %s deletion", ts.ConfigName)
	}

	ts.log.Infof("removing config map %s", ts.ConfigName)
	err = ts.configMapClient.Delete(v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ts.ConfigName,
			Namespace: ts.DeployNamespace,
		},
	})
	if err != nil {
		return errors.Wrapf(err, "while waiting for config map %s deletion", ts.ConfigName)
	}
	return nil
}

func (ts *Suite) testSecret(config *string) v1.Secret {
	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ts.ConfigName,
			Namespace: ts.DeployNamespace,
		},
		Data: map[string][]byte{
			kubeconfigKey: []byte(*config),
		},
	}
}

func (ts *Suite) testConfigMap() v1.ConfigMap {
	return v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ts.ConfigName,
			Namespace: ts.DeployNamespace,
		},
		Data: map[string]string{
			instanceIdKey: ts.InstanceID,
		},
	}
}

func createBrokerOAuthConfig(ctx context.Context, k8sclient client.Client, cfg *Config) (broker.BrokerOAuthConfig, error) {
	var brokerOAuthConfig broker.BrokerOAuthConfig

	err := v1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return brokerOAuthConfig, errors.Wrap(err, "while adding hydra-maester v1alpha1 to schema")
	}

	oAuth2Client := &v1alpha1.OAuth2Client{}
	err = k8sclient.Get(ctx, client.ObjectKey{
		Namespace: cfg.DeployNamespace,
		Name:      cfg.Broker.ClientName,
	}, oAuth2Client)
	if err != nil {
		return brokerOAuthConfig, errors.Wrapf(err, "while getting oAuth2Client %s", cfg.Broker.ClientName)
	}

	brokerSecret := &v1.Secret{}
	err = k8sclient.Get(ctx, client.ObjectKey{
		Namespace: cfg.DeployNamespace,
		Name:      oAuth2Client.Spec.SecretName,
	}, brokerSecret)
	if err != nil {
		return brokerOAuthConfig, errors.Wrapf(err, "while getting secret %s", oAuth2Client.Spec.SecretName)
	}

	clientID, ok := brokerSecret.Data["client_id"]
	if !ok {
		return brokerOAuthConfig, errors.Errorf("cannot find client_id key in secret %s", oAuth2Client.Spec.SecretName)
	}
	clientSecret, ok := brokerSecret.Data["client_secret"]
	if !ok {
		return brokerOAuthConfig, errors.Errorf("cannot find client_secret key in secret %s", oAuth2Client.Spec.SecretName)
	}

	brokerOAuthConfig.ClientID = string(clientID)
	brokerOAuthConfig.ClientSecret = string(clientSecret)
	brokerOAuthConfig.Scope = oAuth2Client.Spec.Scope

	return brokerOAuthConfig, nil
}

func newHTTPClient(insecureSkipVerify bool) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecureSkipVerify,
			},
		},
		Timeout: 30 * time.Second,
	}
}
