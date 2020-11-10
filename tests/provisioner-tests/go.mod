module github.com/kyma-project/control-plane/tests/provisioner-tests

go 1.14

require (
	cloud.google.com/go v0.52.0 // indirect
	github.com/99designs/gqlgen v0.10.2 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/avast/retry-go v2.6.0+incompatible
	github.com/google/uuid v1.1.1
	github.com/googleapis/gnostic v0.4.0 // indirect
	github.com/hashicorp/go-multierror v1.1.0 // indirect
	github.com/kyma-incubator/compass/components/director v0.0.0-20200903080103-6ec34d89c49a
	github.com/kyma-project/control-plane/components/provisioner v0.0.0-20200916150531-dbc988100060
	github.com/kyma-project/kyma/components/kyma-operator v0.0.0-20200902131640-31c29c8feb0c
	github.com/machinebox/graphql v0.2.3-0.20181106130121-3a9253180225
	github.com/matryer/is v1.4.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	github.com/vektah/gqlparser v1.2.1 // indirect
	github.com/vrischmann/envconfig v1.3.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	k8s.io/api v0.18.10
	k8s.io/apimachinery v0.18.10
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

replace (
	github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8
	github.com/containerd/containerd => github.com/containerd/containerd v1.4.1
	github.com/coreos/etcd => go.etcd.io/etcd v0.5.0-alpha.5.0.20200824191128-ae9734ed278b
	github.com/kyma-project/control-plane/components/provisioner => github.com/Maladie/control-plane/components/provisioner v0.0.0-20201109133615-89df11747893
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc92
	golang.org/x/text => golang.org/x/text v0.3.3
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible => k8s.io/client-go v0.18.8
)
