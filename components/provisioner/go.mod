module github.com/kyma-project/control-plane/components/provisioner

go 1.16

require (
	github.com/99designs/gqlgen v0.9.3
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/gardener/gardener v1.23.0
	github.com/gocraft/dbr/v2 v2.6.3
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.7.4
	github.com/kubernetes-sigs/service-catalog v0.3.1
	github.com/kyma-incubator/compass/components/director v0.0.0-20200813093525-96b1a733a11b
	github.com/kyma-incubator/hydroform/install v0.0.0-20210525111154-8fe3a378654f
	github.com/kyma-incubator/hydroform/parallel-install v0.0.0-20210610074205-bbaf6958f1a5
	github.com/kyma-project/kyma/components/compass-runtime-agent v0.0.0-20200902131640-31c29c8feb0c
	github.com/kyma-project/kyma/components/kyma-operator v0.0.0-20201117100007-62918ff463e5
	github.com/lib/pq v1.9.0
	github.com/matryer/is v1.2.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/otiai10/copy v1.5.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.10.0
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.7.0
	github.com/testcontainers/testcontainers-go v0.7.0
	github.com/vektah/gqlparser v1.2.0
	github.com/vrischmann/envconfig v1.3.0
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.20.6
	k8s.io/apiextensions-apiserver v0.20.6
	k8s.io/apimachinery v0.20.6
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	sigs.k8s.io/controller-runtime v0.8.3
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible
	github.com/Microsoft/hcsshim => github.com/Microsoft/hcsshim v0.8.14
	github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8
	github.com/containerd/containerd => github.com/containerd/containerd v1.4.4
	github.com/coreos/etcd => github.com/coreos/etcd v3.3.25+incompatible
	//TODO: remove github.com/docker/distribution and github.com/docker/docker entries as Helm 3.5.4 got released
	//see https://github.com/helm/helm/issues/9354 + https://github.com/helm/helm/pull/9492
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	github.com/gophercloud/gophercloud => github.com/gophercloud/gophercloud v0.0.0-20190125124242-bb1ef8ce758c
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc92
	go.etcd.io/etcd => go.etcd.io/etcd v3.3.25+incompatible
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	golang.org/x/text => golang.org/x/text v0.3.3
	k8s.io/apiserver => k8s.io/apiserver v0.20.5
	k8s.io/client-go => k8s.io/client-go v0.20.5
)
