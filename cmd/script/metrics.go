package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricsRegistry = make(map[string]*AggregatorMetrics)
	metricsLock     sync.Mutex

	// Combined metric to track all aggregators in one place for easy comparison
	allAggregatorLatency *prometheus.GaugeVec

	// Pool discovery latency metric
	poolDiscoveryLatency *prometheus.GaugeVec
)

func init() {
	allAggregatorLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "all_aggregator_latency_milliseconds",
			Help: "Latency in milliseconds for all aggregators by blockchain and source",
		},
		[]string{"aggregator", "chain"},
	)
	prometheus.MustRegister(allAggregatorLatency)

	poolDiscoveryLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pool_discovery_latency_milliseconds",
			Help: "Time from pool creation on-chain to first trade detection (pool discovery latency)",
		},
		[]string{"aggregator", "chain"},
	)
	prometheus.MustRegister(poolDiscoveryLatency)
}

type AggregatorMetrics struct {
	Latency *prometheus.GaugeVec
}

func GetOrCreateMetrics(aggregator string) *AggregatorMetrics {
	metricsLock.Lock()
	defer metricsLock.Unlock()

	if metrics, exists := metricsRegistry[aggregator]; exists {
		return metrics
	}

	metrics := &AggregatorMetrics{
		Latency: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: fmt.Sprintf("%s_latency_milliseconds", aggregator),
				Help: fmt.Sprintf("Latency in milliseconds for %s by blockchain", aggregator),
			},
			[]string{"chain"},
		),
	}

	prometheus.MustRegister(metrics.Latency)

	metricsRegistry[aggregator] = metrics
	return metrics
}

func RecordLatency(aggregator string, chain string, latencyMs float64) {
	metrics := GetOrCreateMetrics(aggregator)
	metrics.Latency.WithLabelValues(chain).Set(latencyMs)

	// Also record to the combined metric for easy comparison
	allAggregatorLatency.WithLabelValues(aggregator, chain).Set(latencyMs)
}

func RecordPoolDiscoveryLatency(aggregator string, chain string, latencyMs float64) {
	poolDiscoveryLatency.WithLabelValues(aggregator, chain).Set(latencyMs)
}

func StartMetricsServer(addr string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(addr, nil)
}
