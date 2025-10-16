"""
Prometheus metrics module for GMGN monitor
Similar to cmd/script/metrics.go
"""

from prometheus_client import start_http_server, Gauge


# Global metrics (similar to Go's package-level vars)
_swap_latency = None
_pool_discovery_latency = None
_all_aggregator_latency = None


def init_metrics():
    """Initialize Prometheus metrics (similar to Go's init())"""
    global _swap_latency, _pool_discovery_latency, _all_aggregator_latency

    _swap_latency = Gauge(
        'gmgn_latency_milliseconds',
        'Latency in milliseconds for GMGN swap trades by blockchain',
        ['chain']
    )

    _pool_discovery_latency = Gauge(
        'pool_discovery_latency_milliseconds',
        'Time from pool creation on-chain to first detection by GMGN',
        ['aggregator', 'chain']
    )

    _all_aggregator_latency = Gauge(
        'all_aggregator_latency_milliseconds',
        'Latency in milliseconds for all aggregators by blockchain and source',
        ['aggregator', 'chain']
    )


def record_latency(aggregator: str, chain: str, latency_ms: float):
    """
    Record latency metric
    Similar to Go's RecordLatency()

    Args:
        aggregator: Name of the aggregator (e.g., "gmgn")
        chain: Blockchain name (e.g., "solana", "base")
        latency_ms: Latency in milliseconds
    """
    # Filter out invalid values: negative or > 2 minutes (120000ms)
    if latency_ms < 0 or latency_ms > 120000:
        return

    # Record to aggregator-specific metric
    if _swap_latency:
        _swap_latency.labels(chain=chain).set(latency_ms)

    # Record to combined metric for easy comparison
    if _all_aggregator_latency:
        _all_aggregator_latency.labels(aggregator=aggregator, chain=chain).set(latency_ms)


def record_pool_discovery_latency(aggregator: str, chain: str, latency_ms: float):
    """
    Record pool discovery latency metric
    Similar to Go's RecordPoolDiscoveryLatency()

    Args:
        aggregator: Name of the aggregator (e.g., "gmgn")
        chain: Blockchain name (e.g., "solana", "base")
        latency_ms: Latency in milliseconds
    """
    # Filter out invalid values: negative or > 2 minutes (120000ms)
    if latency_ms < 0 or latency_ms > 120000:
        return

    if _pool_discovery_latency:
        _pool_discovery_latency.labels(aggregator=aggregator, chain=chain).set(latency_ms)

    # Also record to combined metric
    if _all_aggregator_latency:
        _all_aggregator_latency.labels(aggregator=aggregator, chain=chain).set(latency_ms)


def start_metrics_server(port: int):
    """
    Start Prometheus metrics HTTP server
    Similar to Go's StartMetricsServer()

    Args:
        port: Port number to listen on
    """
    start_http_server(port)
