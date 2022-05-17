module github.com/kyma-project/control-plane/components/provisioner

go 1.17

require (
	github.com/99designs/gqlgen v0.11.3
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/gardener/gardener v1.24.0
	github.com/gocraft/dbr/v2 v2.6.3
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/go-version v1.4.0
	github.com/kubernetes-sigs/service-catalog v0.3.0
	github.com/kyma-incubator/compass/components/director v0.0.0-20220125140301-bfb0c437ad31
	github.com/kyma-incubator/hydroform/install v0.0.0-20210525111154-8fe3a378654f
	github.com/kyma-project/kyma/components/compass-runtime-agent v0.0.0-20200902131640-31c29c8feb0c
	github.com/kyma-project/kyma/components/kyma-operator v0.0.0-20201117100007-62918ff463e5
	github.com/lib/pq v1.10.4
	github.com/matryer/is v1.4.0
	github.com/mitchellh/mapstructure v1.4.2
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.26.0
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/testcontainers/testcontainers-go v0.12.0
	github.com/vektah/gqlparser/v2 v2.1.0
	github.com/vrischmann/envconfig v1.3.0
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.20.7
	k8s.io/apiextensions-apiserver v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	sigs.k8s.io/controller-runtime v0.8.3
)

require (
	cloud.google.com/go v0.93.3 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/semver/v3 v3.1.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.2 // indirect
	github.com/Microsoft/go-winio v0.4.16-0.20201130162521-d1ffc52c7331 // indirect
	github.com/Microsoft/hcsshim v0.8.16 // indirect
	github.com/agnivade/levenshtein v1.1.0 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/containerd/cgroups v0.0.0-20200531161412-0dbf7f05ba59 // indirect
	github.com/containerd/containerd v1.3.4 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/distribution v2.8.0+incompatible // indirect
	github.com/docker/docker v20.10.12+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/evanphx/json-patch v4.9.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/go-logr/logr v0.4.0 // indirect
	github.com/go-logr/zapr v0.4.0 // indirect
	github.com/go-ozzo/ozzo-validation/v4 v4.3.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/googleapis/gnostic v0.5.1 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mattn/go-sqlite3 v1.14.6 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mitchellh/copystructure v1.1.2 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/moby/sys/mount v0.3.0 // indirect
	github.com/moby/sys/mountinfo v0.5.0 // indirect
	github.com/moby/term v0.0.0-20201216013528-df9cb8a40635 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/onrik/logrus v0.9.0 // indirect
	github.com/onsi/ginkgo v1.16.1 // indirect
	github.com/onsi/gomega v1.11.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/urfave/cli/v2 v2.1.1 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.uber.org/goleak v1.1.11-0.20210813005559-691160354723 // indirect
	go.uber.org/zap v1.19.1 // indirect
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/net v0.0.0-20211112202133-69e39bad7dc2 // indirect
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8 // indirect
	golang.org/x/sys v0.0.0-20211116061358-0a5406a5449c // indirect
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba // indirect
	golang.org/x/tools v0.1.7 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gomodules.xyz/jsonpatch/v2 v2.1.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20210828152312-66f60bf46e71 // indirect
	google.golang.org/grpc v1.40.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/alecthomas/kingpin.v2 v2.2.6 // indirect
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/component-base v0.20.7 // indirect
	k8s.io/klog v1.0.0 // indirect
	k8s.io/klog/v2 v2.4.0 // indirect
	k8s.io/kube-openapi v0.0.0-20201113171705-d219536bb9fd // indirect
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.0.3 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible
	github.com/Microsoft/hcsshim => github.com/Microsoft/hcsshim v0.8.14
	github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8
	github.com/containerd/containerd => github.com/containerd/containerd v1.4.4
	github.com/coreos/etcd => github.com/coreos/etcd v3.3.25+incompatible
	github.com/docker/distribution => github.com/docker/distribution v2.8.0+incompatible
	github.com/docker/docker => github.com/docker/docker v20.10.12+incompatible
	github.com/gophercloud/gophercloud => github.com/gophercloud/gophercloud v0.0.0-20190125124242-bb1ef8ce758c
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.1.2
	go.etcd.io/etcd => go.etcd.io/etcd v3.3.25+incompatible
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20220513210258-46612604a0f9
	golang.org/x/text => golang.org/x/text v0.3.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.7
	k8s.io/apiserver => k8s.io/apiserver v0.20.5
	k8s.io/client-go => k8s.io/client-go v0.20.5
)
