module github.com/kyma-project/control-plane/tools/cli

go 1.18

require (
	github.com/golang/mock v1.6.0
	github.com/int128/kubelogin v1.25.1
	github.com/kyma-project/control-plane/components/kubeconfig-service v0.0.0-20201211152036-9bdabffd55fb
	github.com/kyma-project/control-plane/components/kyma-environment-broker v0.0.0
	github.com/kyma-project/control-plane/components/reconciler v0.0.0
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.3.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/stretchr/testify v1.7.1
	golang.org/x/mod v0.6.0-dev.0.20220106191415-9b9b3d81d5e3
	golang.org/x/net v0.0.0-20220418201149-a630d4f3e7a2
	golang.org/x/oauth2 v0.0.0-20220411215720-9780585627b5
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.23.5
	k8s.io/apimachinery v0.23.5
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

require (
	cloud.google.com/go v0.99.0 // indirect
	github.com/99designs/gqlgen v0.17.5 // indirect
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/alexflint/go-filemutex v1.1.0 // indirect
	github.com/coreos/go-oidc/v3 v3.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/deepmap/oapi-codegen v1.8.2 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/gardener/gardener v1.45.0 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/google/wire v0.5.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/int128/listener v1.1.0 // indirect
	github.com/int128/oauth2cli v1.14.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kyma-incubator/compass/components/director v0.0.0-20220419124829-699e6c990877 // indirect
	github.com/kyma-project/control-plane/components/provisioner v0.0.0 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/onrik/logrus v0.9.0 // indirect
	github.com/pelletier/go-toml v1.9.4 // indirect
	github.com/pivotal-cf/brokerapi/v8 v8.2.1 // indirect
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/afero v1.8.2 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	github.com/vektah/gqlparser/v2 v2.4.2 // indirect
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20220412211240-33da011f77ad // indirect
	golang.org/x/term v0.0.0-20220411215600-e5f449aeb171 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20220411224347-583f2d630306 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.66.2 // indirect
	gopkg.in/square/go-jose.v2 v2.5.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/klog/v2 v2.60.1 // indirect
	k8s.io/kube-openapi v0.0.0-20220310132336-3f90b8c54bbb // indirect
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9 // indirect
	sigs.k8s.io/json v0.0.0-20211208200746-9f7c6b3444d2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace (
	github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8
	github.com/kyma-project/control-plane/components/kyma-environment-broker => ../../components/kyma-environment-broker
	github.com/kyma-project/control-plane/components/provisioner => ../../components/provisioner
	github.com/kyma-project/control-plane/components/reconciler => ../../components/reconciler
	k8s.io/api => k8s.io/api v0.23.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.23.5
	k8s.io/apiserver => k8s.io/apiserver v0.23.5
	k8s.io/client-go => k8s.io/client-go v0.23.5
)
