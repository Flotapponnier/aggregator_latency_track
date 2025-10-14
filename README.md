# 📊 Aggregator Latency Monitor

Real-time monitoring dashboard for tracking blockchain data indexation latency across multiple aggregators and chains. Visualize how fast aggregators index on-chain trades with live Grafana dashboards.

![Grafana Dashboard](https://img.shields.io/badge/Grafana-Dashboard-orange) ![Prometheus](https://img.shields.io/badge/Prometheus-Metrics-blue) ![Go](https://img.shields.io/badge/Go-1.24-00ADD8)

---

## 🔑 Environment Variables

Create a `.env` file in the root directory with the following variables:

```bash
# CoinGecko API Key (required for CoinGecko monitor)
COINGECKO_API_KEY=your_coingecko_api_key_here

# Mobula API Key (required for Mobula monitor)
MOBULA_API_KEY=your_mobula_api_key_here

# Add more API keys here as you integrate additional aggregators:
# DEXSCREENER_API_KEY=your_dexscreener_key
# MORALIS_API_KEY=your_moralis_key
```

**Note**: Each aggregator monitor requires its corresponding API key. If a key is not provided, that specific monitor will be skipped.

---

## 🎯 What Does This Do?

This tool measures **data freshness** (indexation lag) for blockchain data providers:

- **Indexation Lag**: Time between when a trade happens on-chain and when it appears in the aggregator's API
- **Real-time Tracking**: WebSocket connections receive trades instantly when indexed
- **Live Dashboard**: Grafana displays metrics in real-time
- **Multi-chain Support**: Track latency across Solana, BNB Chain, and Base simultaneously

### Example:
```
🔹 Trade occurs on Solana:        16:08:25.000
🔹 CoinGecko indexes and pushes:  16:08:28.050
🔹 Calculated lag:                3050ms
```

---

## 🚀 Quick Start

### Prerequisites

- **Go 1.24+**
- **Docker & Docker Compose**
- **Make**
- **API Key** from the aggregator you want to track

### 1. Clone & Configure

```bash
# Clone the repository
git clone <repo-url>
cd aggregator_latency_monitor

# Create .env file
cat > .env << EOF
COINGECKO_API_KEY=your_coingecko_api_key_here
MOBULA_API_KEY=your_mobula_api_key_here
EOF
```

### 2. Run Everything

```bash
make run
```

That's it! This single command will:
1. ✅ Build the Go binary
2. ✅ Start Prometheus + Grafana
3. ✅ Launch the latency monitor
4. ✅ Display real-time metrics in terminal

### 3. Access the Dashboard

Open your browser:

| Service | URL | Credentials |
|---------|-----|-------------|
| **Grafana Dashboard** | http://localhost:3000 | admin / admin |
| **Prometheus** | http://localhost:9090 | - |
| **Metrics Endpoint** | http://localhost:2112/metrics | - |

The **"Aggregator Latency Monitor"** dashboard loads automatically in Grafana!

---

## 📊 Understanding the Dashboard

### Main Panels

The Grafana dashboard contains 3 main panels to compare aggregator performance:

1. **CoinGecko - Indexation Lag by Chain**
   - Shows CoinGecko latency for Solana, BNB, and Base
   - Typical lag: ~4000-8000ms
   - Measurement: Time from on-chain trade to WebSocket reception

2. **Mobula - Indexation Lag by Chain**
   - Shows Mobula latency for Solana, BNB, and Base
   - Typical lag: ~1000-2000ms
   - Measurement: Time from on-chain trade to WebSocket reception

3. **All Aggregators Comparison - Latency by Chain**
   - Overlays both aggregators on the same graph
   - CoinGecko in blue, Mobula in orange
   - Easy side-by-side comparison
   - Legend shows mean, last value, max, and min for each series

### Dashboard Controls

- **🔄 Auto-refresh**: Every 5 seconds
- **🕐 Time Range**: Adjustable (default: last 30 minutes)
- **🔍 Zoom**: Click and drag on any graph
- **📌 Legend**: Click series name to show/hide specific chains or aggregators

---

## 🛠️ How to Add a New Aggregator

### Step 1: Create Monitor File

Create `cmd/script/youraggregator_monitor.go`:

```go
package main

import (
    "fmt"
    "time"
    "github.com/gorilla/websocket"
)

// Chain configurations
var yourAggregatorChains = []struct {
    networkID   string
    chainName   string
    poolAddress string
}{
    {"solana", "solana", "your_pool_address"},
    {"bsc", "bnb", "your_pool_address"},
}

// Connect to aggregator's WebSocket
func connectYourAggregatorWebSocket(apiKey string) (*websocket.Conn, error) {
    conn, _, err := websocket.DefaultDialer.Dial("wss://aggregator-ws-url", nil)
    if err != nil {
        return nil, err
    }
    // Authenticate and subscribe...
    return conn, nil
}

// Handle incoming trades
func handleYourAggregatorMessages(conn *websocket.Conn, config *Config) {
    for {
        // Read message
        var trade YourTradeStruct
        conn.ReadJSON(&trade)

        receiveTime := time.Now()
        lagMs := calculateLag(trade.Timestamp, receiveTime)

        // Record metrics for Prometheus/Grafana
        RecordLatency("youraggregator", trade.Chain, float64(lagMs))
        RecordTrade("youraggregator", trade.Chain, trade.Type, trade.Volume)

        fmt.Printf("[YOURAGGREGATOR][%s] Lag: %dms\n", trade.Chain, lagMs)
    }
}

// Main monitor function
func runYourAggregatorMonitor(config *Config, stopChan <-chan struct{}) {
    fmt.Println(" Starting YourAggregator monitor...")

    conn, err := connectYourAggregatorWebSocket(config.YourAggregatorAPIKey)
    if err != nil {
        fmt.Printf("Failed to connect: %v\n", err)
        return
    }
    defer conn.Close()

    go handleYourAggregatorMessages(conn, config)

    <-stopChan
    fmt.Println(" YourAggregator monitor stopped")
}
```

### Step 2: Update Configuration

Edit `cmd/script/config.go`:

```go
type Config struct {
    CoinGeckoAPIKey      string
    YourAggregatorAPIKey string  // Add this
}

func loadEnv() (*Config, error) {
    // ... existing code ...

    switch key {
    case "COINGECKO_API_KEY":
        config.CoinGeckoAPIKey = value
    case "YOURAGGREGATOR_API_KEY":  // Add this
        config.YourAggregatorAPIKey = value
    }

    // ... rest of code ...
}
```

### Step 3: Enable in Main

Edit `cmd/script/main.go`:

```go
func main() {
    // ... existing code ...

    // Add your aggregator monitor
    wg.Add(1)
    go func() {
        defer wg.Done()
        runYourAggregatorMonitor(config, stopChan)
    }()

    // ... rest of code ...
}
```

### Step 4: Add API Key to .env

```bash
echo "YOURAGGREGATOR_API_KEY=your_key_here" >> .env
```

### Step 5: Update Grafana Dashboard

1. Open http://localhost:3000
2. Go to "Aggregator Latency Monitor" dashboard
3. Click any panel title → **Edit**
4. Update the PromQL query to include your aggregator:

```promql
# Old query (single aggregator)
coingecko_latency_milliseconds

# New query (shows both aggregators)
{__name__=~"coingecko_latency_milliseconds|youraggregator_latency_milliseconds"}
```

5. **Save** the dashboard
6. Export the updated dashboard:
   - Dashboard settings (⚙️) → **JSON Model**
   - Copy and save to `monitoring/grafana/dashboards/aggregator_latency.json`

### Step 6: Rebuild and Run

```bash
make clean
make run
```

You'll now see your aggregator's metrics in both the terminal and Grafana! 🎉

---

## 📋 Available Commands

| Command | Description |
|---------|-------------|
| `make run` | Build + start Grafana + launch monitor (all-in-one) |
| `make build` | Build the Go binary only |
| `make down` | Stop Grafana/Prometheus |
| `make stop` | Alias for `make down` |
| `make clean` | Stop services + remove binary |
| `make destroy` | Remove everything including volumes (asks confirmation) |
| `make help` | Show all commands |

---

## 🏗️ Project Architecture

```
┌──────────────────┐
│  Aggregator APIs │
│   (WebSocket)    │
└────────┬─────────┘
         │ Real-time trades
         ▼
┌──────────────────┐
│ Latency Monitor  │◄─── .env (API keys)
│   (Go App)       │
│   Port: 2112     │
└────────┬─────────┘
         │ /metrics endpoint
         ▼
┌──────────────────┐
│   Prometheus     │◄─── monitoring/prometheus.yml
│   Port: 9090     │
└────────┬─────────┘
         │ Scrapes every 5s
         ▼
┌──────────────────┐
│    Grafana       │◄─── monitoring/grafana/
│   Port: 3000     │
└──────────────────┘
```

### File Structure

```
aggregator_latency_monitor/
├── cmd/script/
│   ├── main.go                    # Entry point
│   ├── config.go                  # .env loader
│   ├── metrics.go                 # Prometheus metrics
│   ├── geckoterminal_monitor.go   # CoinGecko monitor
│   └── mobula_monitor.go          # Mobula monitor
│
├── monitoring/
│   ├── prometheus.yml             # Prometheus config
│   └── grafana/
│       ├── provisioning/
│       │   ├── datasources/       # Auto-add Prometheus
│       │   └── dashboards/        # Auto-load dashboards
│       └── dashboards/
│           └── aggregator_latency.json  # Main dashboard
│
├── docker-compose.yml             # Grafana + Prometheus stack
├── Makefile                       # Build commands
├── .env                           # API keys (not in git)
└── README.md
```

---

## 🔧 Advanced Configuration

### Modify Scrape Interval

Edit `monitoring/prometheus.yml`:

```yaml
global:
  scrape_interval: 5s  # Change to 10s, 30s, etc.
```

### Add More Chains

Edit your monitor file (e.g., `geckoterminal_monitor.go`):

```go
var coinGeckoChains = []struct {
    networkID   string
    chainName   string
    poolAddress string
}{
    {"solana", "solana", "pool_address"},
    {"bsc", "bnb", "pool_address"},
    {"base", "base", "pool_address"},
    {"polygon", "polygon", "new_pool_address"},  // Add this
}
```

Rebuild and restart:
```bash
make clean && make run
```

### Custom Grafana Dashboard

1. Edit the dashboard in Grafana UI
2. Export JSON: Dashboard settings → JSON Model
3. Save to `monitoring/grafana/dashboards/aggregator_latency.json`
4. Restart: `make stop && make run`

---

## 🐛 Troubleshooting

### Issue: No data in Grafana

**Check:**
```bash
# 1. Is the Go app running?
ps aux | grep latency_monitor

# 2. Are metrics being exposed?
curl http://localhost:2112/metrics | grep aggregator_latency

# 3. Is Prometheus scraping?
# Go to http://localhost:9090/targets - should show "UP"

# 4. Is Grafana connected?
# Go to http://localhost:3000 → Settings → Data Sources → Prometheus
```

### Issue: WebSocket connection failed

- Verify API key in `.env`
- Check if API key has WebSocket access (may need paid tier)
- Look for errors in terminal output

### Issue: Grafana shows "No data"

```bash
# Restart everything
make clean
make run

# Check Prometheus has data
# Go to http://localhost:9090
# Query: aggregator_latency_milliseconds
# Should see results
```

### Issue: Docker errors

```bash
# Full reset
docker-compose down -v
make clean
make run
```

---

## 📊 Exported Metrics

The following Prometheus metrics are available:

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `{aggregator}_latency_milliseconds` | Gauge | chain | Current indexation lag in ms |
| `{aggregator}_trades_total` | Counter | chain, type | Total trades processed |
| `{aggregator}_trade_volume_usd` | Gauge | chain | Last trade volume in USD |

*Note: `{aggregator}` is replaced with the actual aggregator name (e.g., `coingecko_latency_milliseconds`)*

### Example PromQL Queries

```promql
# Average lag for CoinGecko across all chains
avg(coingecko_latency_milliseconds)

# Max lag in last 5 minutes for a specific chain
max_over_time(coingecko_latency_milliseconds{chain="solana"}[5m])

# Trade rate (per second) for CoinGecko
rate(coingecko_trades_total[1m])

# Compare latency between aggregators
coingecko_latency_milliseconds{chain="solana"}
- youraggregator_latency_milliseconds{chain="solana"}
```

---

## 🎨 Currently Tracked

| Aggregator | Chains | Method | Status |
|------------|--------|--------|--------|
| **CoinGecko** | Solana, BNB, Base | WebSocket | ✅ Active |
| **Mobula** | Solana, BNB, Base | WebSocket | ✅ Active |

---

## 💡 Pro Tips

1. **Monitor Multiple Aggregators**: Add several aggregators to compare their performance side-by-side
2. **Set Alerts**: Configure Grafana alerts when lag exceeds thresholds
3. **Export Data**: Use Prometheus API to export historical data for analysis
4. **Optimize Pools**: Monitor high-volume pools for more frequent data points
5. **API Rate Limits**: Use WebSocket (persistent connection) instead of REST polling to avoid rate limits
