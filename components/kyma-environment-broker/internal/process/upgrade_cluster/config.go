package upgrade_cluster

import "time"

type TimeSchedule struct {
	Retry                 time.Duration
	StatusCheck           time.Duration
	UpgradeClusterTimeout time.Duration
}
