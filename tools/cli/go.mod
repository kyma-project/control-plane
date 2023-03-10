module github.com/kyma-project/control-plane/tools/cli

go 1.18

require (
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.3.0
	github.com/int128/kubelogin v1.25.1
	github.com/kyma-project/control-plane/components/kubeconfig-service v0.0.0-20220704092952-6bdae76be31d
	github.com/kyma-project/control-plane/components/kyma-environment-broker v0.0.0-20230120130843-9e94e77792a5
	github.com/kyma-project/control-plane/components/reconciler v0.0.0
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.9.0
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.12.0
	github.com/stretchr/testify v1.8.1
	golang.org/x/mod v0.7.0
	golang.org/x/net v0.7.0
	golang.org/x/oauth2 v0.0.0-20220630143837-2104d58473e0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.26.1
	k8s.io/apimachinery v0.26.2
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

require (
	cloud.google.com/go/compute v1.7.0 // indirect
	github.com/99designs/gqlgen v0.17.22 // indirect
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/alexflint/go-filemutex v1.1.0 // indirect
	github.com/coreos/go-oidc v2.1.0+incompatible // indirect
	github.com/coreos/go-oidc/v3 v3.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/deepmap/oapi-codegen v1.8.2 // indirect
	github.com/emicklei/go-restful/v3 v3.9.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/swag v0.21.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/wire v0.5.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/int128/listener v1.1.0 // indirect
	github.com/int128/oauth2cli v1.14.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kyma-incubator/compass/components/director v0.0.0-20221021121045-dec2d997352a // indirect
	github.com/kyma-project/control-plane/components/provisioner v0.0.0-20221207124017-8b2cd58faf4b // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/onrik/logrus v0.9.0 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.0.1 // indirect
	github.com/pivotal-cf/brokerapi/v8 v8.2.3 // indirect
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/pquerna/cachecontrol v0.1.0 // indirect
	github.com/spf13/afero v1.9.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/subosito/gotenv v1.3.0 // indirect
	github.com/vektah/gqlparser/v2 v2.5.1 // indirect
	golang.org/x/crypto v0.1.0 // indirect
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/term v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.66.4 // indirect
	gopkg.in/square/go-jose.v2 v2.5.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiserver v0.25.2 // indirect
	k8s.io/klog/v2 v2.80.1 // indirect
	k8s.io/kube-openapi v0.0.0-20221012153701-172d655c2280 // indirect
	k8s.io/utils v0.0.0-20221128185143-99ec85e7a448 // indirect
	sigs.k8s.io/json v0.0.0-20220713155537-f223a00ba0e2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace (
	github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8
	github.com/kyma-project/control-plane/components/kubeconfig-service => ../../components/kubeconfig-service
	github.com/kyma-project/control-plane/components/provisioner => ../../components/provisioner
	github.com/kyma-project/control-plane/components/reconciler => ../../components/reconciler
	golang.org/x/net => golang.org/x/net v0.7.0
	k8s.io/api => k8s.io/api v0.26.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.25.2
	k8s.io/apiserver => k8s.io/apiserver v0.25.2
	k8s.io/client-go => k8s.io/client-go v0.25.2
)
