package pvc

type ConfigInf interface {
	NewClient(string) (*Client, error)
}

type Config struct {
	kubeconfig string
}
