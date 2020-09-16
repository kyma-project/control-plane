package upgrade_kyma

import "time"

type IntervalConfig struct {
	Retry              time.Duration
	StatusCheck        time.Duration
	UpgradeKymaTimeout time.Duration
}
