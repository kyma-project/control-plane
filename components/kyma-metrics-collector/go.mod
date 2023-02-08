module github.com/kyma-project/control-plane/components/kyma-metrics-collector

go 1.19

require (
	github.com/gardener/gardener v1.63.1
	github.com/gardener/gardener-extension-provider-aws v1.41.0
	github.com/gardener/gardener-extension-provider-azure v1.33.0
	github.com/gardener/gardener-extension-provider-gcp v1.27.1
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kyma-project/control-plane/components/kyma-environment-broker v0.0.0-20221108071341-0e3dd93e1f50
	github.com/onsi/gomega v1.22.1
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.13.1
	go.uber.org/zap v1.24.0
	k8s.io/api v0.25.2
	k8s.io/apimachinery v0.25.2
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

require (
	github.com/99designs/gqlgen v0.17.20 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kyma-incubator/compass/components/director v0.0.0-20220706110254-3d5dce79e48d // indirect
	github.com/kyma-project/control-plane/components/provisioner v0.0.0-20220929072045-bfb8d6dac310 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/onrik/logrus v0.9.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/vektah/gqlparser/v2 v2.5.1 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	golang.org/x/net v0.1.0 // indirect
	golang.org/x/oauth2 v0.0.0-20220630143837-2104d58473e0 // indirect
	golang.org/x/sys v0.1.0 // indirect
	golang.org/x/term v0.0.0-20220526004731-065cf7ba2467 // indirect
	golang.org/x/text v0.4.0 // indirect
	golang.org/x/time v0.0.0-20220609170525-579cf78fd858 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/klog/v2 v2.80.1 // indirect
	k8s.io/kube-openapi v0.0.0-20220803162953-67bda5d908f1 // indirect
	k8s.io/utils v0.0.0-20220728103510-ee6ede2d64ed // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace (
	k8s.io/api => k8s.io/api v0.22.12
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.22.12
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.12
	k8s.io/apiserver => k8s.io/apiserver v0.22.12
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.22.12
	k8s.io/client-go => k8s.io/client-go v0.22.12
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.22.12
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.22.12
	k8s.io/code-generator => k8s.io/code-generator v0.22.12
	k8s.io/component-base => k8s.io/component-base v0.22.12
	k8s.io/component-helpers => k8s.io/component-helpers v0.22.12
	k8s.io/controller-manager => k8s.io/controller-manager v0.22.12
	k8s.io/cri-api => k8s.io/cri-api v0.22.12
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.22.12
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.22.12
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.22.12
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.22.12
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.22.12
	k8s.io/kubectl => k8s.io/kubectl v0.22.12
	k8s.io/kubelet => k8s.io/kubelet v0.22.12
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.22.12
	k8s.io/metrics => k8s.io/metrics v0.22.12
	k8s.io/mount-utils => k8s.io/mount-utils v0.22.12
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.22.12
)

replace (
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.12.2
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20220518034528-6f7dac969898
	golang.org/x/net => golang.org/x/net v0.0.0-20220418201149-a630d4f3e7a2
	k8s.io/helm => k8s.io/helm v2.16.1+incompatible
	k8s.io/kubernetes => k8s.io/kubernetes v1.20.11
)

replace k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20220627174259-011e075b9cb8 //this fixes cve-2022-1996

exclude (
	github.com/emicklei/go-restful v2.9.5+incompatible //this fixes cve-2022-1996
	github.com/emicklei/go-restful v2.9.6+incompatible //this fixes cve-2022-1996
)
