package gardener

import "github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"

type Config struct {
	Project        string                     `envconfig:"default=gardenerProject"`
	ShootDomain    string                     `envconfig:"optional"`
	KubeconfigPath string                     `envconfig:"default=./dev/kubeconfig.yaml"`
	DNSProvider    gqlschema.DNSProviderInput `envconfig:"optional"`
}
