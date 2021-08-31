module github.com/kyma-incubator/compass/components/connector

go 1.15

require (
	github.com/99designs/gqlgen v0.11.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/kyma-incubator/compass/components/director v0.0.0-20210820085625-1e07ac4da895
	github.com/machinebox/graphql v0.2.3-0.20181106130121-3a9253180225
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	github.com/vektah/gqlparser/v2 v2.1.0
	github.com/vektra/mockery/v2 v2.9.0 // indirect
	github.com/vrischmann/envconfig v1.3.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
)
