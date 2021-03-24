package cache

import "github.com/kyma-incubator/metris/pkg/edp"

type Record struct {
	SubAccountID string
	ShootName    string
	KubeConfig   string
	Metric       *edp.ConsumptionMetrics
}
