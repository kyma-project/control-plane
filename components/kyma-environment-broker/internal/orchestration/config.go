package orchestration

type Config struct {
	KymaVersion        string `envconfig:"-"`
	KymaPreviewVersion string
	KubernetesVersion  string `envconfig:"-"`
	Namespace          string
	Name               string
}
