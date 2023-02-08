module github.com/kyma-project/control-plane/components/kubeconfig-service

go 1.19

require (
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/fsnotify/fsnotify v1.6.0
	github.com/gorilla/mux v1.8.0
	github.com/kyma-project/control-plane/components/provisioner v0.0.0-20220427105742-42ddda791e49
	github.com/machinebox/graphql v0.2.3-0.20181106130121-3a9253180225
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.9.0
	github.com/smartystreets/goconvey v1.7.2
	github.com/stretchr/testify v1.8.1
	github.com/vrischmann/envconfig v1.3.0
	golang.org/x/oauth2 v0.0.0-20220223155221-ee480838109b
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apiserver v0.25.2
	k8s.io/kubernetes v1.26.1
)

require (
	k8s.io/api v0.25.2
	k8s.io/apimachinery v0.25.2
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

require (
	github.com/99designs/gqlgen v0.11.3 // indirect
	github.com/agnivade/levenshtein v1.1.0 // indirect
	github.com/coreos/go-oidc v2.1.0+incompatible // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.9.0 // indirect
	github.com/evanphx/json-patch v4.12.0+incompatible // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/gnostic v0.5.7-v3refs // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/kyma-incubator/compass/components/director v0.0.0-20220125140301-bfb0c437ad31 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/onrik/logrus v0.9.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/pquerna/cachecontrol v0.1.0 // indirect
	github.com/smartystreets/assertions v1.2.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/vektah/gqlparser/v2 v2.1.0 // indirect
	golang.org/x/crypto v0.1.0 // indirect
	golang.org/x/net v0.3.1-0.20221206200815-1e63c2f08a10 // indirect
	golang.org/x/sys v0.3.0 // indirect
	golang.org/x/term v0.3.0 // indirect
	golang.org/x/text v0.5.0 // indirect
	golang.org/x/time v0.0.0-20220210224613-90d013bbcef8 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/square/go-jose.v2 v2.5.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/klog/v2 v2.80.1 // indirect
	k8s.io/kube-openapi v0.0.0-20221012153701-172d655c2280 // indirect
	k8s.io/utils v0.0.0-20221107191617-1a15be271d1d // indirect
	sigs.k8s.io/json v0.0.0-20220713155537-f223a00ba0e2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace (
	github.com/docker/distribution => github.com/docker/distribution v2.8.1+incompatible //CVE-2021-41190
	github.com/emicklei/go-restful => github.com/emicklei/go-restful/v3 v3.9.0 // CVE-2022-1996
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.1.3 // CVE-2021-43784
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.12.1 // CVE-2022-21698
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20220315160706-3147a52a75dd // CVE-2022-27191
	golang.org/x/net => golang.org/x/net v0.0.0-20221014081412-f15817d10f9b // CVE-2022-27664
	golang.org/x/text => golang.org/x/text v0.3.8 // CVE-2022-32149
	k8s.io/api => k8s.io/api v0.25.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.25.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.25.2
	k8s.io/apiserver => k8s.io/apiserver v0.25.2
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.25.2
	k8s.io/client-go => k8s.io/client-go v0.25.2
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.25.2
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.25.2
	k8s.io/code-generator => k8s.io/code-generator v0.25.2
	k8s.io/component-base => k8s.io/component-base v0.25.2
	k8s.io/component-helpers => k8s.io/component-helpers v0.25.2
	k8s.io/controller-manager => k8s.io/controller-manager v0.25.2
	k8s.io/cri-api => k8s.io/cri-api v0.25.2
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.25.2
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.25.2
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.25.2
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.25.2
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.25.2
	k8s.io/kubectl => k8s.io/kubectl v0.25.2
	k8s.io/kubelet => k8s.io/kubelet v0.25.2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.25.2
	k8s.io/metrics => k8s.io/metrics v0.25.2
	k8s.io/mount-utils => k8s.io/mount-utils v0.25.2
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.25.2
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.25.2
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.25.2
	k8s.io/sample-controller => k8s.io/sample-controller v0.25.2
)
