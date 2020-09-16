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
	github.com/huandu/xstrings v1.3.0 // indirect
	github.com/kyma-incubator/compass/components/director v0.0.0-20200903080103-6ec34d89c49a
	github.com/kyma-project/control-plane/components/provisioner v0.0.0-20200819110923-1c7da9ea4eca
	github.com/kyma-project/kyma v0.5.1-0.20200416091733-68742a10ec23
	github.com/machinebox/graphql v0.2.3-0.20181106130121-3a9253180225
	github.com/matryer/is v1.4.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.5.1
	github.com/vektah/gqlparser v1.2.1 // indirect
	github.com/vrischmann/envconfig v1.2.0
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200121175148-a6ecf24a6d71
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

replace (
	github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8
	github.com/kyma-project/control-plane => github.com/rafalpotempa/control-plane v0.0.0-20200904095315-9b9d4c8f16d6
	golang.org/x/text => golang.org/x/text v0.3.3
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible => k8s.io/client-go v0.18.8
)
