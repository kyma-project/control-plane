module github.com/kyma-project/control-plane/components/kubeconfig-service

go 1.17

require (
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/fsnotify/fsnotify v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/kyma-project/control-plane/components/provisioner v0.0.0-20211117114650-e8ce053b0aa4
	github.com/machinebox/graphql v0.2.3-0.20181106130121-3a9253180225
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/smartystreets/goconvey v1.6.4
	github.com/stretchr/testify v1.6.1
	github.com/vrischmann/envconfig v1.3.0
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apiserver v0.20.7
)

require (
	github.com/99designs/gqlgen v0.9.3 // indirect
	github.com/agnivade/levenshtein v1.0.3 // indirect
	github.com/coreos/go-oidc v2.1.0+incompatible // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v0.3.0 // indirect
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20181017120253-0766667cb4d1 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/kyma-incubator/compass/components/director v0.0.0-20200813093525-96b1a733a11b // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/pquerna/cachecontrol v0.0.0-20171018203845-0dec1b30a021 // indirect
	github.com/smartystreets/assertions v0.0.0-20180927180507-b2de0cb4f26d // indirect
	github.com/vektah/gqlparser v1.2.0 // indirect
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0 // indirect
	golang.org/x/net v0.0.0-20201202161906-c7110b5ffcbb // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	golang.org/x/text v0.3.4 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/square/go-jose.v2 v2.2.2 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	k8s.io/apimachinery v0.20.7 // indirect
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible // indirect
	k8s.io/klog/v2 v2.4.0 // indirect
)

replace (
	github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8
	golang.org/x/text => golang.org/x/text v0.3.3
)
