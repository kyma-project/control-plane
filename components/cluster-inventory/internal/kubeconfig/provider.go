package kubeconfig

type Provider struct {
}

func NewProvider() Provider {
	return Provider{}
}

func (p Provider) Fetch(shootName string) error {
	return nil
}
