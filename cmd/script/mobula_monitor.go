package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	mobulaWSURL = "wss://api.mobula.io"
)

var mobulaChains = []struct {
	blockchain  string
	blockchainID int64  // For matching responses
	chainName   string
	poolAddress string
}{
	{"solana", 1399811149, "solana", "7qbRF6YsyGuLUVs6Y1q64bdVrfe4ZcUUz1JRdoVNUJnm"},
	{"evm:56", 56, "bnb", "0x58f876857a02d6762e0101bb5c46a8c1ed44dc16"},
	{"evm:8453", 8453, "base", "0x4c36388be6f416a29c8d8eee81c771ce6be14b18"},
}

type MobulaSubscribeMessage struct {
	Type          string        `json:"type"`
	Authorization string        `json:"authorization"`
	Payload       MobulaPayload `json:"payload"`
}

type MobulaPayload struct {
	AssetMode bool           `json:"assetMode"`
	Items     []MobulaItem   `json:"items"`
}

type MobulaItem struct {
	Blockchain string `json:"blockchain"`
	Address    string `json:"address"`
}

type MobulaTradeData struct {
	Date              int64   `json:"date"`
	TokenPrice        float64 `json:"tokenPrice"`
	TokenPriceVs      float64 `json:"tokenPriceVs"`
	TokenAmount       float64 `json:"tokenAmount"`
	TokenAmountVs     float64 `json:"tokenAmountVs"`
	TokenAmountUsd    float64 `json:"tokenAmountUsd"`
	Type              string  `json:"type"`
	Operation         string  `json:"operation"`
	Blockchain        string  `json:"blockchain"`
	Hash              string  `json:"hash"`
	Sender            string  `json:"sender"`
	Timestamp         int64   `json:"timestamp"`
	Pair              string  `json:"pair"`
}

func connectMobulaWebSocket(apiKey string) (*websocket.Conn, error) {
	conn, _, err := websocket.DefaultDialer.Dial(mobulaWSURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	return conn, nil
}

func subscribeToMobulaChannel(conn *websocket.Conn, apiKey string) error {
	var items []MobulaItem
	for _, chain := range mobulaChains {
		items = append(items, MobulaItem{
			Blockchain: chain.blockchain,
			Address:    chain.poolAddress,
		})
	}

	subscribeMsg := MobulaSubscribeMessage{
		Type:          "fast-trade",
		Authorization: apiKey,
		Payload: MobulaPayload{
			AssetMode: false, // false = pools, true = tokens
			Items:     items,
		},
	}

	// Debug: print the subscription message
	msgJSON, _ := json.MarshalIndent(subscribeMsg, "", "  ")
	fmt.Printf("[MOBULA DEBUG] Sending subscription:\n%s\n", string(msgJSON))

	if err := conn.WriteJSON(subscribeMsg); err != nil {
		return fmt.Errorf("failed to subscribe to fast trades: %w", err)
	}

	return nil
}

func calculateMobulaLag(tradeTimestamp int64, receiveTime time.Time) int64 {
	tradeTime := time.UnixMilli(tradeTimestamp)
	lag := receiveTime.Sub(tradeTime)
	return lag.Milliseconds()
}

func getChainNameForMobula(blockchainName string) string {
	// Normalize blockchain name (lowercase)
	switch blockchainName {
	case "Solana", "solana":
		return "solana"
	case "Base", "base":
		return "base"
	case "BSC", "BNB Smart Chain", "bnb":
		return "bnb"
	default:
		return blockchainName
	}
}

func handleMobulaWebSocketMessages(conn *websocket.Conn, config *Config) {
	messageCount := 0
	for {
		_, messageBytes, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[MOBULA] WebSocket read error: %v", err)
			return
		}

		receiveTime := time.Now()
		messageCount++

		// Debug: Print first few raw messages
		if messageCount <= 5 {
			fmt.Printf("[MOBULA DEBUG] Message #%d: %s\n", messageCount, string(messageBytes))
		}

		var trade MobulaTradeData
		if err := json.Unmarshal(messageBytes, &trade); err != nil {
			// Log for debugging
			if messageCount <= 5 {
				fmt.Printf("[MOBULA DEBUG] Failed to parse as trade: %v\n", err)
			}
			continue
		}

		if trade.Hash == "" || trade.Blockchain == "" {
			if messageCount <= 5 {
				fmt.Printf("[MOBULA DEBUG] Skipping message - Hash: %s, Blockchain: %s\n", trade.Hash, trade.Blockchain)
			}
			continue
		}

		lagMs := calculateMobulaLag(trade.Date, receiveTime)

		chainName := getChainNameForMobula(trade.Blockchain)
		timestamp := receiveTime.Format("2006-01-02 15:04:05")

		tradeTime := time.UnixMilli(trade.Date).Format("15:04:05.000")

		fmt.Printf("\n[DEBUG] Raw timestamp: %d | Trade time parsed: %s | Receive time: %s | Lag: %dms\n",
			trade.Date, tradeTime, timestamp, lagMs)

		txHashShort := trade.Hash
		if len(txHashShort) > 8 {
			txHashShort = txHashShort[:8]
		}

		fmt.Printf("[MOBULA][%s][%s] New fast trade! Tx: %s... | Type: %s | Volume: $%.2f | Trade time: %s | Lag: %dms\n",
			timestamp,
			chainName,
			txHashShort,
			trade.Type,
			trade.TokenAmountUsd,
			tradeTime,
			lagMs,
		)

		RecordLatency("mobula", chainName, float64(lagMs))
		RecordTrade("mobula", chainName, trade.Type, trade.TokenAmountUsd)
	}
}

func runMobulaMonitor(config *Config, stopChan <-chan struct{}) {
	fmt.Println("🚀 Starting Mobula WebSocket monitor...")
	fmt.Printf("   Monitoring %d chains with real-time WebSocket\n", len(mobulaChains))
	fmt.Printf("   Measuring TRUE indexation lag (WebSocket push timing)\n")
	fmt.Println()

	if config.MobulaAPIKey == "" {
		fmt.Println("⚠ MOBULA_API_KEY not set in .env file. Skipping Mobula monitor.")
		return
	}

	conn, err := connectMobulaWebSocket(config.MobulaAPIKey)
	if err != nil {
		log.Printf("[MOBULA] Failed to connect: %v", err)
		return
	}
	defer conn.Close()

	fmt.Println("   ✓ Connected to Mobula WebSocket")

	if err := subscribeToMobulaChannel(conn, config.MobulaAPIKey); err != nil {
		log.Printf("[MOBULA] Failed to subscribe to channel: %v", err)
		return
	}
	fmt.Println("   ✓ Subscribed to fast-trade stream")

	time.Sleep(500 * time.Millisecond)

	fmt.Println("   ✓ Configured pools for monitoring:")
	for _, chain := range mobulaChains {
		fmt.Printf("     - %s (%s)\n", chain.chainName, chain.poolAddress)
	}
	fmt.Println()

	go handleMobulaWebSocketMessages(conn, config)

	<-stopChan
	fmt.Println("🛑 Mobula monitor stopped")
}
