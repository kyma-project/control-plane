package gardener

type Config struct {
	Project        string           `envconfig:"default=gardenerProject"`
	ShootDomain    string           `envconfig:"optional"`
	KubeconfigPath string           `envconfig:"default=./dev/kubeconfig.yaml"`
	DNSProviders   DNSProvidersData `envconfig:"-"`
}

type DNSProvidersData struct {
	Providers []DNSProviderData `json:"providers" yaml:"providers"`
}

type DNSProviderData struct {
	DomainsInclude []string `json:"domainsInclude" yaml:"domainsInclude"`
	Primary        bool     `json:"primary" yaml:"primary"`
	SecretName     string   `json:"secretName" yaml:"secretName"`
	Type           string   `json:"type" yaml:"type"`
}
