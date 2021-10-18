module github.com/kyma-project/control-plane/tools/cli

go 1.16

require (
	github.com/int128/kubelogin v1.22.0
	github.com/kyma-project/control-plane/components/kubeconfig-service v0.0.0-20201211152036-9bdabffd55fb
	github.com/kyma-project/control-plane/components/kyma-environment-broker v0.0.0-20210601124357-126439534aac
	github.com/kyma-project/control-plane/components/reconciler v0.0.0
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/mod v0.4.2
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

replace (
	github.com/99designs/gqlgen => github.com/99designs/gqlgen v0.9.3
	github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8
	github.com/gardener/gardener => github.com/gardener/gardener v1.2.3
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.0
	github.com/kyma-incubator/compass/components/director => github.com/kyma-incubator/compass/components/director v0.0.0-20210329081251-209fb6d91e72
	github.com/kyma-project/control-plane/components/reconciler => ../../components/reconciler
	k8s.io/api => k8s.io/api v0.17.17
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.17
	k8s.io/apiserver => k8s.io/apiserver v0.17.17
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible => k8s.io/client-go v0.17.17
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29
)
