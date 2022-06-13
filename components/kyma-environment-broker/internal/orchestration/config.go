package orchestration

type Config struct {
	KymaVersion       string `envconfig:"-"`
	KubernetesVersion string `envconfig:"-"`
	Namespace         string
	Name              string
}
