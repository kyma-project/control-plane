package btpmgrcreds

import (
	"context"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	uuid2 "github.com/google/uuid"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/provisioner"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal/storage/dbmodel"
	kymaevent "github.com/kyma-project/runtime-watcher/listener/pkg/event"
	"github.com/kyma-project/runtime-watcher/listener/pkg/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apicorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

const (
	expectedTakenInstancesCount    = 3
	expectedRejectedInstancesCount = 1
	expectedAllInstancesCount      = expectedTakenInstancesCount + expectedRejectedInstancesCount
	credentialsLen                 = 16
)

var (
	changedInstancesCount = int(math.Ceil(expectedTakenInstancesCount / 2))
	testDataIndexes       = []int{0, 2}
	random                = rand.New(rand.NewSource(1))
)

const (
	envTestAssets = "KUBEBUILDER_ASSETS"
)

type Environment struct {
	ctx          context.Context
	skrs         []*envtest.Environment
	skrRuntimeId map[string]string
	kcp          client.Client
	kebDb        storage.BrokerStorage
	logs         *logrus.Logger
	manager      *Manager
	watcher      *Watcher
	job          *Job
	t            *testing.T
}

func InitEnvironment(ctx context.Context, t *testing.T) *Environment {
	logs := logrus.New()
	logs.SetFormatter(&logrus.JSONFormatter{})
	newEnvironment := &Environment{
		skrs:  make([]*envtest.Environment, 0),
		kebDb: storage.NewMemoryStorage(),
		logs:  logs,
		ctx:   ctx,
		t:     t,
	}

	newEnvironment.createTestData()
	newEnvironment.manager = NewManager(ctx, newEnvironment.kcp, newEnvironment.kebDb.Instances(), logs, false, provisioner.NewFakeClient())
	newEnvironment.watcher = NewWatcher(ctx, "3333", "btp-manager-secret-watcher", newEnvironment.manager, logs)
	newEnvironment.job = NewJob(newEnvironment.manager, logs)
	newEnvironment.assertThatCorrectNumberOfInstancesExists()
	return newEnvironment
}

func TestBtpManagerReconciler(t *testing.T) {
	if os.Getenv(envTestAssets) == "" {
		out, err := exec.Command("/bin/sh", "../../../setup-envtest.sh").Output()
		require.NoError(t, err)
		path := strings.Replace(string(out), "\n", "", -1)
		os.Setenv(envTestAssets, path)
	}

	t.Run("btp manager credentials tests", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		environment := InitEnvironment(ctx, t)

		t.Run("reconcile, when all secrets are not set", func(t *testing.T) {
			environment.assertAllSecretsNotExists()
			takenInstancesCount, updateDone, updateNotDoneDueError, updateNotDoneDueOkState, err := environment.manager.ReconcileAll()
			assert.NoError(t, err)
			assert.Equal(t, expectedTakenInstancesCount, takenInstancesCount)
			assert.Equal(t, expectedTakenInstancesCount, updateDone)
			assert.Equal(t, 0, updateNotDoneDueError+updateNotDoneDueOkState)
			environment.assertAllSecretDataAreSet()
			environment.assureConsistency()
		})

		t.Run("reconcile, when all secrets are correct", func(t *testing.T) {
			environment.assertAllSecretDataAreSet()
			takenInstancesCount, updateDone, updateNotDoneDueError, updateNotDoneDueOkState, err := environment.manager.ReconcileAll()
			assert.NoError(t, err)
			environment.assertThatCorrectNumberOfInstancesExists()
			assert.Equal(t, expectedTakenInstancesCount, takenInstancesCount)
			assert.Equal(t, updateDone, 0)
			assert.Equal(t, updateNotDoneDueError+updateNotDoneDueOkState, expectedTakenInstancesCount)
			environment.assertAllSecretDataAreSet()
			environment.assureConsistency()
		})

		t.Run("reconcile, when some secrets are incorrect (dynamic selected)", func(t *testing.T) {
			skrs := environment.getSkrsForSimulateChange([]int{})
			environment.simulateSecretChangeOnSkr(skrs)
			environment.assertAllSecretDataAreSet()
			takenInstancesCount, updateDone, updateNotDoneDueError, updateNotDoneDueOkState, err := environment.manager.ReconcileAll()
			assert.NoError(t, err)
			environment.assertThatCorrectNumberOfInstancesExists()
			assert.Equal(t, expectedTakenInstancesCount, takenInstancesCount)
			assert.Equal(t, updateDone, len(skrs))
			assert.Equal(t, updateNotDoneDueError+updateNotDoneDueOkState, expectedTakenInstancesCount-len(skrs))
			environment.assertAllSecretDataAreSet()
			environment.assureConsistency()
		})

		t.Run("reconcile, when some secrets are incorrect (static selected)", func(t *testing.T) {
			max := max(testDataIndexes)
			assert.GreaterOrEqual(t, expectedTakenInstancesCount-1, max)
			skrs := environment.getSkrsForSimulateChange(testDataIndexes)
			environment.simulateSecretChangeOnSkr(skrs)
			environment.assertAllSecretDataAreSet()
			takenInstancesCount, updateDone, updateNotDoneDueError, updateNotDoneDueOkState, err := environment.manager.ReconcileAll()
			assert.NoError(t, err)
			environment.assertThatCorrectNumberOfInstancesExists()
			assert.Equal(t, expectedTakenInstancesCount, takenInstancesCount)
			assert.Equal(t, updateDone, len(testDataIndexes))
			assert.Equal(t, updateNotDoneDueError+updateNotDoneDueOkState, expectedTakenInstancesCount-len(testDataIndexes))
			environment.assertAllSecretDataAreSet()
			environment.assureConsistency()
		})

		t.Run("change one instance", func(t *testing.T) {
			skrs := environment.getSkrsForSimulateChange([]int{0})
			environment.simulateSecretChangeOnSkr(skrs)
			kymaName := environment.findRuntimeIdForSkr(skrs[0].Config.Host)
			inconsistentClusters := environment.assureThatClusterIsInIncorrectState()
			assert.Equal(t, 1, inconsistentClusters)
			go environment.watcher.ReactOnSkrEvent()
			time.Sleep(time.Millisecond * 100)
			environment.watcher.listener.ReceivedEvents <- event.GenericEvent{Object: kymaevent.GenericEvent(&types.WatchEvent{
				Owner: client.ObjectKey{
					Name: kymaName,
				},
			})}
			time.Sleep(time.Millisecond * 100)
			environment.assureConsistency()
		})

		t.Run("change many instances", func(t *testing.T) {
			assert.GreaterOrEqual(t, expectedTakenInstancesCount, 1)
			skrs := environment.getSkrsForSimulateChange([]int{1, expectedTakenInstancesCount - 1})
			environment.simulateSecretChangeOnSkr(skrs)
			assert.GreaterOrEqual(t, len(skrs), 2)
			kymaName := environment.findRuntimeIdForSkr(skrs[0].Config.Host)
			kymaName2 := environment.findRuntimeIdForSkr(skrs[1].Config.Host)
			inconsistentClusters := environment.assureThatClusterIsInIncorrectState()
			assert.Equal(t, 2, inconsistentClusters)
			go environment.watcher.ReactOnSkrEvent()
			time.Sleep(time.Millisecond * 100)
			wg := &sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				environment.watcher.listener.ReceivedEvents <- event.GenericEvent{Object: kymaevent.GenericEvent(&types.WatchEvent{
					Owner: client.ObjectKey{
						Name: kymaName,
					},
				})}
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				environment.watcher.listener.ReceivedEvents <- event.GenericEvent{Object: kymaevent.GenericEvent(&types.WatchEvent{
					Owner: client.ObjectKey{
						Name: kymaName2,
					},
				})}
			}()
			defer wg.Wait()
			time.Sleep(time.Millisecond * 100)
			environment.assureConsistency()
		})

		t.Cleanup(func() {
			cancel()
		})
	})
}

func TestManager(t *testing.T) {
	manager := Manager{
		logger: logrus.New(),
	}
	t.Run("compare secrets with all different data", func(t *testing.T) {
		current, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "a",
			ClientSecret:      "a",
			ServiceManagerURL: "a",
			URL:               "a",
			XSAppName:         "a",
		}, "a")
		assert.NoError(t, err)

		expected, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "b",
			ClientSecret:      "b",
			ServiceManagerURL: "b",
			URL:               "b",
			XSAppName:         "b",
		}, "b")
		assert.NoError(t, err)

		notMatchingKeys, err := manager.compareSecrets(current, expected)
		assert.NoError(t, err)
		assert.NotNil(t, notMatchingKeys)
		assert.Greater(t, len(notMatchingKeys), 0)
		assert.Equal(t, notMatchingKeys, []string{secretClientSecret, secretClientId, secretSmUrl, secretTokenUrl, secretClusterId})
	})

	t.Run("compare secrets with partially different data", func(t *testing.T) {
		current, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "a",
			ClientSecret:      "a",
			ServiceManagerURL: "a",
			URL:               "a",
			XSAppName:         "a",
		}, "a")
		assert.NoError(t, err)

		expected, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "b",
			ClientSecret:      "b",
			ServiceManagerURL: "a",
			URL:               "a",
			XSAppName:         "a",
		}, "a")
		assert.NoError(t, err)

		notMatchingKeys, err := manager.compareSecrets(current, expected)
		assert.NoError(t, err)
		assert.NotNil(t, notMatchingKeys)
		assert.Greater(t, len(notMatchingKeys), 0)
		assert.Equal(t, notMatchingKeys, []string{secretClientSecret, secretClientId})
	})

	t.Run("compare secrets with the same data", func(t *testing.T) {
		current, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "a1",
			ClientSecret:      "a2",
			ServiceManagerURL: "a3",
			URL:               "a4",
			XSAppName:         "a5",
		}, "a6")
		assert.NoError(t, err)

		expected, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "a1",
			ClientSecret:      "a2",
			ServiceManagerURL: "a3",
			URL:               "a4",
			XSAppName:         "a5",
		}, "a6")
		assert.NoError(t, err)

		notMatchingKeys, err := manager.compareSecrets(current, expected)
		assert.NoError(t, err)
		assert.NotNil(t, notMatchingKeys)
		assert.Equal(t, len(notMatchingKeys), 0)
	})

	t.Run("compare secrets where some of data is missing and data is same", func(t *testing.T) {
		current, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "a",
			ClientSecret:      "a",
			ServiceManagerURL: "a",
			URL:               "a",
			XSAppName:         "a",
		}, "a")
		assert.NoError(t, err)
		delete(current.Data, secretClientSecret)

		expected, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "a",
			ClientSecret:      "a",
			ServiceManagerURL: "a",
			URL:               "a",
			XSAppName:         "a",
		}, "a")

		notMatchingKeys, err := manager.compareSecrets(current, expected)
		assert.Nil(t, notMatchingKeys)
		assert.Error(t, err)
	})

	t.Run("compare secrets where some of data is missing and data are different", func(t *testing.T) {
		current, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "a",
			ClientSecret:      "a",
			ServiceManagerURL: "a",
			URL:               "a",
			XSAppName:         "a",
		}, "a")
		assert.NoError(t, err)
		delete(current.Data, secretClientSecret)

		expected, err := PrepareSecret(&internal.ServiceManagerOperatorCredentials{
			ClientID:          "b",
			ClientSecret:      "b",
			ServiceManagerURL: "b",
			URL:               "b",
			XSAppName:         "b",
		}, "b")
		assert.NoError(t, err)

		notMatchingKeys, err := manager.compareSecrets(current, expected)
		assert.Nil(t, notMatchingKeys)
		assert.Error(t, err)
	})
}

func (e *Environment) createTestData() {
	e.createClusters(expectedTakenInstancesCount)
	e.skrRuntimeId = make(map[string]string, 0)
	for i := 0; i < expectedTakenInstancesCount; i++ {
		cfg := *e.skrs[i].Config
		clusterId := cfg.Host
		kubeConfig := restConfigToString(cfg)
		require.NotEmpty(e.t, kubeConfig)
		instanceId, runtimeId := e.createInstance(kubeConfig, generateServiceManagerCredentials(), clusterId)
		e.createKyma(runtimeId, instanceId)
		e.skrRuntimeId[clusterId] = runtimeId
	}

	for i := 0; i < expectedRejectedInstancesCount; i++ {
		e.createInstance("", generateServiceManagerCredentials(), "")
	}
}

func (e *Environment) createClusters(count int) {
	tempSkrs := make([]*envtest.Environment, 0)
	wg := &sync.WaitGroup{}
	for i := 0; i <= count; i++ {
		wg.Add(1)
		func(i int) {
			defer wg.Done()
			if i == count {
				//KCP
				testEnv := &envtest.Environment{
					CRDDirectoryPaths: []string{"testdata/crds/kyma.yaml"},
				}
				cfg, err := testEnv.Start()
				if err != nil {
					e.logs.Errorf("%e", err)
					return
				}
				k8sClient, err := client.New(cfg, client.Options{})
				if err != nil {
					e.logs.Errorf("%e", err)
					return
				}
				e.kcp = k8sClient

				namespace := &apicorev1.Namespace{}
				namespace.ObjectMeta = metav1.ObjectMeta{Name: kcpNamespace}
				err = e.kcp.Create(context.Background(), namespace)
				if err != nil {
					e.logs.Errorf("%e", err)
					return
				}
			} else {
				//SKR
				testEnv := &envtest.Environment{}
				_, err := testEnv.Start()
				if err != nil {
					e.logs.Errorf("%e", err)
					return
				}
				tempSkrs = append(tempSkrs, testEnv)
			}
		}(i)
	}
	wg.Wait()
	e.skrs = append(e.skrs, tempSkrs...)
	require.Equal(e.t, len(e.skrs), count)
	require.NotNil(e.t, e.kcp)
}

func (e *Environment) createInstance(kubeConfig string, credentials *internal.ServiceManagerOperatorCredentials, clusterId string) (string, string) {
	instanceId, err := uuid2.NewUUID()
	require.NoError(e.t, err)

	runtimeId := ""
	reconcilable := false
	if kubeConfig != "" && clusterId != "" {
		runtimeUUID, err := uuid2.NewUUID()
		require.NoError(e.t, err)
		runtimeId = runtimeUUID.String()
		e.createKubeConfigSecret(kubeConfig, runtimeId)
		reconcilable = true
	}

	instance := &internal.Instance{
		InstanceID: instanceId.String(),
		RuntimeID:  runtimeId,
		InstanceDetails: internal.InstanceDetails{
			ServiceManagerClusterID: clusterId,
		},
		Parameters: internal.ProvisioningParameters{
			ErsContext: internal.ERSContext{
				SMOperatorCredentials: credentials,
			},
			Parameters: internal.ProvisioningParametersDTO{
				Kubeconfig: kubeConfig,
			},
		},
	}
	instance.Reconcilable = reconcilable

	err = e.kebDb.Instances().Insert(*instance)
	require.NoError(e.t, err)
	return instanceId.String(), runtimeId
}

func (e *Environment) createKyma(runtimeId, instanceId string) {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(KymaGvk)
	u.SetNamespace(kcpNamespace)
	u.SetName(runtimeId)
	labels := make(map[string]string, 1)
	labels[instanceIdLabel] = instanceId
	u.SetLabels(labels)
	err := e.kcp.Create(e.ctx, u)
	require.NoError(e.t, err)
}

func (e *Environment) createKubeConfigSecret(cfg, runtimeId string) {
	secret := &apicorev1.Secret{
		TypeMeta: metav1.TypeMeta{Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getKubeConfigSecretName(runtimeId),
			Namespace: kcpNamespace,
		},
		Data: map[string][]byte{
			"config": []byte(cfg),
		},
		Type: apicorev1.SecretTypeOpaque,
	}
	err := e.kcp.Create(e.ctx, secret)
	require.NoError(e.t, err)
}

func (e *Environment) changeSecret(restCfg *rest.Config) {
	skrSecret := e.getSecretFromSkr(restCfg)
	newCredentials := generateServiceManagerCredentials()
	skrSecret.Data[secretClientSecret] = []byte(newCredentials.ClientSecret)
	skrSecret.Data[secretSmUrl] = []byte(newCredentials.ServiceManagerURL)
	skrSecret.Data[secretTokenUrl] = []byte(newCredentials.URL)
	skrSecret.Data[secretClusterId] = []byte(generateRandomText(credentialsLen))
	skrSecret.Data[secretClientId] = []byte(newCredentials.ClientID)
	e.updateSecretToSkr(restCfg, skrSecret)
}

func (e *Environment) getSecretFromSkr(restCfg *rest.Config) *apicorev1.Secret {
	skrClient, err := client.New(restCfg, client.Options{})
	require.NoError(e.t, err)
	skrSecret := &apicorev1.Secret{}
	err = skrClient.Get(context.Background(), client.ObjectKey{Name: BtpManagerSecretName, Namespace: BtpManagerSecretNamespace}, skrSecret)
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	require.NoError(e.t, err)
	return skrSecret
}

func (e *Environment) updateSecretToSkr(restCfg *rest.Config, secret *apicorev1.Secret) {
	skrClient, err := client.New(restCfg, client.Options{})
	require.NoError(e.t, err)
	err = skrClient.Update(context.Background(), secret)
	require.NoError(e.t, err)
}

func (e *Environment) getSkrsForSimulateChange(skrIndexes []int) []*envtest.Environment {
	var result []*envtest.Environment
	if skrIndexes == nil || len(skrIndexes) == 0 {
		indexSet := map[int]struct{}{}
		for {
			if len(indexSet) == changedInstancesCount {
				break
			}
			random := rand.Intn(expectedTakenInstancesCount)
			_, ok := indexSet[random]
			if !ok {
				indexSet[random] = struct{}{}
			}
		}

		for index, _ := range indexSet {
			testEnv := e.skrs[index]
			result = append(result, testEnv)
		}
	} else {
		for _, index := range skrIndexes {
			testEnv := e.skrs[index]
			result = append(result, testEnv)
		}
	}
	return result
}

func (e *Environment) simulateSecretChangeOnSkr(skrs []*envtest.Environment) {
	for _, skr := range skrs {
		e.changeSecret(skr.Config)
	}
}

func (e *Environment) findRuntimeIdForSkr(host string) string {
	value, ok := e.skrRuntimeId[host]
	require.True(e.t, ok)
	return value
}

func (e *Environment) assertAllSecretsNotExists() {
	for _, skr := range e.skrs {
		skrSecret := e.getSecretFromSkr(skr.Config)
		require.Nil(e.t, skrSecret)
	}
}

func (e *Environment) assertAllSecretsExists() {
	for _, skr := range e.skrs {
		skrSecret := e.getSecretFromSkr(skr.Config)
		require.NotNil(e.t, skrSecret)
	}
}

func (e *Environment) assertAllSecretDataAreSet() {
	for _, skr := range e.skrs {
		skrSecret := e.getSecretFromSkr(skr.Config)
		require.NotNil(e.t, skrSecret)

		require.NotEmpty(e.t, getString(skrSecret.Data, secretClientId))
		require.NotEmpty(e.t, getString(skrSecret.Data, secretClientSecret))
		require.NotEmpty(e.t, getString(skrSecret.Data, secretSmUrl))
		require.NotEmpty(e.t, getString(skrSecret.Data, secretTokenUrl))
		require.NotEmpty(e.t, getString(skrSecret.Data, secretClusterId))

	}
}

func (e *Environment) assureConsistency() {
	takenInstances, err := e.manager.GetReconcileCandidates()
	require.NoError(e.t, err)
	require.Equal(e.t, expectedTakenInstancesCount, len(takenInstances))

	for _, instance := range takenInstances {
		skrK8sCfg, credentials := []byte(instance.Parameters.Parameters.Kubeconfig), instance.Parameters.ErsContext.SMOperatorCredentials
		restCfg, err := clientcmd.RESTConfigFromKubeConfig(skrK8sCfg)
		require.NoError(e.t, err)
		skrSecret := e.getSecretFromSkr(restCfg)
		require.NotNil(e.t, skrSecret)

		require.Equal(e.t, getString(skrSecret.Data, secretClientId), credentials.ClientID)
		require.Equal(e.t, getString(skrSecret.Data, secretClientSecret), credentials.ClientSecret)
		require.Equal(e.t, getString(skrSecret.Data, secretSmUrl), credentials.ServiceManagerURL)
		require.Equal(e.t, getString(skrSecret.Data, secretTokenUrl), credentials.URL)
		require.Equal(e.t, getString(skrSecret.Data, secretClusterId), instance.InstanceDetails.ServiceManagerClusterID)
	}
}

func (e *Environment) assureThatClusterIsInIncorrectState() int {
	takenInstances, err := e.manager.GetReconcileCandidates()
	require.NoError(e.t, err)
	require.Equal(e.t, expectedTakenInstancesCount, len(takenInstances))

	incorrectClusters := 0
	for _, instance := range takenInstances {
		require.NoError(e.t, err)
		skrK8sCfg, credentials := []byte(instance.Parameters.Parameters.Kubeconfig), instance.Parameters.ErsContext.SMOperatorCredentials
		restCfg, err := clientcmd.RESTConfigFromKubeConfig(skrK8sCfg)
		require.NoError(e.t, err)
		skrSecret := e.getSecretFromSkr(restCfg)
		require.NotNil(e.t, skrSecret)

		if getString(skrSecret.Data, secretClientSecret) != credentials.ClientSecret {
			incorrectClusters++
			continue
		}
		if getString(skrSecret.Data, secretClientId) != credentials.ClientID {
			incorrectClusters++
			continue
		}
		if getString(skrSecret.Data, secretTokenUrl) != credentials.URL {
			incorrectClusters++
			continue
		}
		if getString(skrSecret.Data, secretClusterId) != instance.InstanceDetails.ServiceManagerClusterID {
			incorrectClusters++
			continue
		}
	}

	return incorrectClusters
}

func (e *Environment) assertThatCorrectNumberOfInstancesExists() {
	instances, _, _, err := e.kebDb.Instances().List(dbmodel.InstanceFilter{})
	require.NoError(e.t, err)
	require.Equal(e.t, expectedAllInstancesCount, len(instances))
}

func restConfigToString(restConfig rest.Config) string {
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
		return ""
	} else {
		return string(bytes)
	}
}

func generateServiceManagerCredentials() *internal.ServiceManagerOperatorCredentials {
	return &internal.ServiceManagerOperatorCredentials{
		ClientID:          generateRandomText(credentialsLen),
		ClientSecret:      generateRandomText(credentialsLen),
		ServiceManagerURL: generateRandomText(credentialsLen),
		URL:               generateRandomText(credentialsLen),
		XSAppName:         generateRandomText(credentialsLen),
	}
}

func generateRandomText(count int) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	runes := make([]rune, count)
	for i := range runes {
		runes[i] = letterRunes[random.Intn(len(letterRunes))]
	}
	return string(runes)
}

func max(slice []int) int {
	max := 0
	for _, v := range slice {
		if v > max {
			max = v
		}
	}
	return max
}

func getString(m map[string][]byte, key string) string {
	value, ok := m[key]
	if !ok {
		return ""
	}
	return string(value)
}
