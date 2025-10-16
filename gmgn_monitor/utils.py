"""
Utility functions for GMGN monitor
Helper functions for chain mapping, latency calculation, formatting, etc.
"""

import time
from typing import Optional
from colorama import Fore, Style


# Chain mapping for consistent naming with Go monitors
CHAIN_MAP = {
    'sol': 'solana',
    'bsc': 'bnb',
    'base': 'base',
    'eth': 'ethereum',
    'Solana': 'solana',
    'BNB Chain': 'bnb',
    'Base': 'base',
    'Ethereum': 'ethereum'
}


def get_chain_name(chain_code: str) -> str:
    """
    Normalize chain name to match Go monitor naming convention

    Args:
        chain_code: Raw chain code from GMGN (e.g., "sol", "bsc")

    Returns:
        Normalized chain name (e.g., "solana", "bnb")
    """
    normalized = CHAIN_MAP.get(chain_code, chain_code.lower())
    return normalized


def calculate_latency(server_timestamp: Optional[int]) -> Optional[float]:
    """
    Calculate latency from server timestamp to now.

    Args:
        server_timestamp: Unix timestamp in seconds (not milliseconds)

    Returns:
        Latency in milliseconds or None if invalid
    """
    if not server_timestamp:
        return None

    try:
        now = time.time()
        latency_ms = (now - server_timestamp) * 1000

        # Filter out invalid latencies (negative or > 2 minutes)
        if latency_ms < 0 or latency_ms > 120000:
            return None

        return latency_ms
    except Exception:
        return None


def extract_timestamp(data: any) -> Optional[int]:
    """
    Extract timestamp from GMGN data.

    Args:
        data: GMGN message data (can be list or dict)

    Returns:
        Unix timestamp or None
    """
    if not data:
        return None

    # Handle array data
    if isinstance(data, list) and len(data) > 0:
        item = data[0]

        # Try "t" first (trade timestamp)
        if isinstance(item, dict) and "t" in item and item["t"]:
            return item["t"]

        # Try "ot" (open time for pools)
        if isinstance(item, dict) and "ot" in item and item["ot"]:
            return item["ot"]

        # Try nested pool data
        if isinstance(item, dict) and "p" in item and isinstance(item["p"], list) and len(item["p"]) > 0:
            pool = item["p"][0]
            if "ot" in pool:
                return pool["ot"]

    # Handle dict data
    if isinstance(data, dict):
        if "t" in data:
            return data["t"]
        if "ot" in data:
            return data["ot"]

    return None


def format_latency_color(latency_ms: float) -> str:
    """
    Format latency with color based on value

    Args:
        latency_ms: Latency in milliseconds

    Returns:
        Formatted string with ANSI color codes
    """
    if latency_ms < 100:
        return f"{Fore.GREEN}{int(latency_ms)}ms [Excellent]{Style.RESET_ALL}"
    elif latency_ms < 300:
        return f"{Fore.YELLOW}{int(latency_ms)}ms [Good]{Style.RESET_ALL}"
    elif latency_ms < 1000:
        return f"{Fore.LIGHTYELLOW_EX}{int(latency_ms)}ms [Medium]{Style.RESET_ALL}"
    else:
        return f"{Fore.RED}{int(latency_ms)}ms [Slow]{Style.RESET_ALL}"


def safe_float(val, default=0.0):
    """
    Safely convert value to float, handling None and strings

    Args:
        val: Value to convert
        default: Default value if conversion fails

    Returns:
        Float value or default
    """
    if val is None:
        return default
    try:
        return float(val)
    except (ValueError, TypeError):
        return default
