package node

import kmccache "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/cache"

type ConfigInf interface {
	NewClient(kmccache.Record) (*Client, error)
}

type Config struct {
	kubeconfig string
}
