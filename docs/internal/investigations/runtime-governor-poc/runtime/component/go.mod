module github.com/kyma-project/control-plane/docs/internal/investigations/runtime-governor-poc/runtime/component

go 1.14

require (
	github.com/dapr/dapr v0.8.0
	github.com/hashicorp/go-multierror v1.0.0
	github.com/kyma-incubator/compass/components/director v0.0.0-20200716103056-05645f5ba9b8
	github.com/vrischmann/envconfig v1.2.0
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v0.17.4
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.0+incompatible
	k8s.io/client => github.com/kubernetes-client/go v0.0.0-20190928040339-c757968c4c36
)
