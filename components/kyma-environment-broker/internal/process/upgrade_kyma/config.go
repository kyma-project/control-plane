package upgrade_kyma

import "time"

type TimeSchedule struct {
	Retry              time.Duration
	StatusCheck        time.Duration
	UpgradeKymaTimeout time.Duration
}
