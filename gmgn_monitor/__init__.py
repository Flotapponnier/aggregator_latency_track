"""
GMGN Monitor Package
WebSocket-based latency tracker for GMGN aggregator
"""

__version__ = "1.0.0"
__author__ = "Aggregator Latency Tracker"

from .config import Config
from .metrics import init_metrics, record_latency, record_pool_discovery_latency, start_metrics_server
from .gmgn_websocket import run_gmgn_monitor

__all__ = [
    'Config',
    'init_metrics',
    'record_latency',
    'record_pool_discovery_latency',
    'start_metrics_server',
    'run_gmgn_monitor',
]
