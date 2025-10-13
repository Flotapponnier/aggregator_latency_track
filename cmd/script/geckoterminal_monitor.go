package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	coinGeckoWSURL = "wss://stream.coingecko.com/v1"
)

var coinGeckoChains = []struct {
	networkID   string
	chainName   string
	poolAddress string
}{
	{"solana", "solana", "7qbRF6YsyGuLUVs6Y1q64bdVrfe4ZcUUz1JRdoVNUJnm"},
	{"bsc", "bnb", "0x58f876857a02d6762e0101bb5c46a8c1ed44dc16"},
	{"base", "base", "0x4c36388be6f416a29c8d8eee81c771ce6be14b18"},
}

type WSCommand struct {
	Command    string `json:"command"`
	Identifier string `json:"identifier,omitempty"`
	Data       string `json:"data,omitempty"`
}

type CoinGeckoMessage struct {
	Type       string          `json:"type,omitempty"`
	Message    json.RawMessage `json:"message,omitempty"`
	Identifier string          `json:"identifier,omitempty"`
}

type TradeData struct {
	C  string  `json:"c"`
	N  string  `json:"n"`
	Pa string  `json:"pa"`
	Tx string  `json:"tx"`
	Ty string  `json:"ty"`
	To float64 `json:"to"`
	Vo float64 `json:"vo"`
	Pc float64 `json:"pc"`
	Pu float64 `json:"pu"`
	T  int64   `json:"t"`
}

func connectCoinGeckoWebSocket(apiKey string) (*websocket.Conn, error) {
	url := fmt.Sprintf("%s?x_cg_pro_api_key=%s", coinGeckoWSURL, apiKey)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	return conn, nil
}

func subscribeToCoinGeckoChannel(conn *websocket.Conn) error {
	subscribeCmd := WSCommand{
		Command:    "subscribe",
		Identifier: `{"channel":"OnchainTrade"}`,
	}

	if err := conn.WriteJSON(subscribeCmd); err != nil {
		return fmt.Errorf("failed to subscribe to channel: %w", err)
	}

	return nil
}

func setPoolsForCoinGecko(conn *websocket.Conn, pools []string) error {
	poolsJSON, err := json.Marshal(pools)
	if err != nil {
		return fmt.Errorf("failed to marshal pools: %w", err)
	}

	dataPayload := fmt.Sprintf(`{"network_id:pool_addresses":%s,"action":"set_pools"}`, string(poolsJSON))

	messageCmd := WSCommand{
		Command:    "message",
		Identifier: `{"channel":"OnchainTrade"}`,
		Data:       dataPayload,
	}

	if err := conn.WriteJSON(messageCmd); err != nil {
		return fmt.Errorf("failed to set pools: %w", err)
	}

	return nil
}

func calculateCoinGeckoLag(tradeTimestamp int64, receiveTime time.Time) int64 {
	tradeTime := time.UnixMilli(tradeTimestamp)
	lag := receiveTime.Sub(tradeTime)
	return lag.Milliseconds()
}

func getChainNameForCoinGecko(networkID string) string {
	for _, chain := range coinGeckoChains {
		if chain.networkID == networkID {
			return chain.chainName
		}
	}
	return networkID
}

func handleCoinGeckoWebSocketMessages(conn *websocket.Conn, config *Config) {
	for {
		_, messageBytes, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[COINGECKO] WebSocket read error: %v", err)
			return
		}

		receiveTime := time.Now().UTC()

		var trade TradeData
		if err := json.Unmarshal(messageBytes, &trade); err != nil {
			continue
		}

		if trade.Tx == "" || trade.N == "" {
			continue
		}

		lagMs := calculateCoinGeckoLag(trade.T, receiveTime)

		chainName := getChainNameForCoinGecko(trade.N)
		timestamp := receiveTime.Format("2006-01-02 15:04:05")

		tradeTime := time.UnixMilli(trade.T).Format("15:04:05.000")

		fmt.Printf("\n[DEBUG] Raw timestamp: %d | Trade time parsed: %s | Receive time: %s | Lag: %dms\n",
			trade.T, tradeTime, timestamp, lagMs)

		txHashShort := trade.Tx
		if len(txHashShort) > 8 {
			txHashShort = txHashShort[:8]
		}

		tradeType := "buy"
		if trade.Ty == "s" {
			tradeType = "sell"
		}

		fmt.Printf("[COINGECKO][%s][%s] New trade! Tx: %s... | Type: %s | Volume: $%.2f | Trade time: %s | Lag: %dms\n",
			timestamp,
			chainName,
			txHashShort,
			tradeType,
			trade.Vo,
			tradeTime,
			lagMs,
		)

		RecordLatency("coingecko", chainName, float64(lagMs))
	}
}

func runGeckoTerminalMonitor(config *Config, stopChan <-chan struct{}) {
	fmt.Println("🚀 Starting CoinGecko WebSocket monitor...")
	fmt.Printf("   Monitoring %d chains with real-time WebSocket\n", len(coinGeckoChains))
	fmt.Printf("   Measuring TRUE indexation lag (WebSocket push timing)\n")
	fmt.Println()

	if config.CoinGeckoAPIKey == "" {
		fmt.Println("⚠ COINGECKO_API_KEY not set in .env file. Skipping CoinGecko monitor.")
		return
	}

	conn, err := connectCoinGeckoWebSocket(config.CoinGeckoAPIKey)
	if err != nil {
		log.Printf("[COINGECKO] Failed to connect: %v", err)
		return
	}
	defer conn.Close()

	fmt.Println("   ✓ Connected to CoinGecko WebSocket")

	if err := subscribeToCoinGeckoChannel(conn); err != nil {
		log.Printf("[COINGECKO] Failed to subscribe to channel: %v", err)
		return
	}
	fmt.Println("   ✓ Subscribed to OnchainTrade channel")

	time.Sleep(500 * time.Millisecond)

	var pools []string
	for _, chain := range coinGeckoChains {
		poolAddress := fmt.Sprintf("%s:%s", chain.networkID, chain.poolAddress)
		pools = append(pools, poolAddress)
	}

	if err := setPoolsForCoinGecko(conn, pools); err != nil {
		log.Printf("[COINGECKO] Failed to set pools: %v", err)
		return
	}

	fmt.Println("   ✓ Configured pools for monitoring:")
	for _, chain := range coinGeckoChains {
		fmt.Printf("     - %s (%s)\n", chain.chainName, chain.poolAddress)
	}
	fmt.Println()

	go handleCoinGeckoWebSocketMessages(conn, config)

	<-stopChan
	fmt.Println("🛑 CoinGecko monitor stopped")
}
