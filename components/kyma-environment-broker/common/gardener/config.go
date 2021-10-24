package gardener

import (
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/internal"
)

type Config struct {
	Project        string `envconfig:"default=gardenerProject"`
	ShootDomain    string `envconfig:"optional"`
	KubeconfigPath string `envconfig:"default=./dev/kubeconfig.yaml"`
	DNSProviders   internal.DNSProvidersData
}
