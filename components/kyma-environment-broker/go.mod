module github.com/kyma-project/control-plane/components/kyma-environment-broker

go 1.16

require (
	code.cloudfoundry.org/lager v2.0.0+incompatible
	github.com/99designs/gqlgen v0.9.3
	github.com/Azure/azure-sdk-for-go v54.2.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.18
	github.com/Azure/go-autorest/autorest/adal v0.9.13
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/Peripli/service-manager v0.19.4
	github.com/Peripli/service-manager-cli v1.11.8
	github.com/dlmiddlecote/sqlstats v1.0.2
	github.com/gardener/gardener v1.23.0
	github.com/gocraft/dbr v0.0.0-20190714181702-8114670a83bd
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/uuid v1.2.0
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/kyma-incubator/compass/components/director v0.0.0-20200813093525-96b1a733a11b
	github.com/kyma-project/control-plane/components/provisioner v0.0.0-20210527140555-2d0735d391e3
	github.com/kyma-project/kyma/components/kyma-operator v0.0.0-20201117100007-62918ff463e5
	github.com/lib/pq v1.10.2
	github.com/machinebox/graphql v0.2.3-0.20181106130121-3a9253180225
	github.com/matryer/is v1.4.0
	github.com/pivotal-cf/brokerapi/v8 v8.0.1-0.20210524135831-3563fe51db34
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.10.0
	github.com/sebdah/goldie/v2 v2.5.3
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/afero v1.5.1
	github.com/stretchr/testify v1.7.0
	github.com/testcontainers/testcontainers-go v0.11.0
	github.com/vburenin/nsync v0.0.0-20160822015540-9a75d1c80410
	github.com/vrischmann/envconfig v1.3.0
	golang.org/x/mod v0.4.2
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.6
	k8s.io/apimachinery v0.20.6
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	sigs.k8s.io/controller-runtime v0.8.3
)

replace (
	// NOTE: some dependencies require old style client-go version k8s.io/client-go@v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	// github.com/gardener/hvpa-controller, github.com/kyma-project/kyma/components/compass-runtime-agent, github.com/kyma-project/control-plane/components/provisioner, github.com/gardener/gardener
	k8s.io/api => k8s.io/api v0.19.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.0
	k8s.io/client-go => k8s.io/client-go v0.19.0
)
