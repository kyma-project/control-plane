package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/client-go/util/workqueue"
)

const (
	Namespace = "metris"
)

var (
	// Definition of metrics for provider queue
	workqueueDepthMetricVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: "workqueue",
			Name:      "depth",
			Help:      "Current depth of the work queue.",
		},
		[]string{"queue_name"},
	)
	workqueueAddsMetricVec = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "workqueue",
			Name:      "items_total",
			Help:      "Total number of items added to the work queue.",
		},
		[]string{"queue_name"},
	)
	workqueueLatencyMetricVec = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  Namespace,
			Subsystem:  "workqueue",
			Name:       "latency_seconds",
			Help:       "How long an item stays in the work queue.",
			Objectives: map[float64]float64{},
		},
		[]string{"queue_name"},
	)
	workqueueUnfinishedWorkSecondsMetricVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: "workqueue",
			Name:      "unfinished_work_seconds",
			Help:      "How long an item has remained unfinished in the work queue.",
		},
		[]string{"queue_name"},
	)
	workqueueLongestRunningProcessorMetricVec = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: "workqueue",
			Name:      "longest_running_processor_seconds",
			Help:      "Duration of the longest running processor in the work queue.",
		},
		[]string{"queue_name"},
	)
	workqueueWorkDurationMetricVec = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  Namespace,
			Subsystem:  "workqueue",
			Name:       "work_duration_seconds",
			Help:       "How long processing an item from the work queue takes.",
			Objectives: map[float64]float64{},
		},
		[]string{"queue_name"},
	)

	// register our prometheus metrics with the workqueue metric provider
	workqueueMetricsProviderInstance = new(WorkqueueMetricsProvider)

	// set the metrics provider for all work queues.
	_ = func() struct{} {
		workqueue.SetProvider(workqueueMetricsProviderInstance)
		return struct{}{}
	}()
)

// Definition of dummy metric used as a placeholder if we don't want to observe some data.
type noopMetric struct{}

func (noopMetric) Inc()            {}
func (noopMetric) Dec()            {}
func (noopMetric) Observe(float64) {}
func (noopMetric) Set(float64)     {}

// Definition of workqueue metrics provider definition
type WorkqueueMetricsProvider struct{}

func (f *WorkqueueMetricsProvider) NewDepthMetric(name string) workqueue.GaugeMetric {
	return workqueueDepthMetricVec.WithLabelValues(name)
}

func (f *WorkqueueMetricsProvider) NewAddsMetric(name string) workqueue.CounterMetric {
	return workqueueAddsMetricVec.WithLabelValues(name)
}

func (f *WorkqueueMetricsProvider) NewLatencyMetric(name string) workqueue.HistogramMetric {
	return workqueueLatencyMetricVec.WithLabelValues(name)
}

func (f *WorkqueueMetricsProvider) NewWorkDurationMetric(name string) workqueue.HistogramMetric {
	return workqueueWorkDurationMetricVec.WithLabelValues(name)
}

func (f *WorkqueueMetricsProvider) NewUnfinishedWorkSecondsMetric(name string) workqueue.SettableGaugeMetric {
	return workqueueUnfinishedWorkSecondsMetricVec.WithLabelValues(name)
}

func (f *WorkqueueMetricsProvider) NewLongestRunningProcessorSecondsMetric(name string) workqueue.SettableGaugeMetric {
	return workqueueLongestRunningProcessorMetricVec.WithLabelValues(name)
}

func (WorkqueueMetricsProvider) NewRetriesMetric(name string) workqueue.CounterMetric {
	// Retries are not used so the metric is omitted.
	return noopMetric{}
}
