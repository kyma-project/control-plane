module github.com/kyma-project/control-plane/components/kyma-metrics-collector

go 1.15

require (
	github.com/gardener/gardener v1.19.0
	github.com/gardener/gardener-extension-provider-aws v1.22.2
	github.com/gardener/gardener-extension-provider-azure v1.18.1
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.7.3
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kyma-project/control-plane v0.0.0-20210131083023-031b4c8683db
	github.com/onsi/gomega v1.10.5
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.8.0
	github.com/sirupsen/logrus v1.7.0
	golang.org/x/oauth2 v0.0.0-20210126194326-f9ce19ea3013 // indirect
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

replace (
	k8s.io/api => k8s.io/api v0.20.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.2
	k8s.io/client-go => k8s.io/client-go v0.20.2
)
