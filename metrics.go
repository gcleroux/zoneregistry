package zoneregistry

import (
	"sync"

	"github.com/coredns/coredns/plugin"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	queryCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: plugin.Namespace,
			Subsystem: pluginName,
			Name:      "query_count_total",
			Help:      "Total number of DNS queries handled by the ZoneRegistry plugin.",
		},
		[]string{"server", "zone"},
	)
	responseDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: plugin.Namespace,
		Subsystem: pluginName,
		Name:      "response_duration_seconds",
		Help:      "Histogram of response times for delegated DNS queries.",
		Buckets:   prometheus.DefBuckets, // Use default latency buckets
	}, []string{"server", "zone"},
	)
	healthyPeers = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: plugin.Namespace,
		Subsystem: pluginName,
		Name:      "healthy_peers",
		Help:      "Number of healthy peers",
	}, []string{"role"},
	)
	unhealthyPeers = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: plugin.Namespace,
		Subsystem: pluginName,
		Name:      "unhealthy_peers",
		Help:      "Number of unhealthy peers",
	}, []string{"role"},
	)
)

var once sync.Once
