module github.com/kyma-project/control-plane/components/kubeconfig-service

go 1.13

require (
	github.com/avast/retry-go v2.6.0+incompatible
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gorilla/mux v1.7.4
	github.com/kyma-project/control-plane/components/provisioner v0.0.0-20200702142454-d5c043eb0dbe
	github.com/machinebox/graphql v0.2.3-0.20181106130121-3a9253180225
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/smartystreets/goconvey v1.6.4
	github.com/stretchr/testify v1.5.1
	github.com/vrischmann/envconfig v1.2.0
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/apiserver v0.18.10
)

replace (
	github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8

	k8s.io/apimachinery => k8s.io/apimachinery v0.18.10
)
