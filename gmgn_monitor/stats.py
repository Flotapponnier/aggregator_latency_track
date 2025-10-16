"""
Statistics tracking for GMGN monitor
Tracks min/max/avg latency per event type and chain
"""

from colorama import Fore, Style
import utils


class Statistics:
    """Statistics tracker for GMGN events"""

    def __init__(self):
        self.swap_count = 0
        self.pool_count = 0
        self.total_swap_latency = 0
        self.total_pool_latency = 0
        self.min_swap_latency = float('inf')
        self.max_swap_latency = 0
        self.min_pool_latency = float('inf')
        self.max_pool_latency = 0
        self.by_chain = {}

    def update(self, event_type: str, chain: str, latency_ms: float):
        """
        Update statistics with new event

        Args:
            event_type: "swap" or "pool"
            chain: Blockchain name
            latency_ms: Latency in milliseconds
        """
        if event_type == 'swap':
            self.swap_count += 1
            self.total_swap_latency += latency_ms
            self.min_swap_latency = min(self.min_swap_latency, latency_ms)
            self.max_swap_latency = max(self.max_swap_latency, latency_ms)
        elif event_type == 'pool':
            self.pool_count += 1
            self.total_pool_latency += latency_ms
            self.min_pool_latency = min(self.min_pool_latency, latency_ms)
            self.max_pool_latency = max(self.max_pool_latency, latency_ms)

        # Per-chain stats
        if chain not in self.by_chain:
            self.by_chain[chain] = {
                'swap_count': 0,
                'pool_count': 0,
                'total_latency': 0
            }

        self.by_chain[chain][f'{event_type}_count'] += 1
        self.by_chain[chain]['total_latency'] += latency_ms

    def print_summary(self):
        """Print statistics summary"""
        print(f"\n{Fore.CYAN}{'='*70}")
        print(f"📊 GMGN Monitor Statistics{Style.RESET_ALL}")
        print(f"{Fore.CYAN}{'='*70}{Style.RESET_ALL}")

        # Swap stats
        if self.swap_count > 0:
            avg_swap = self.total_swap_latency / self.swap_count
            print(f"\n{Fore.YELLOW}Swap Trades:{Style.RESET_ALL}")
            print(f"  Total: {self.swap_count}")
            print(f"  Avg Latency: {utils.format_latency_color(avg_swap)}")
            print(f"  Min: {int(self.min_swap_latency)}ms")
            print(f"  Max: {int(self.max_swap_latency)}ms")

        # Pool stats
        if self.pool_count > 0:
            avg_pool = self.total_pool_latency / self.pool_count
            print(f"\n{Fore.YELLOW}New Pools:{Style.RESET_ALL}")
            print(f"  Total: {self.pool_count}")
            print(f"  Avg Latency: {utils.format_latency_color(avg_pool)}")
            print(f"  Min: {int(self.min_pool_latency)}ms")
            print(f"  Max: {int(self.max_pool_latency)}ms")

        # Per-chain stats
        if self.by_chain:
            print(f"\n{Fore.YELLOW}Per-Chain:{Style.RESET_ALL}")
            for chain, data in self.by_chain.items():
                total = data['swap_count'] + data['pool_count']
                if total > 0:
                    avg = data['total_latency'] / total
                    print(f"  {chain}: {total} events, Avg: {utils.format_latency_color(avg)}")

        print(f"{Fore.CYAN}{'='*70}{Style.RESET_ALL}\n")
