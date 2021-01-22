package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"

	kebgardenerclient "github.com/kyma-project/control-plane/components/kyma-environment-broker/common/gardener"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"

	"k8s.io/client-go/tools/cache"

	"github.com/gorilla/mux"
	"github.com/kyma-project/control-plane/components/metris-poc/pkg/env"
	system_info "github.com/kyma-project/control-plane/components/metris-poc/pkg/system-info"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
)

const (
	livenessURI  = "/healthz"
	readinessURI = "/readyz"
)

const (
	DefaultResyncPeriod = 10 * time.Second
)

var (
	secretGVK = schema.GroupVersionResource{
		Version:  v1.SchemeGroupVersion.Version,
		Group:    v1.SchemeGroupVersion.Group,
		Resource: "secrets",
	}
)

type options struct {
	requestTimeout *int
	cfg            *env.Config
}

func main() {
	fmt.Println("Starting POC")
	requestTimeout := flag.Int("requestTimeout", 1, "Timeout for services.")
	flag.Parse()

	cfg := env.GetConfig()

	opts := &options{
		requestTimeout: requestTimeout,
		cfg:            cfg,
	}

	k8sConfig, err := kebgardenerclient.NewGardenerClusterConfig(opts.cfg.GardenerKubeconfig)
	if err != nil {
		log.Fatalf("failed to initialize Gerdener cluster client")
	}
	secretDynamicFactory := GenerateSecretInfFactory(k8sConfig)
	secretLister := secretDynamicFactory.ForResource(secretGVK).Lister()
	sysInfoHandler := &SysInfoHandler{
		SecretLister: &secretLister,
	}

	// TODO remove me
	fmt.Println(cfg)

	// Create client for gardener

	//gardenerClient, err := createClientForGardener()

	// Create client for KEB

	// Create client for SKRs

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
	dFilteredSharedInfFactory.ForResource(secretGVK)
	return dFilteredSharedInfFactory
}

type SysInfoHandler struct {
	SecretLister *cache.GenericLister
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

		shootName := "foo"
		secretForShoot := fmt.Sprintf("%s.kubeconfig", shootName)
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
				log.Errorf("failed to fetch nodes from shoot: %s, err: %v", shootName, err)
			}

			log.Printf("%v", nodes)
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

func CheckHealth() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		return
	})
}
