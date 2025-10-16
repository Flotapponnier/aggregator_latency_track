"""
GMGN WebSocket monitor
Handles connection, subscription, and message processing
Similar to cmd/script/codex_monitor.go structure
"""

import asyncio
import websockets
import json
import uuid
import time
from datetime import datetime
from typing import List
from colorama import Fore, Style

import utils
import metrics
import stats


# GMGN WebSocket configuration
GMGN_WS_URL = "wss://ws.gmgn.ai/quotation"


def create_subscription(channel: str, chain: str) -> dict:
    """
    Create a GMGN subscription message.

    Args:
        channel: Channel name ("new_pool_info", "new_pair_update")
        chain: Chain code ("sol", "bsc", "base")

    Returns:
        Subscription message dict
    """
    return {
        "action": "subscribe",
        "channel": channel,
        "f": "w",  # REQUIRED: format = websocket
        "id": uuid.uuid4().hex[:16],  # REQUIRED: max 16 chars
        "data": [{"chain": chain}],  # REQUIRED: array with dict
        "access_token": None,
        "retry": None
    }


def handle_swap_trade(data: List[dict], receive_time: float, stats_obj: stats.Statistics, debug: bool):
    """
    Handle new_pair_update (swap trade) messages

    Args:
        data: List of trade data from GMGN
        receive_time: Timestamp when message was received
        stats: Statistics tracker
        debug: Debug mode flag
    """
    if not data or len(data) == 0:
        return

    for trade in data:
        # Extract key fields
        symbol = trade.get('s', 'UNKNOWN')
        address = trade.get('a', '')
        price = trade.get('p', 0)
        market_cap = trade.get('mc', 0)
        volume_1h = trade.get('v1h', 0)
        buys_1h = trade.get('b1h', 0)
        sells_1h = trade.get('s1h', 0)
        chain_raw = trade.get('c', trade.get('n', 'unknown'))

        # Debug: Show full trade data when chain is unknown
        if chain_raw == 'unknown' and debug:
            print(f"[DEBUG] Unknown chain trade - Full data: {trade}")

            # Try to detect chain from address format
            if address.startswith('0x'):
                print(f"[DEBUG] EVM-style address detected: {address[:10]}...")
                if 'lc_ex' in trade:
                    print(f"[DEBUG] Exchange field: {trade.get('lc_ex')}")

        # Normalize chain name
        chain = utils.get_chain_name(chain_raw)

        # Calculate latency
        server_time = trade.get('t')
        latency = utils.calculate_latency(server_time)

        if latency is None:
            # Silently skip incomplete data (common at startup with backlog)
            continue

        # Update metrics
        metrics.record_latency("gmgn", chain, latency)

        # Update stats
        stats_obj.update('swap', chain, latency)

        # Format output
        timestamp = datetime.fromtimestamp(receive_time).strftime('%Y-%m-%d %H:%M:%S')
        trade_time = datetime.fromtimestamp(server_time).strftime('%H:%M:%S.%f')[:-3] if server_time else 'N/A'

        addr_short = address[:8] if len(address) > 8 else address

        # Debug output with raw timestamp
        if debug:
            print(f"\n[DEBUG] Raw timestamp: {server_time} | Trade time: {trade_time} | "
                  f"Receive time: {timestamp} | Lag: {int(latency)}ms")

        # Handle None values and convert strings to numbers safely
        price_val = utils.safe_float(price)
        mc_val = utils.safe_float(market_cap)
        vol_val = utils.safe_float(volume_1h)
        buy_val = int(utils.safe_float(buys_1h))
        sell_val = int(utils.safe_float(sells_1h))

        # Show ALL swaps - don't filter by data completeness
        if mc_val > 0 or vol_val > 0:
            print(f"[GMGN-SWAP][{timestamp}][{chain}] New swap! "
                  f"Token: {symbol} ({addr_short}...) | "
                  f"Price: ${price_val:.8f} | MC: ${mc_val:,.0f} | "
                  f"Vol 1h: ${vol_val:,.0f} | B/S: {buy_val}/{sell_val} | "
                  f"Trade time: {trade_time} | Lag: {utils.format_latency_color(latency)}")
        else:
            # Show minimal data swaps too (they have valid latency!)
            print(f"[GMGN-SWAP][{timestamp}][{chain}] Swap: {addr_short}... | "
                  f"Trade time: {trade_time} | Lag: {utils.format_latency_color(latency)}")


def handle_new_pool(data: List[dict], receive_time: float, stats_obj: stats.Statistics, debug: bool):
    """
    Handle new_pool_info messages

    Args:
        data: List of pool data from GMGN
        receive_time: Timestamp when message was received
        stats: Statistics tracker
        debug: Debug mode flag
    """
    if not data or len(data) == 0:
        return

    for pool_event in data:
        chain_raw = pool_event.get('c', pool_event.get('n', 'unknown'))
        chain = utils.get_chain_name(chain_raw)

        # Handle two formats:
        # Format 1: Nested pools array (Solana) - {'c': 'sol', 'p': [{pool1}, {pool2}]}
        # Format 2: Direct pool data (Base) - {pool data directly}
        pools = pool_event.get('p', [])

        # If no 'p' array, treat the entire event as a single pool
        if not pools and 'pa' in pool_event:
            pools = [pool_event]

        open_time = pool_event.get('ot')

        for pool in pools:
            exchange = pool.get('ex', 'unknown')
            pool_addr = pool.get('pa', '')
            initial_liquidity = pool.get('il', '0')
            pool_open_time = pool.get('ot', open_time)

            # Token info
            token_info = pool.get('bti', {})
            symbol = token_info.get('s', 'UNKNOWN')
            name = token_info.get('n', 'Unknown')
            price = token_info.get('p', 0)
            market_cap = token_info.get('mc', 0)
            volume_1h = token_info.get('v1h', 0)
            holder_count = token_info.get('hc', 0)

            # Calculate latency
            latency = utils.calculate_latency(pool_open_time)

            if latency is None:
                # Silently skip pools without valid timestamps
                continue

            # Update metrics
            metrics.record_pool_discovery_latency("gmgn", chain, latency)

            # Update stats
            stats_obj.update('pool', chain, latency)

            # Format output
            timestamp = datetime.fromtimestamp(receive_time).strftime('%Y-%m-%d %H:%M:%S')
            pool_time = datetime.fromtimestamp(pool_open_time).strftime('%H:%M:%S.%f')[:-3] if pool_open_time else 'N/A'

            pool_short = pool_addr[:8] if len(pool_addr) > 8 else pool_addr

            # Debug output
            if debug:
                print(f"\n[DEBUG] Raw timestamp: {pool_open_time} | Pool time: {pool_time} | "
                      f"Receive time: {timestamp} | Lag: {int(latency)}ms")

            # Handle None values and convert strings to numbers safely
            price_val = utils.safe_float(price)
            mc_val = utils.safe_float(market_cap)
            vol_val = utils.safe_float(volume_1h)
            holder_val = int(utils.safe_float(holder_count))
            liq_val = utils.safe_float(initial_liquidity)

            print(f"[GMGN-POOL][{timestamp}][{chain}] New pool detected! "
                  f"Token: {symbol} ({name}) | Exchange: {exchange} | "
                  f"Pool: {pool_short}... | Initial Liq: ${liq_val:,.2f} | "
                  f"Price: ${price_val:.8f} | MC: ${mc_val:,.0f} | "
                  f"Holders: {holder_val} | "
                  f"Pool time: {pool_time} | Lag: {utils.format_latency_color(latency)}")


async def listen_to_gmgn(websocket, stats_obj: stats.Statistics, debug: bool):
    """
    Listen to GMGN WebSocket messages
    Similar to handleWebSocketMessages in Go

    Args:
        websocket: WebSocket connection
        stats: Statistics tracker
        debug: Debug mode flag
    """
    message_count = 0

    try:
        async for message in websocket:
            receive_time = time.time()
            message_count += 1

            try:
                data = json.loads(message)
            except json.JSONDecodeError:
                if debug:
                    print(f"[DEBUG] Failed to parse JSON: {message[:100]}")
                continue

            channel = data.get('channel')
            payload = data.get('data')

            if not channel:
                if debug:
                    print(f"[DEBUG] No channel in message: {data}")
                continue

            # Handle ACK messages
            if channel == 'ack':
                print(f"{Fore.GREEN}✓ ACK received for subscription{Style.RESET_ALL}")
                continue

            # Handle swap trades
            if channel == 'new_pair_update':
                handle_swap_trade(payload, receive_time, stats_obj, debug)

            # Handle new pools
            elif channel == 'new_pool_info':
                handle_new_pool(payload, receive_time, stats_obj, debug)

            # Print stats every 50 messages
            if message_count % 50 == 0:
                stats_obj.print_summary()

    except websockets.exceptions.ConnectionClosed:
        print(f"{Fore.RED}❌ WebSocket connection closed{Style.RESET_ALL}")
    except Exception as e:
        print(f"{Fore.RED}❌ Error in message handler: {e}{Style.RESET_ALL}")
        if debug:
            import traceback
            traceback.print_exc()


async def run_gmgn_monitor(config, stop_event: asyncio.Event):
    """
    Main monitor loop with reconnection logic
    Similar to runCodexMonitor in Go

    Args:
        config: Configuration object
        stop_event: Event to signal shutdown
    """
    print("Starting GMGN WebSocket monitor...")
    print(f"   Monitoring {len(config.chains)} chains with real-time WebSocket")
    print(f"   Measuring TRUE indexation lag (WebSocket push timing)")
    print()

    stats_obj = stats.Statistics()
    reconnect_delay = 5
    max_reconnect_delay = 60

    while not stop_event.is_set():
        try:
            # Build connection URL with required parameters
            device_id = str(uuid.uuid4())
            client_id = f"gmgn_python_{uuid.uuid4().hex[:8]}"

            params = {
                "device_id": device_id,
                "client_id": client_id,
                "from_app": "gmgn",
                "app_ver": "20250729-1647-ffac485",
                "tz_name": "UTC",
                "tz_offset": "0",
                "app_lang": "en-US",
                "fp_did": str(uuid.uuid4()),
                "os": "python",
                "uuid": str(uuid.uuid4())
            }

            url = f"{GMGN_WS_URL}?{'&'.join(f'{k}={v}' for k, v in params.items())}"

            # Required headers for Cloudflare protection
            headers = {
                "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
                "Origin": "https://gmgn.ai",
                "Cache-Control": "no-cache",
                "Pragma": "no-cache"
            }

            print(f"{Fore.YELLOW}🔌 Connecting to GMGN WebSocket...{Style.RESET_ALL}")

            # Connect with proper settings
            async with websockets.connect(
                url,
                extra_headers=headers,
                ping_interval=20,
                ping_timeout=10,
                max_size=2**20,  # 1MB max message size
                compression=None  # IMPORTANT: no compression
            ) as websocket:
                print(f"{Fore.GREEN}✓ Connected to GMGN WebSocket{Style.RESET_ALL}")

                # Subscribe to channels for each chain
                for chain in config.chains:
                    # Subscribe to swap trades
                    swap_sub = create_subscription("new_pair_update", chain)
                    await websocket.send(json.dumps(swap_sub))
                    print(f"{Fore.CYAN}📡 Subscribed to swaps on {utils.get_chain_name(chain)}{Style.RESET_ALL}")

                    # Subscribe to new pools
                    pool_sub = create_subscription("new_pool_info", chain)
                    await websocket.send(json.dumps(pool_sub))
                    print(f"{Fore.CYAN}📡 Subscribed to new pools on {utils.get_chain_name(chain)}{Style.RESET_ALL}")

                print(f"\n{Fore.GREEN}✓ All subscriptions active. Listening for events...{Style.RESET_ALL}\n")

                # Reset reconnect delay on successful connection
                reconnect_delay = 5

                # Listen for messages
                await listen_to_gmgn(websocket, stats_obj, config.debug)

        except websockets.exceptions.InvalidStatusCode as e:
            print(f"{Fore.RED}❌ Connection failed with status {e.status_code}{Style.RESET_ALL}")
            if e.status_code == 403:
                print(f"{Fore.YELLOW}⚠ Cloudflare protection detected. Check headers.{Style.RESET_ALL}")
        except Exception as e:
            print(f"{Fore.RED}❌ Connection error: {e}{Style.RESET_ALL}")
            if config.debug:
                import traceback
                traceback.print_exc()

        if stop_event.is_set():
            break

        # Reconnect with exponential backoff
        print(f"{Fore.YELLOW}⏳ Reconnecting in {reconnect_delay}s...{Style.RESET_ALL}")
        await asyncio.sleep(reconnect_delay)
        reconnect_delay = min(reconnect_delay * 2, max_reconnect_delay)

    # Print final stats on shutdown
    stats_obj.print_summary()
    print(f"{Fore.GREEN}✓ GMGN monitor stopped{Style.RESET_ALL}")
