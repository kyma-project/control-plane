package cache

import "github.com/kyma-project/control-plane/components/kyma-metrics-collector/pkg/edp"

type Record struct {
	SubAccountID string
	RuntimeID    string
	ShootName    string
	KubeConfig   string
	Metric       *edp.ConsumptionMetrics
}
