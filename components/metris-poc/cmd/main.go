package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/kyma-project/control-plane/components/metris-poc/pkg/env"
	"github.com/kyma-project/control-plane/components/metris-poc/pkg/keb"
	restclient "k8s.io/client-go/rest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	corev1 "k8s.io/api/core/v1"

	kebgardenerclient "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"
	"k8s.io/client-go/tools/clientcmd"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"

	"k8s.io/client-go/tools/cache"

	gardenerv1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gorilla/mux"
	system_info "github.com/kyma-project/control-plane/components/metris-poc/pkg/system-info"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
)

const (
	livenessURI         = "/healthz"
	readinessURI        = "/readyz"
	DefaultResyncPeriod = 10 * time.Second
)

var (
	shootGVR = schema.GroupVersionResource{
		Version:  gardenerv1beta1.SchemeGroupVersion.Version,
		Group:    gardenerv1beta1.SchemeGroupVersion.Group,
		Resource: "shoots",
	}
	secretGVR = schema.GroupVersionResource{
		Version:  v1.SchemeGroupVersion.Version,
		Group:    v1.SchemeGroupVersion.Group,
		Resource: "secrets",
	}
)

type options struct {
	requestTimeout *time.Duration
	cfg            *env.Config
}

func main() {
	log.Print("Starting POC")
	requestTimeout := flag.Duration("requestTimeout", 10*time.Second, "Timeout for services.")
	flag.Parse()

	cfg := env.GetConfig()

	opts := &options{
		requestTimeout: requestTimeout,
		cfg:            cfg,
	}

	config, err := GetGardenerKubeconfig(cfg)
	if err != nil {
		log.Fatalf("failed to get client kubeconfig: %v", err)
	}
	log.Print("gardener kubeconfig fetched successfully")

	shootDynamicSharedInfFactory := GenerateShootInfFactory(config)
	shootLister := shootDynamicSharedInfFactory.ForResource(shootGVR).Lister().ByNamespace("garden-kyma-dev")

	secretDynamicSharedInfFactory := GenerateSecretInfFactory(config)
	secretLister := secretDynamicSharedInfFactory.ForResource(secretGVR).Lister().ByNamespace("garden-kyma-dev")

	sysInfoHandler := SysInfoHandler{
		SecretLister: &secretLister,
		ShootLister:  &shootLister,
		KEBEndpoint:  cfg.KEBEndpoint,
	}
	ctx := context.Background()
	WaitForCacheSyncOrDie(ctx, shootDynamicSharedInfFactory)
	WaitForCacheSyncOrDie(ctx, secretDynamicSharedInfFactory)

	log.Printf("Shoot informers are synced")
	log.Printf("Secret informers are synced")

	server := &http.Server{
		Addr:         ":8080",
		Handler:      sysInfoHandler.NewHandler(),
		WriteTimeout: time.Duration(*opts.requestTimeout) * time.Second,
	}

	go start(server)
}

func GenerateSecretInfFactory(k8sConfig *rest.Config) dynamicinformer.DynamicSharedInformerFactory {
	dynamicClient := dynamic.NewForConfigOrDie(k8sConfig)
	dFilteredSharedInfFactory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamicClient,
		DefaultResyncPeriod,
		v1.NamespaceAll,
		nil,
	)
	dFilteredSharedInfFactory.ForResource(secretGVR)
	return dFilteredSharedInfFactory
}

func GenerateShootInfFactory(k8sConfig *rest.Config) dynamicinformer.DynamicSharedInformerFactory {
	dynamicClient := dynamic.NewForConfigOrDie(k8sConfig)
	dFilteredSharedInfFactory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamicClient,
		DefaultResyncPeriod,
		v1.NamespaceAll,
		nil,
	)
	dFilteredSharedInfFactory.ForResource(shootGVR)
	return dFilteredSharedInfFactory
}

func GetGardenerKubeconfig(cfg *env.Config) (*restclient.Config, error) {
	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: cfg.GardenerKubeconfig}
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	return kubeConfig.ClientConfig()
}

type SysInfoHandler struct {
	SecretLister *cache.GenericNamespaceLister
	ShootLister  *cache.GenericNamespaceLister
	KEBEndpoint  string
}

func start(server *http.Server) {
	if server == nil {
		log.Error("cannot start a nil HTTP server")
		return
	}

	if err := server.ListenAndServe(); err != nil {
		log.Errorf("failed to start server: %v", err)
	}
}

func (sh SysInfoHandler) NewHandler() http.Handler {
	router := mux.NewRouter()

	router.Path("/systemInfo").Handler(sh.NewSystemStatsHandler()).Methods(http.MethodGet)

	router.Path(livenessURI).Handler(CheckHealth()).Methods(http.MethodGet)

	router.Path(readinessURI).Handler(CheckHealth()).Methods(http.MethodGet)

	return router
}

func (sh SysInfoHandler) NewSystemStatsHandler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

		runtimeObjs, err := (*sh.ShootLister).List(labels.Everything())
		if err != nil {
			log.Errorf("failed to fetch shoots from gardener: %v", err)
			writer.WriteHeader(http.StatusInternalServerError)
		}
		scannedShoots := make([]*gardenerv1beta1.Shoot, 0)
		var testShoot *gardenerv1beta1.Shoot
		for _, sObj := range runtimeObjs {
			shoot, err := ConvertRuntimeObjToSubscription(sObj)
			if err != nil {
				log.Errorf("failed to convert a runtime obj to a Shoot: %v", err)
				continue
			}
			scannedShoots = append(scannedShoots, shoot)
			if !shoot.Status.IsHibernated {
				testShoot = shoot
			}
		}

		runtimes, err := keb.GetRuntimes(sh.KEBEndpoint)
		if err != nil {
			log.Errorf("failed to fetch the runtimes: %v", err)
		}
		for _, runtime := range (*runtimes).Data {
			if runtime.Status.Provisioning.State == "succeeded" {
				secretForShoot := fmt.Sprintf("%s.kubeconfig", testShoot)
				secretObj, err := (*sh.SecretLister).Get(secretForShoot)
				if err != nil {
					log.Errorf("failed to list secrets for shoots: %v", err)
				}
				if secret, ok := secretObj.(*corev1.Secret); ok {
					// connect to skr cluster
					skrKubeConfigStr := secret.Data["kubeconfig"]
					skrKubeConfig, err := kebgardenerclient.RESTConfig([]byte(skrKubeConfigStr))
					if err != nil {
						log.Errorf("failed to create kubeconfig client: %v", skrKubeConfig)
					}
					dynamicClient, err := dynamic.NewForConfig(skrKubeConfig)
					nodeGVK := schema.GroupVersionResource{
						Version:  v1.SchemeGroupVersion.Version,
						Group:    v1.SchemeGroupVersion.Group,
						Resource: "nodes",
					}
					nodeClient := dynamicClient.Resource(nodeGVK)
					ctx := context.Background()
					nodes, err := nodeClient.List(ctx, metav1.ListOptions{})
					if err != nil {
						log.Errorf("failed to fetch nodes from shoot: %s, err: %v", testShoot, err)
					}
					log.Printf("%v", nodes)
				}
			}
		}

		// GetNodes from SKR shoot
		//skrShootClient := getShootClient(secret.)
		sysInfo, err := system_info.GetShootInfo()
		if err != nil {
			writer.Write([]byte(fmt.Sprintf("failed to get sys info: %v", err)))
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		sysInfoBytes, err := json.Marshal(sysInfo)
		if err != nil {
			log.Errorf("failed to Marshal the response: %v", err)
			return
		}
		_, err = writer.Write([]byte(sysInfoBytes))
		if err != nil {
			log.Errorf("failed to write in the response: %v", err)
		}
	})
}

func ConvertRuntimeObjToSubscription(shootObj runtime.Object) (*gardenerv1beta1.Shoot, error) {
	shoot := &gardenerv1beta1.Shoot{}
	if shootUnstructured, ok := shootObj.(*unstructured.Unstructured); ok {
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(shootUnstructured.Object, shoot)
		if err != nil {
			return nil, err
		}
	}
	return shoot, nil
}

func CheckHealth() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		return
	})
}

type waitForCacheSyncFunc func(stopCh <-chan struct{}) map[schema.GroupVersionResource]bool

func WaitForCacheSyncOrDie(ctx context.Context, dc dynamicinformer.DynamicSharedInformerFactory) {
	dc.Start(ctx.Done())

	ctx, cancel := context.WithTimeout(context.Background(), DefaultResyncPeriod)
	defer cancel()

	err := hasSynced(ctx, dc.WaitForCacheSync)
	if err != nil {
		log.Fatalf("Failed to sync informer caches: %v", err)
	}
}

func hasSynced(ctx context.Context, fn waitForCacheSyncFunc) error {
	// synced gets closed as soon as fn returns
	synced := make(chan struct{})
	// closing stopWait forces fn to return, which happens whenever ctx
	// gets canceled
	stopWait := make(chan struct{})
	defer close(stopWait)

	// close the synced channel if the `WaitForCacheSync()` finished the execution cleanly
	go func() {
		informersCacheSync := fn(stopWait)
		res := true
		for _, sync := range informersCacheSync {
			if !sync {
				res = false
			}
		}
		if res {
			close(synced)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-synced:
	}

	return nil
}
