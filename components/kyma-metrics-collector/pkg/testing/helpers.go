package testing

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/gorilla/mux"
	kebruntime "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	timeout = 10 * time.Second
)

type NewRuntimeOpts func(*kebruntime.RuntimeDTO)

func NewRuntimesDTO(subAccountID string, shootName string, opts ...NewRuntimeOpts) kebruntime.RuntimeDTO {
	runtime := kebruntime.RuntimeDTO{
		ShootName:    shootName,
		SubAccountID: subAccountID,
		Status: kebruntime.RuntimeStatus{
			Provisioning: &kebruntime.Operation{
				State: "succeeded",
			},
		},
	}

	for _, opt := range opts {
		opt(&runtime)
	}

	return runtime
}

func WithProvisioningSucceededStatus(statusState kebruntime.State) func(*kebruntime.RuntimeDTO) {
	return func(runtime *kebruntime.RuntimeDTO) {
		runtime.Status.State = statusState
		runtime.Status.Provisioning = &kebruntime.Operation{
			State: string(kebruntime.StateSucceeded),
		}
	}
}

func WithProvisioningFailedState(runtime *kebruntime.RuntimeDTO) {
	runtime.Status.Provisioning = &kebruntime.Operation{
		State: string(kebruntime.StateFailed),
	}
}

func WithProvisionedAndDeprovisionedStatus(statusState kebruntime.State) func(*kebruntime.RuntimeDTO) {
	return func(runtime *kebruntime.RuntimeDTO) {
		runtime.Status.State = statusState
		runtime.Status.Provisioning = &kebruntime.Operation{
			State: string(kebruntime.StateSucceeded),
		}
		runtime.Status.Deprovisioning = &kebruntime.Operation{
			State: string(kebruntime.StateSucceeded),
		}
	}
}

func LoadFixtureFromFile(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func StartTestServer(path string, testHandler http.HandlerFunc, g gomega.Gomega) *httptest.Server {
	testRouter := mux.NewRouter()
	testRouter.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}).Methods(http.MethodGet)
	testRouter.HandleFunc(path, testHandler)

	// Start a local test HTTP server
	srv := httptest.NewServer(testRouter)

	// Wait until test server is ready
	g.Eventually(func() int {
		// Ignoring error is ok as it goes for retry for non-200 cases
		healthResp, err := http.Get(fmt.Sprintf("%s/health", srv.URL))
		log.Printf("retrying :%v", err)
		return healthResp.StatusCode
	}, timeout).Should(gomega.Equal(http.StatusOK))

	return srv
}

func Get2Nodes() *corev1.NodeList {
	node1 := GetNode("node1", "Standard_D8_v3")
	node2 := GetNode("node2", "Standard_D8_v3")
	return &corev1.NodeList{
		Items: []corev1.Node{node1, node2},
	}
}

func Get2NodesOpenStack() *corev1.NodeList {
	node1 := GetNode("node1", "g_c12_m48")
	node2 := GetNode("node2", "g_c12_m48")
	return &corev1.NodeList{
		Items: []corev1.Node{node1, node2},
	}
}

func Get3NodesWithStandardD8v3VMType() *corev1.NodeList {
	node1 := GetNode("node1", "Standard_D8_v3")
	node2 := GetNode("node2", "Standard_D8_v3")
	node3 := GetNode("node3", "Standard_D8_v3")
	return &corev1.NodeList{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "NodeList",
			APIVersion: "v1",
		},
		Items: []corev1.Node{node1, node2, node3},
	}
}

func Get3NodesWithFooVMType() *corev1.NodeList {
	node1 := GetNode("node1", "foo")
	node2 := GetNode("node2", "foo")
	node3 := GetNode("node3", "foo")
	return &corev1.NodeList{
		Items: []corev1.Node{node1, node2, node3},
	}
}

func GetNode(name, vmType string) corev1.Node {
	return corev1.Node{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"node.kubernetes.io/instance-type": vmType,
				"node.kubernetes.io/role":          "node",
			},
		},
	}
}

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" // 52 possibilities
	letterIdxBits = 6                                                      // 6 bits to represent 64 possibilities / indexes
	letterIdxMask = 1<<letterIdxBits - 1                                   // All 1-bits, as many as letterIdxBits
)

func GenerateRandomAlphaString(length int) string {
	result := make([]byte, length)
	bufferSize := int(float64(length) * 1.3)
	for i, j, randomBytes := 0, 0, []byte{}; i < length; j++ {
		if j%bufferSize == 0 {
			randomBytes = secureRandomBytes(bufferSize)
		}
		if idx := int(randomBytes[j%length] & letterIdxMask); idx < len(letterBytes) {
			result[i] = letterBytes[idx]
			i++
		}
	}

	return string(result)
}

// secureRandomBytes returns the requested number of bytes using crypto/rand.
func secureRandomBytes(length int) []byte {
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Fatal("Unable to generate random bytes")
	}
	return randomBytes
}

func Get3PVCs() *corev1.PersistentVolumeClaimList {
	pv5GInFooNs := GetPV("foo-5G", "foo", "5Gi")
	pv10GInFooNs := GetPV("foo-10G", "foo", "10Gi")
	pv20GInBarNs := GetPV("foo-20G", "bar", "20Gi")

	return &corev1.PersistentVolumeClaimList{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "PersistentVolumeClaimList",
			APIVersion: "v1",
		},
		ListMeta: metaV1.ListMeta{},
		Items: []corev1.PersistentVolumeClaim{
			*pv5GInFooNs,
			*pv10GInFooNs,
			*pv20GInBarNs,
		},
	}
}

func GetPVCs() *corev1.PersistentVolumeClaimList {
	pv10GInFooNs := GetPV("foo-10G", "foo", "10Gi")
	pv20GInBarNs := GetPV("foo-20G", "bar", "20Gi")

	return &corev1.PersistentVolumeClaimList{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "PersistentVolumeClaimList",
			APIVersion: "v1",
		},
		ListMeta: metaV1.ListMeta{},
		Items: []corev1.PersistentVolumeClaim{
			*pv10GInFooNs,
			*pv20GInBarNs,
		},
	}
}

func GetPV(name, namespace, capacity string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.VolumeResourceRequirements{
				Limits: nil,
				Requests: corev1.ResourceList{
					"storage": resource.MustParse(capacity),
				},
			},
		},
		Status: corev1.PersistentVolumeClaimStatus{
			Phase: corev1.ClaimBound,
			Capacity: corev1.ResourceList{
				"storage": resource.MustParse(capacity),
			},
			Conditions: nil,
		},
	}
}

func Get2SvcsOfDiffTypes() *corev1.ServiceList {
	svc1 := GetSvc("svc1", "foo", WithClusterIP)
	svc2 := GetSvc("svc2", "foo", WithLoadBalancer)
	return &corev1.ServiceList{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "ServiceList",
			APIVersion: "v1",
		},
		Items: []corev1.Service{
			*svc1, *svc2,
		},
	}
}

func GetSvcsWithLoadBalancers() *corev1.ServiceList {
	svc1 := GetSvc("svc1", "foo", WithLoadBalancer)
	svc2 := GetSvc("svc2", "bar", WithLoadBalancer)
	return &corev1.ServiceList{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "ServiceList",
			APIVersion: "v1",
		},
		Items: []corev1.Service{
			*svc1, *svc2,
		},
	}
}

type svcOpts func(service *corev1.Service)

func GetSvc(name, ns string, opts ...svcOpts) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}

	for _, opt := range opts {
		opt(svc)
	}

	return svc
}

func WithClusterIP(service *corev1.Service) {
	service.Spec = corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name:     "test",
				Protocol: "tcp",
				Port:     80,
			},
		},
	}
}

func WithLoadBalancer(service *corev1.Service) {
	service.Spec = corev1.ServiceSpec{
		Type: "LoadBalancer",
	}
}

func NewKCPStoredSecret(shootName, kubeconfigVal string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fmt.Sprintf("kubeconfig-%s", shootName),
			Namespace: "kcp-system",
		},
		Data: map[string][]byte{
			"config": []byte(kubeconfigVal),
		},
	}
}

func PrometheusGatherAndReturn(c prometheus.Collector, metricName string) (*dto.MetricFamily, error) {
	reg := prometheus.NewPedanticRegistry()
	if err := reg.Register(c); err != nil {
		return nil, err
	}
	mf, err := reg.Gather()
	if err != nil {
		return nil, err
	}
	for _, m := range mf {
		if m.GetName() == metricName {
			return m, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func PrometheusFilterLabelPair(pairs []*dto.LabelPair, name string) *dto.LabelPair {
	for _, p := range pairs {
		if p.GetName() == name {
			return p
		}
	}
	return nil
}
