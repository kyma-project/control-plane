module github.com/kyma-project/control-plane/components/kyma-metrics-collector

go 1.17

require (
	github.com/gardener/gardener v1.45.1
	github.com/gardener/gardener-extension-provider-aws v1.35.0
	github.com/gardener/gardener-extension-provider-azure v1.26.3
	github.com/gardener/gardener-extension-provider-gcp v1.22.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kyma-project/control-plane/components/kyma-environment-broker v0.0.0-20220504101926-5384b7ad1db0
	github.com/onsi/gomega v1.19.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.1
	go.uber.org/zap v1.21.0
	k8s.io/api v0.24.0
	k8s.io/apimachinery v0.24.0
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

require (
	github.com/99designs/gqlgen v0.17.5 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kyma-incubator/compass/components/director v0.0.0-20220419124829-699e6c990877 // indirect
	github.com/kyma-project/control-plane/components/provisioner v0.0.0-20220420073630-1208b41756a8 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/onrik/logrus v0.9.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.33.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/vektah/gqlparser/v2 v2.4.2 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/net v0.0.0-20220418201149-a630d4f3e7a2 // indirect
	golang.org/x/oauth2 v0.0.0-20220411215720-9780585627b5 // indirect
	golang.org/x/sys v0.0.0-20220412211240-33da011f77ad // indirect
	golang.org/x/term v0.0.0-20220411215600-e5f449aeb171 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20220411224347-583f2d630306 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/klog/v2 v2.60.1 // indirect
	k8s.io/kube-openapi v0.0.0-20220310132336-3f90b8c54bbb // indirect
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace (
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.12.1
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20220331220935-ae2d96664a29
	golang.org/x/net => golang.org/x/net v0.0.0-20220418201149-a630d4f3e7a2
	k8s.io/api => k8s.io/api v0.22.8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.22.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.8
	k8s.io/apiserver => k8s.io/apiserver v0.22.8
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.22.8
	k8s.io/client-go => k8s.io/client-go v0.22.8
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.22.8
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.22.8
	k8s.io/code-generator => k8s.io/code-generator v0.22.8
	k8s.io/component-base => k8s.io/component-base v0.22.8
	k8s.io/component-helpers => k8s.io/component-helpers v0.22.8
	k8s.io/controller-manager => k8s.io/controller-manager v0.22.8
	k8s.io/cri-api => k8s.io/cri-api v0.22.8
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.22.8
	k8s.io/helm => k8s.io/helm v2.16.1+incompatible
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.22.8
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.22.8
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.22.8
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.22.8
	k8s.io/kubectl => k8s.io/kubectl v0.22.8
	k8s.io/kubelet => k8s.io/kubelet v0.22.8
	k8s.io/kubernetes => k8s.io/kubernetes v1.20.11
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.22.8
	k8s.io/metrics => k8s.io/metrics v0.22.8
	k8s.io/mount-utils => k8s.io/mount-utils v0.22.8
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.22.8
)
