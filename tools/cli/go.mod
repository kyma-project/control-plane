module github.com/kyma-project/control-plane/tools/cli

go 1.14

require (
	github.com/int128/kubelogin v1.22.0
	github.com/kyma-project/control-plane v0.0.0-20210422063135-dcde668f340b
	github.com/kyma-project/control-plane/components/kubeconfig-service v0.0.0-20201211152036-9bdabffd55fb
	github.com/kyma-project/control-plane/components/provisioner v0.0.0-20201211152036-9bdabffd55fb // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de
	github.com/pivotal-cf/brokerapi/v7 v7.4.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	golang.org/x/mod v0.3.0
	golang.org/x/oauth2 v0.0.0-20201109201403-9fd604954f58
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

replace (
	github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8
	github.com/gardener/gardener => github.com/gardener/gardener v1.2.3
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.3.1
	k8s.io/api => k8s.io/api v0.17.14
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.14
	k8s.io/apiserver => k8s.io/apiserver v0.17.14
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible => k8s.io/client-go v0.17.14
)
