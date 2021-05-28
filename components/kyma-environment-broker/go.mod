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
	github.com/go-logr/logr v0.2.1 // indirect
	github.com/gocraft/dbr v0.0.0-20190714181702-8114670a83bd
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/uuid v1.2.0
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/kyma-incubator/compass/components/director v0.0.0-20210416142045-25b90bbc9ee6
	github.com/kyma-project/control-plane/components/provisioner v0.0.0-20210527140555-2d0735d391e3
	github.com/kyma-project/kyma/components/kyma-operator v0.0.0-20201117100007-62918ff463e5
	github.com/lib/pq v1.10.2
	github.com/machinebox/graphql v0.2.3-0.20181106130121-3a9253180225
	github.com/matryer/is v1.4.0
	github.com/pivotal-cf/brokerapi/v7 v7.5.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.10.0
	github.com/sebdah/goldie/v2 v2.5.3
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/afero v1.5.1
	github.com/stretchr/testify v1.7.0
	github.com/testcontainers/testcontainers-go v0.9.0
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
	// NOTE: currently needs replace because of helm v3.5.2 deps
	// https://github.com/helm/helm/blob/167aac70832d3a384f65f9745335e9fb40169dc2/go.mod#L51-L54
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible

	// NOTE: github.com/kyma-project/control-plane/components/provisioner already depends on 1.10 but KEB is not ready yet
	github.com/gardener/gardener => github.com/gardener/gardener v1.0.4

	// NOTE: currently compass references no longer existing versions of some components and don't have valid go modules tag
	// go: github.com/kyma-incubator/compass/components/director@v0.0.0-20210517162920-9017d63d1185 requires
	//      github.com/kyma-incubator/compass/components/operations-controller@v0.0.0-20210416142045-25b90bbc9ee6 requires
	//      github.com/kyma-incubator/compass/components/system-broker@v0.0.0-20210301181003-c1c76083a015: invalid version: unknown revision c1c76083a015
	github.com/kyma-incubator/compass/components/connector => github.com/kyma-incubator/compass/components/connector v0.0.0-20210329081251-209fb6d91e72
	github.com/kyma-incubator/compass/components/director => github.com/kyma-incubator/compass/components/director v0.0.0-20210329081251-209fb6d91e72
	github.com/kyma-incubator/compass/components/operations-controller => github.com/kyma-incubator/compass/components/operations-controller v0.0.0-20210329081251-209fb6d91e72
	github.com/kyma-incubator/compass/components/system-broker => github.com/kyma-incubator/compass/components/system-broker v0.0.0-20210329081251-209fb6d91e72

	// NOTE: some dependencies require old style client-go version k8s.io/client-go@v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	// github.com/gardener/hvpa-controller, github.com/kyma-project/kyma/components/compass-runtime-agent, github.com/kyma-project/control-plane/components/provisioner, github.com/gardener/gardener
	k8s.io/api => k8s.io/api v0.17.17
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.17
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.17
	k8s.io/client-go => k8s.io/client-go v0.17.17
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29

	// NOTE: some dependencies already depend on sigs.k8s.io/controller-runtime@v0.8.3 but KEB is not ready yet
	// github.com/kyma-project/control-plane/components/provisioner and github.com/kyma-project/control-plane/components/kyma-environment-broker
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.5.14
)
