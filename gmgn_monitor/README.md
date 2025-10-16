# GMGN WebSocket Monitor

Python-based latency tracker for GMGN aggregator, monitoring swap trades and new pool discoveries across multiple chains.

## Features

- **Swap Trade Tracking**: Real-time monitoring of token swaps with latency measurement
- **New Pool Discovery**: Tracks when new liquidity pools are created and indexed
- **Multi-Chain Support**: Solana, BNB Chain, Base, and Ethereum
- **Prometheus Metrics**: Export metrics for Grafana dashboards
- **Debug Logging**: Detailed logging similar to Go monitors for troubleshooting
- **Automatic Reconnection**: Handles disconnections with exponential backoff
- **Statistics**: Live statistics on latency (min/max/avg) per chain

## Quick Start

```bash
# From project root
make gmgn
```

This will:
1. Set up Python virtual environment
2. Install dependencies
3. Create .env file if needed
4. Start the GMGN monitor

## Manual Setup

```bash
cd gmgn_monitor

# Create virtual environment
python3 -m venv venv

# Activate it
source venv/bin/activate  # macOS/Linux
# or
venv\Scripts\activate     # Windows

# Install dependencies
pip install -r requirements.txt

# Configure
cp .env.example .env
# Edit .env as needed

# Run
python3 gmgn_monitor.py
```

## Configuration

Edit `gmgn_monitor/.env`:

```bash
# Prometheus metrics port
METRICS_PORT=2113

# Enable debug output
DEBUG=true

# Chains to monitor (comma-separated)
# Options: sol, bsc, base, eth
CHAINS=sol,bsc,base

# Optional: Monitor specific pools
# CUSTOM_POOLS=sol:7qbRF6YsyGuLUVs6Y1q64bdVrfe4ZcUUz1JRdoVNUJnm
```

## Metrics

Access metrics at: `http://localhost:2113/metrics`

Available metrics:
- `gmgn_latency_milliseconds{chain="..."}` - Swap trade latency
- `pool_discovery_latency_milliseconds{aggregator="gmgn", chain="..."}` - Pool discovery latency
- `all_aggregator_latency_milliseconds{aggregator="gmgn", chain="..."}` - Combined metric

## Output Format

### Swap Trades
```
[GMGN-SWAP][2025-01-16 14:23:45][solana] New swap!
Token: BONK (So11111q...) | Price: $0.00001234 | MC: $500,000 |
Vol 1h: $10,000 | B/S: 50/30 | Trade time: 14:23:45.123 | Lag: 150ms [Good]
```

### New Pools
```
[GMGN-POOL][2025-01-16 14:25:30][base] New pool detected!
Token: NEWTOKEN (Base Token) | Exchange: uniswap-v2 |
Pool: 0x4c3638... | Initial Liq: $5000 | Price: $0.00010000 |
MC: $100,000 | Holders: 5 | Pool time: 14:25:30.456 | Lag: 200ms [Good]
```

### Debug Output
When `DEBUG=true`:
```
[DEBUG] Raw timestamp: 1705412625 | Trade time: 14:23:45.000 |
Receive time: 2025-01-16 14:23:45 | Lag: 150ms
```

## Architecture

```
gmgn_monitor.py
├── WebSocket Connection
│   ├── GMGN WS URL with auth params
│   ├── Cloudflare bypass headers
│   └── Ping/pong keep-alive
├── Channel Subscriptions
│   ├── new_pair_update (swaps)
│   └── new_pool_info (pools)
├── Message Handlers
│   ├── Timestamp extraction
│   ├── Latency calculation
│   └── Prometheus metric updates
└── Statistics & Logging
    ├── Per-chain stats
    ├── Debug output
    └── Periodic summaries
```

## Supported Chains

| Code | Blockchain | Chain ID |
|------|------------|----------|
| `sol` | Solana | - |
| `bsc` | BNB Chain | 56 |
| `base` | Base | 8453 |
| `eth` | Ethereum | 1 |

## Troubleshooting

### Connection Issues

**403 Forbidden Error**
- Headers are missing or incorrect
- The script includes proper Cloudflare bypass headers

**No Messages Received**
- Check chain codes are correct (`sol`, not `solana`)
- Verify subscription format in logs
- Enable DEBUG mode to see raw messages

**WebSocket Version Error**
```bash
# Must use websockets 12.0 (not 15.x)
pip install websockets==12.0
```

### Debug Mode

Enable in `.env`:
```bash
DEBUG=true
```

This shows:
- Raw timestamps
- Parsed vs receive times
- Failed JSON parsing
- Missing fields in messages
- Full error tracebacks

## Integration with Grafana

The GMGN monitor exports metrics on port 2113. To integrate with your existing Prometheus:

1. Update `monitoring/grafana/provisioning/datasources/prometheus.yml`:
```yaml
scrape_configs:
  - job_name: 'gmgn'
    static_configs:
      - targets: ['host.docker.internal:2113']
```

2. Restart Prometheus:
```bash
docker-compose restart prometheus
```

3. Metrics will appear in Grafana alongside CoinGecko, Mobula, and Codex data

## Comparison with Go Monitors

The Python GMGN monitor follows the same patterns as the Go monitors:

| Feature | Go Monitors | GMGN (Python) |
|---------|-------------|---------------|
| Debug output | `[DEBUG] Raw timestamp...` | ✓ Same format |
| Latency calculation | `(receiveTime - tradeTime) * 1000` | ✓ Same |
| Chain naming | `solana`, `bnb`, `base` | ✓ Normalized |
| Metrics port | 2112 | 2113 (separate) |
| Prometheus labels | `aggregator`, `chain` | ✓ Same |
| Output format | `[AGG][TIME][CHAIN]` | ✓ Same |

## Development

### Adding New Chains

Edit `gmgn_monitor.py`:

```python
CHAIN_MAP = {
    'sol': 'solana',
    'new_chain': 'custom_name',  # Add here
}
```

### Adding New Channels

GMGN supports additional channels:
- `new_launched_info` - Newly launched tokens
- Custom market data channels

Example:
```python
launched_sub = create_subscription("new_launched_info", "sol")
await websocket.send(json.dumps(launched_sub))
```

## Statistics Output

Every 50 messages, see live stats:
```
======================================================================
📊 GMGN Monitor Statistics
======================================================================

Swap Trades:
  Total: 125
  Avg Latency: 250ms [Good]
  Min: 50ms
  Max: 1500ms

New Pools:
  Total: 15
  Avg Latency: 500ms [Medium]
  Min: 200ms
  Max: 2000ms

Per-Chain:
  solana: 80 events, Avg: 200ms [Good]
  bnb: 40 events, Avg: 350ms [Medium]
  base: 20 events, Avg: 180ms [Good]
======================================================================
```

## Requirements

- Python 3.8+
- websockets 12.0 (not 15.x - has breaking changes)
- prometheus-client
- python-dotenv
- colorama

## License

Part of the Aggregator Latency Tracker project.
