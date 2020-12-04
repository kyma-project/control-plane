package gardener

type Config struct {
	Project        string `envconfig:"default=gardenerProject"`
	ShootDomain    string
	KubeconfigPath string `envconfig:"default=./dev/kubeconfig.yaml"`
}
