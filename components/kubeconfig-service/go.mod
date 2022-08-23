module github.com/kyma-project/control-plane/components/kubeconfig-service

go 1.18

require (
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/fsnotify/fsnotify v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/kyma-project/control-plane/components/provisioner v0.0.0-20220427105742-42ddda791e49
	github.com/machinebox/graphql v0.2.3-0.20181106130121-3a9253180225
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/smartystreets/goconvey v1.6.4
	github.com/stretchr/testify v1.7.0
	github.com/vrischmann/envconfig v1.3.0
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apiserver v0.23.8
	k8s.io/kubernetes v1.23.8
)

require (
	k8s.io/api v0.23.8
	k8s.io/apimachinery v0.23.8
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

require (
	github.com/99designs/gqlgen v0.11.3 // indirect
	github.com/agnivade/levenshtein v1.1.0 // indirect
	github.com/coreos/go-oidc v2.1.0+incompatible // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/evanphx/json-patch v4.12.0+incompatible // indirect
	github.com/go-logr/logr v1.2.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/kyma-incubator/compass/components/director v0.0.0-20220125140301-bfb0c437ad31 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/onrik/logrus v0.9.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/pquerna/cachecontrol v0.0.0-20171018203845-0dec1b30a021 // indirect
	github.com/smartystreets/assertions v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/vektah/gqlparser/v2 v2.1.0 // indirect
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
	golang.org/x/net v0.0.0-20211209124913-491a49abca63 // indirect
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9 // indirect
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/square/go-jose.v2 v2.5.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/klog/v2 v2.30.0 // indirect
	k8s.io/kube-openapi v0.0.0-20211115234752-e816edb12b65 // indirect
	k8s.io/utils v0.0.0-20211116205334-6203023598ed // indirect
	sigs.k8s.io/json v0.0.0-20211020170558-c049b76a60c6 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace (
	github.com/docker/distribution => github.com/docker/distribution v2.8.1+incompatible //CVE-2021-41190
	github.com/emicklei/go-restful => github.com/emicklei/go-restful/v3 v3.8.0 // CVE-2022-1996
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.1.3 // CVE-2021-43784
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.12.1 // CVE-2022-21698
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20220315160706-3147a52a75dd // CVE-2022-27191
	k8s.io/api => k8s.io/api v0.23.8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.23.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.23.8
	k8s.io/apiserver => k8s.io/apiserver v0.23.8
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.23.8
	k8s.io/client-go => k8s.io/client-go v0.23.8
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.23.8
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.23.8
	k8s.io/code-generator => k8s.io/code-generator v0.23.8
	k8s.io/component-base => k8s.io/component-base v0.23.8
	k8s.io/component-helpers => k8s.io/component-helpers v0.23.8
	k8s.io/controller-manager => k8s.io/controller-manager v0.23.8
	k8s.io/cri-api => k8s.io/cri-api v0.23.8
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.23.8
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.23.8
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.23.8
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.23.8
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.23.8
	k8s.io/kubectl => k8s.io/kubectl v0.23.8
	k8s.io/kubelet => k8s.io/kubelet v0.23.8
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.23.8
	k8s.io/metrics => k8s.io/metrics v0.23.8
	k8s.io/mount-utils => k8s.io/mount-utils v0.23.8
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.23.8
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.23.8
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.23.8
	k8s.io/sample-controller => k8s.io/sample-controller v0.23.8
)
