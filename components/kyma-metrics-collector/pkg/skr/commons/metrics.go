package commons

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	SuccessListingSVCLabel   = "success_listing_svc"
	SuccessListingPVCLabel   = "success_listing_pvc"
	SuccessListingNodesLabel = "success_listing_nodes"
	SuccessStatusLabel       = "success"
	CallsTotalLabel          = "calls_total"
	ListingNodesLabel        = "listing_nodes"
	ListingPVCLabel          = "listing_pvc"
	ListingSVCLabel          = "listing_svc"
)

var (
	TotalCalls = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kmc",
			Subsystem: "skr",
			Name:      "calls_total",
			Help:      "Total number of calls to SKR to get the metrics of the cluster.",
		},
		[]string{"status", "reason"},
	)
)
