package skrlisteners

import (
	"context"
	uuid2 "github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"testing"
)

const (
	instanceCount = 5
)

type TestInstances struct {
	testEnvs       []*envtest.Environment
	testInstances  []internal.Instance
	dbInMemory     storage.BrokerStorage
	instancesCount int
}

func Test(t *testing.T) {
	os.Setenv("KUBEBUILDER_ASSETS", "bin/k8s/1.25.0-darwin-arm64")

	t.Run("Test Reconcile", func(t *testing.T) {
		test := TestInstances{}
		test.instancesCount = instanceCount
		defer test.clean()
		test.dbInMemory = storage.NewMemoryStorage()
		errs := test.PrepareFakeData()
		assert.Len(t, errs, 0)

		h := NewBtpManagerSecretListener(context.Background(), test.dbInMemory.Instances(), "", "", nil, nil)

		//here we assume that all instances on cluster are empty, by reconcile they should be set
		h.Reconcile()
		test.SimulateChangeOnSkr()
		h.Reconcile()
	})
}

func (t *TestInstances) SimulateChangeOnSkr() {
	for i := 0; i < 10; i++ {
		te := t.testEnvs[i]
		_, _ = client.New(te.Config, client.Options{})
	}
}

func (t *TestInstances) PrepareFakeData() []error {
	var errs []error
	for i := 0; i < instanceCount; i++ {
		kubeConfig, err := t.createCluster()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := t.appendInstance(kubeConfig); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (t *TestInstances) clean() {
	for _, testEnv := range t.testEnvs {
		testEnv.Stop()
	}
}

func (t *TestInstances) createCluster() (string, error) {
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}
	restConfig, err := testEnv.Start()
	if err != nil {
		return "", err
	}
	t.testEnvs = append(t.testEnvs, testEnv)
	return t.RestConfigToString(*restConfig)
}

func (t *TestInstances) RestConfigToString(restConfig rest.Config) (string, error) {
	bytes, err := clientcmd.Write(api.Config{
		Clusters: map[string]*api.Cluster{
			"default": {
				Server:                   restConfig.Host,
				InsecureSkipTLSVerify:    restConfig.Insecure,
				CertificateAuthorityData: restConfig.CAData,
			},
		},
		Contexts: map[string]*api.Context{
			"default": {
				Cluster:  "default",
				AuthInfo: "default",
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"default": {
				ClientCertificateData: restConfig.CertData,
				ClientKeyData:         restConfig.KeyData,
				Token:                 restConfig.BearerToken,
				Username:              restConfig.Username,
				Password:              restConfig.Password,
			},
		},
		CurrentContext: "default",
	})
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (t *TestInstances) appendInstance(kubeConfig string) error {
	uuid, err := uuid2.NewUUID()
	if err != nil {
		return err
	}
	instance := &internal.Instance{
		InstanceID: uuid.String(),
		Parameters: internal.ProvisioningParameters{
			ErsContext: internal.ERSContext{
				SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{
					ClientID:          "",
					ClientSecret:      "",
					ServiceManagerURL: "",
					URL:               "",
					XSAppName:         "",
				},
			},
			Parameters: internal.ProvisioningParametersDTO{
				Kubeconfig: kubeConfig,
			},
		},
	}
	t.testInstances = append(t.testInstances, *instance)
	err = t.dbInMemory.Instances().Insert(*instance)
	if err != nil {
		return err
	}
	return nil
}

func (t *TestInstances) GetFakeCredentials() []internal.ServiceManagerCredentials {
	var x []internal.ServiceManagerCredentials
	return x
}
