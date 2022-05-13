package routinepool

import (
	"github.com/ccheers/xpkg/stat/metric"
)

const (
	_metricNamespace = "sync"
	_metricSubSystem = "routinepool"
)

var (
	_metricQueueSize = metric.NewGaugeVec(&metric.GaugeVecOpts{
		Namespace: _metricNamespace,
		Subsystem: _metricSubSystem,
		Name:      "queue_len",
		Help:      "sync routinepool current queue size.",
		Labels:    []string{"name"},
	})

	_metricCount = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: _metricNamespace,
		Subsystem: _metricSubSystem,
		Name:      "process_count",
		Help:      "sync routinepool process task count",
		Labels:    []string{"name"},
	})
)
