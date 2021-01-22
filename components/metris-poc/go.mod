module github.com/kyma-project/control-plane/components/metris-poc

go 1.15

require (
	github.com/gardener/gardener v1.15.3
	github.com/gorilla/mux v1.8.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kyma-project/control-plane v0.0.0-20210122120316-efdace16a3fc
	github.com/kyma-project/kyma/components/event-publisher-proxy v0.0.0-20210121145257-56404c125b3d // indirect
	github.com/sirupsen/logrus v1.7.0
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

replace (
	k8s.io/api => k8s.io/api v0.19.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.7
	k8s.io/client-go => k8s.io/client-go v0.19.7
)
