module github.com/kyma-project/control-plane/tests/hibernation

go 1.15

require (
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/akgalwas/control-plane v0.0.0-20201221144741-db4af7e22cb6 // indirect
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/google/uuid v1.1.4
	github.com/kyma-project/control-plane/components/provisioner v0.0.0-20210107104259-4dc8c0864a8a
	github.com/kyma-project/control-plane/tests/provisioner-tests v0.0.0-20210107104259-4dc8c0864a8a
	github.com/kyma-project/kyma/components/kyma-operator v0.0.0-20201117100007-62918ff463e5
	github.com/machinebox/graphql v0.2.3-0.20181106130121-3a9253180225
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/vrischmann/envconfig v1.3.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	k8s.io/apimachinery v0.20.1
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

replace github.com/kyma-project/control-plane/components/provisioner => github.com/akgalwas/control-plane/components/provisioner v0.0.0-20201221144741-db4af7e22cb6
