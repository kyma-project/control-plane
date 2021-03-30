module github.com/kyma-project/control-plane/components/subscription-cleanup-job

go 1.13

require (
	github.com/Azure/azure-sdk-for-go v42.2.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.18
	github.com/Azure/go-autorest/autorest/adal v0.9.13
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.0 // indirect
	github.com/gardener/gardener v1.10.1-0.20200903060046-8bed4ed6c257
	github.com/kyma-project/control-plane v0.0.0-20210308123720-0b44eb87eaaa
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.5.1
	github.com/vrischmann/envconfig v1.2.0
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	k8s.io/api v0.19.6
	k8s.io/apimachinery v0.19.6
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

replace (
	github.com/dgrijalva/jwt-go => github.com/dgrijalva/jwt-go v1.0.2
	github.com/kyma-project/control-plane => github.com/franpog859/control-plane v0.0.0-20210308125010-7afa0e3bf7d8
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	golang.org/x/text => golang.org/x/text v0.3.3
	k8s.io/api => k8s.io/api v0.18.15
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.15
	k8s.io/client-go => k8s.io/client-go v0.18.15
)
