package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/kyma-project/control-plane/components/provisioner/internal/operations/queue"

	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning/persistence/dbsession"

	"github.com/kyma-project/control-plane/components/provisioner/internal/director"
	"github.com/kyma-project/control-plane/components/provisioner/internal/gardener"
	"github.com/kyma-project/control-plane/components/provisioner/internal/graphql"
	"github.com/kyma-project/control-plane/components/provisioner/internal/oauth"
	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning"
	"github.com/kyma-project/control-plane/components/provisioner/internal/uuid"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	restclient "k8s.io/client-go/rest"
)

const (
	databaseConnectionRetries = 20
	defaultSyncPeriod         = 10 * time.Minute
)

type DynamicKubeconfigProvider interface {
	FetchFromRequest(shootName string) ([]byte, error)
}

func newProvisioningService(
	gardenerProject string,
	provisioner provisioning.Provisioner,
	dbsFactory dbsession.Factory,
	directorService director.DirectorClient,
	shootProvider gardener.ShootProvider,
	provisioningQueue queue.OperationQueue,
	deprovisioningQueue queue.OperationQueue,
	shootUpgradeQueue queue.OperationQueue,
	defaultEnableKubernetesVersionAutoUpdate,
	defaultEnableMachineImageVersionAutoUpdate bool,
	dynamicKubeconfigProvider DynamicKubeconfigProvider,

) provisioning.Service {
	uuidGenerator := uuid.NewUUIDGenerator()
	inputConverter := provisioning.NewInputConverter(uuidGenerator, gardenerProject, defaultEnableKubernetesVersionAutoUpdate, defaultEnableMachineImageVersionAutoUpdate)
	graphQLConverter := provisioning.NewGraphQLConverter()

	return provisioning.NewProvisioningService(
		inputConverter,
		graphQLConverter,
		directorService,
		dbsFactory,
		provisioner,
		uuidGenerator,
		shootProvider,
		provisioningQueue,
		deprovisioningQueue,
		shootUpgradeQueue,
		dynamicKubeconfigProvider,
	)
}

func newDirectorClient(config config) (director.DirectorClient, error) {
	file, err := os.ReadFile(config.DirectorOAuthPath)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to open director config")
	}

	cfg := DirectorOAuth{}
	err = yaml.Unmarshal(file, &cfg)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to unmarshal director config")
	}

	gqlClient := graphql.NewGraphQLClient(config.DirectorURL, true, config.SkipDirectorCertVerification)
	oauthClient := oauth.NewOauthClient(newHTTPClient(config.SkipDirectorCertVerification), cfg.Data.ClientID, cfg.Data.ClientSecret, cfg.Data.TokensEndpoint)

	return director.NewDirectorClient(gqlClient, oauthClient), nil
}

type DirectorOAuth struct {
	Data struct {
		ClientID       string `json:"client_id"`
		ClientSecret   string `json:"client_secret"`
		TokensEndpoint string `json:"tokens_endpoint"`
	} `json:"data"`
}

func newShootController(gardenerNamespace string, gardenerClusterCfg *restclient.Config, dbsFactory dbsession.Factory, auditLogTenantConfigPath string) (*gardener.ShootController, error) {

	syncPeriod := defaultSyncPeriod

	mgr, err := ctrl.NewManager(gardenerClusterCfg, ctrl.Options{SyncPeriod: &syncPeriod, Namespace: gardenerNamespace})
	if err != nil {
		return nil, fmt.Errorf("unable to create shoot controller manager: %w", err)
	}

	return gardener.NewShootController(mgr, dbsFactory, auditLogTenantConfigPath)
}

func newGardenerClusterConfig(cfg config) (*restclient.Config, error) {
	rawKubeconfig, err := os.ReadFile(cfg.Gardener.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Gardener Kubeconfig from path %s: %s", cfg.Gardener.KubeconfigPath, err.Error())
	}

	gardenerClusterConfig, err := gardener.Config(rawKubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gardener cluster config: %s", err.Error())
	}

	return gardenerClusterConfig, nil
}

func newHTTPClient(skipCertVerification bool) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: skipCertVerification},
		},
		Timeout: 30 * time.Second,
	}
}
