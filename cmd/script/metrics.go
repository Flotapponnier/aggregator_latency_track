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
}

type AggregatorMetrics struct {
	Latency *prometheus.GaugeVec
	Trades  *prometheus.CounterVec
	Volume  *prometheus.GaugeVec
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
		Trades: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_trades_total", aggregator),
				Help: fmt.Sprintf("Total number of trades processed by %s", aggregator),
			},
			[]string{"chain", "type"},
		),
		Volume: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: fmt.Sprintf("%s_trade_volume_usd", aggregator),
				Help: fmt.Sprintf("Last trade volume in USD for %s", aggregator),
			},
			[]string{"chain"},
		),
	}

	prometheus.MustRegister(metrics.Latency)
	prometheus.MustRegister(metrics.Trades)
	prometheus.MustRegister(metrics.Volume)

	metricsRegistry[aggregator] = metrics
	return metrics
}

func RecordLatency(aggregator string, chain string, latencyMs float64) {
	metrics := GetOrCreateMetrics(aggregator)
	metrics.Latency.WithLabelValues(chain).Set(latencyMs)

	// Also record to the combined metric for easy comparison
	allAggregatorLatency.WithLabelValues(aggregator, chain).Set(latencyMs)
}

func RecordTrade(aggregator string, chain string, tradeType string, volumeUSD float64) {
	metrics := GetOrCreateMetrics(aggregator)
	metrics.Trades.WithLabelValues(chain, tradeType).Inc()
	metrics.Volume.WithLabelValues(chain).Set(volumeUSD)
}

func StartMetricsServer(addr string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(addr, nil)
}
