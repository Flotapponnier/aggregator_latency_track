#!/usr/bin/env python3
"""
GMGN Latency Monitor - Main Entry Point
Similar to cmd/script/main.go structure
"""

import asyncio
import sys
import signal
from colorama import init, Fore, Style

import config
import metrics
import gmgn_websocket


# Initialize colorama for cross-platform colored output
init(autoreset=True)


def print_banner(cfg):
    """Print startup banner with configuration"""
    print(f"{Fore.CYAN}{'='*70}")
    print(f"🚀 GMGN Latency Monitor")
    print(f"{'='*70}{Style.RESET_ALL}")
    print(f"Monitoring chains: {', '.join(cfg.chains)}")
    print(f"Metrics port: {cfg.metrics_port}")
    print(f"Debug mode: {cfg.debug}")
    print(f"{Fore.CYAN}{'='*70}{Style.RESET_ALL}\n")


async def main():
    """
    Main entry point
    Similar to Go's main() function
    """
    # Load configuration
    try:
        cfg = config.Config.load()
    except Exception as e:
        print(f"{Fore.RED}❌ Error loading configuration: {e}{Style.RESET_ALL}")
        sys.exit(1)

    # Print banner
    print_banner(cfg)

    # Initialize Prometheus metrics
    metrics.init_metrics()

    # Start Prometheus metrics server
    print(f"{Fore.GREEN}📊 Starting Prometheus metrics server on :{cfg.metrics_port}{Style.RESET_ALL}")
    try:
        metrics.start_metrics_server(cfg.metrics_port)
        print(f"{Fore.GREEN}✓ Metrics available at http://localhost:{cfg.metrics_port}/metrics{Style.RESET_ALL}\n")
    except Exception as e:
        print(f"{Fore.RED}❌ Failed to start metrics server: {e}{Style.RESET_ALL}")
        sys.exit(1)

    # Create stop event (similar to Go's stopChan)
    stop_event = asyncio.Event()

    # Setup signal handlers (similar to Go's signal.Notify)
    def signal_handler(signum, frame):
        print(f"\n\n{Fore.YELLOW}🛑 Shutting down GMGN monitor...{Style.RESET_ALL}")
        stop_event.set()

    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)

    # Run GMGN monitor (similar to Go's goroutine)
    try:
        await gmgn_websocket.run_gmgn_monitor(cfg, stop_event)
    except KeyboardInterrupt:
        print(f"\n\n{Fore.YELLOW}🛑 Shutting down GMGN monitor...{Style.RESET_ALL}")
    except Exception as e:
        print(f"{Fore.RED}❌ Fatal error: {e}{Style.RESET_ALL}")
        if cfg.debug:
            import traceback
            traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        pass  # Already handled by signal handler
    except Exception as e:
        print(f"{Fore.RED}❌ Fatal error: {e}{Style.RESET_ALL}")
        sys.exit(1)
