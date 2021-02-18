module github.com/kyma-project/control-plane/components/metris

go 1.14

require (
	contrib.go.opencensus.io/exporter/zipkin v0.1.2
	github.com/Azure/azure-sdk-for-go v48.2.0+incompatible
	github.com/Azure/azure-sdk-for-go/sdk/to v0.1.2
	github.com/Azure/go-autorest/autorest v0.11.12
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.3
	github.com/Azure/go-autorest/autorest/date v0.3.0
	github.com/Azure/go-autorest/tracing/opencensus v0.1.0
	github.com/alecthomas/kong v0.2.9
	github.com/gardener/gardener v1.12.8
	github.com/kr/text v0.2.0 // indirect
	github.com/mitchellh/mapstructure v1.3.3
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/openzipkin/zipkin-go v0.2.5
	github.com/prometheus/client_golang v1.8.0
	github.com/stretchr/testify v1.6.1
	go.opencensus.io v0.22.5
	go.uber.org/zap v1.16.0
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
	k8s.io/api v0.18.10
	k8s.io/apimachinery v0.18.10
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/klog v1.0.0
)

replace (
	k8s.io/api => k8s.io/api v0.18.10
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.10
	k8s.io/apiserver => k8s.io/apiserver v0.18.10
	k8s.io/client-go => k8s.io/client-go v0.18.10
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.18.10
)
