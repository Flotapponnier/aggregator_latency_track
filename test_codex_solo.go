package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	codexWSURL = "wss://graph.codex.io/graphql"
)

type CodexWSMessage struct {
	Type    string                 `json:"type"`
	ID      string                 `json:"id,omitempty"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

type CodexConnectionInit struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

type CodexSubscribe struct {
	Type    string                 `json:"type"`
	ID      string                 `json:"id"`
	Payload map[string]interface{} `json:"payload"`
}

type CodexEvent struct {
	NetworkID          int    `json:"networkId"`
	BlockNumber        int64  `json:"blockNumber"`
	Timestamp          int64  `json:"timestamp"`
	TransactionHash    string `json:"transactionHash"`
	EventType          string `json:"eventType"`
	Token0Address      string `json:"token0Address"`
	Token1Address      string `json:"token1Address"`
	Token0SwapValueUsd string `json:"token0SwapValueUsd"`
	Token1SwapValueUsd string `json:"token1SwapValueUsd"`
}

type CodexEventData struct {
	Data struct {
		OnUnconfirmedEventsCreated struct {
			Address   string       `json:"address"`
			NetworkID int          `json:"networkId"`
			Events    []CodexEvent `json:"events"`
		} `json:"onUnconfirmedEventsCreated"`
	} `json:"data"`
}

func loadAPIKey() (string, error) {
	// Try environment variable first
	apiKey := os.Getenv("CODEX_API_KEY")
	if apiKey != "" {
		return apiKey, nil
	}

	// Try .env file
	file, err := os.Open(".env")
	if err != nil {
		return "", fmt.Errorf("no CODEX_API_KEY in env and cannot open .env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if key == "CODEX_API_KEY" {
			return value, nil
		}
	}

	return "", fmt.Errorf("CODEX_API_KEY not found")
}

func connectCodexWebSocket(apiKey string) (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		Subprotocols: []string{"graphql-transport-ws"},
	}

	conn, _, err := dialer.Dial(codexWSURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	initMsg := CodexConnectionInit{
		Type: "connection_init",
		Payload: map[string]interface{}{
			"Authorization": apiKey,
		},
	}

	if err := conn.WriteJSON(initMsg); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send connection_init: %w", err)
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read connection_ack: %w", err)
	}

	var ackMsg CodexWSMessage
	if err := json.Unmarshal(msg, &ackMsg); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to parse connection_ack: %w", err)
	}

	if ackMsg.Type != "connection_ack" {
		conn.Close()
		return nil, fmt.Errorf("expected connection_ack, got: %s", ackMsg.Type)
	}

	fmt.Println("✓ Connection acknowledged by Codex")

	return conn, nil
}

func subscribeToSolanaPool(conn *websocket.Conn, poolAddress string) error {
	// Use UNCONFIRMED events for true latency measurement
	// Format: id = "address:networkId" (Solana-specific)
	query := `subscription OnUnconfirmedPoolEvents($id: String!) {
		onUnconfirmedEventsCreated(id: $id) {
			address
			networkId
			events {
				networkId
				blockNumber
				timestamp
				transactionHash
				eventType
				token0Address
				token1Address
				token0SwapValueUsd
				token1SwapValueUsd
			}
		}
	}`

	// Format: "address:networkId"
	pairID := fmt.Sprintf("%s:%d", poolAddress, 1399811149) // Solana network ID

	subscribeMsg := CodexSubscribe{
		Type: "subscribe",
		ID:   "solana_test",
		Payload: map[string]interface{}{
			"query": query,
			"variables": map[string]interface{}{
				"id": pairID,
			},
		},
	}

	if err := conn.WriteJSON(subscribeMsg); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return nil
}

func main() {
	fmt.Println("🧪 Test Codex Solana - Vérification de la latence")
	fmt.Println("================================================")
	fmt.Println()

	// Load API key
	apiKey, err := loadAPIKey()
	if err != nil {
		log.Fatalf("❌ Error loading API key: %v", err)
	}

	// Connect to WebSocket
	conn, err := connectCodexWebSocket(apiKey)
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Println("✓ Connected to Codex WebSocket")

	// Subscribe to Solana pool
	solanaPool := "7qbRF6YsyGuLUVs6Y1q64bdVrfe4ZcUUz1JRdoVNUJnm"
	if err := subscribeToSolanaPool(conn, solanaPool); err != nil {
		log.Fatalf("❌ Failed to subscribe: %v", err)
	}

	fmt.Printf("✓ Subscribed to Solana pool (%s)\n", solanaPool)
	fmt.Println("   Using UNCONFIRMED events for lowest latency measurement")
	fmt.Println()
	fmt.Println("📊 Waiting for trades... (Press Ctrl+C to stop)")
	fmt.Println()

	// Listen for messages
	for {
		_, messageBytes, err := conn.ReadMessage()
		if err != nil {
			log.Printf("❌ WebSocket read error: %v", err)
			return
		}

		receiveTime := time.Now().UTC()

		var genericMsg CodexWSMessage
		if err := json.Unmarshal(messageBytes, &genericMsg); err != nil {
			continue
		}

		switch genericMsg.Type {
		case "next":
			if genericMsg.Payload == nil {
				continue
			}

			payloadBytes, _ := json.Marshal(genericMsg.Payload)
			var eventData CodexEventData
			if err := json.Unmarshal(payloadBytes, &eventData); err != nil {
				continue
			}

			eventsOutput := eventData.Data.OnUnconfirmedEventsCreated
			if len(eventsOutput.Events) == 0 {
				continue
			}

			for _, event := range eventsOutput.Events {
				if event.EventType != "Swap" {
					continue
				}

				if event.TransactionHash == "" {
					continue
				}

				// Display raw timestamp
				fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
				fmt.Printf("📦 New Swap Event Received\n")
				fmt.Printf("   Transaction: %s\n", event.TransactionHash[:16]+"...")
				fmt.Printf("   Block: %d\n", event.BlockNumber)
				fmt.Println()

				// Current time
				fmt.Printf("⏰ Receive Time (Now):        %s (Unix: %d)\n",
					receiveTime.Format("2006-01-02 15:04:05.000"),
					receiveTime.Unix())
				fmt.Println()

				// Raw timestamp from API
				fmt.Printf("📡 Raw Timestamp from API:    %d\n", event.Timestamp)
				fmt.Println()

				// Test 1: Interpret as seconds
				tradeTimeSeconds := time.Unix(event.Timestamp, 0)
				lagSeconds := receiveTime.Sub(tradeTimeSeconds).Milliseconds()
				fmt.Printf("🧪 Test 1 - Interpret as SECONDS:\n")
				fmt.Printf("   Trade Time:  %s\n", tradeTimeSeconds.Format("2006-01-02 15:04:05.000"))
				fmt.Printf("   Latency:     %d ms\n", lagSeconds)
				fmt.Println()

				// Test 2: Interpret as milliseconds
				tradeTimeMillis := time.Unix(event.Timestamp/1000, (event.Timestamp%1000)*1000000)
				lagMillis := receiveTime.Sub(tradeTimeMillis).Milliseconds()
				fmt.Printf("🧪 Test 2 - Interpret as MILLISECONDS:\n")
				fmt.Printf("   Trade Time:  %s\n", tradeTimeMillis.Format("2006-01-02 15:04:05.000"))
				fmt.Printf("   Latency:     %d ms\n", lagMillis)
				fmt.Println()

				// Determine which is more reasonable
				if lagSeconds < 0 || lagSeconds > 300000 { // More than 5 minutes or negative
					fmt.Printf("⚠️  Test 1 looks WRONG (negative or >5 minutes)\n")
				}
				if lagMillis < 0 || lagMillis > 300000 {
					fmt.Printf("⚠️  Test 2 looks WRONG (negative or >5 minutes)\n")
				}

				if lagMillis >= 0 && lagMillis < 60000 {
					fmt.Printf("✅ Test 2 looks CORRECT (reasonable latency: %d ms)\n", lagMillis)
				}
				if lagSeconds >= 0 && lagSeconds < 60000 {
					fmt.Printf("✅ Test 1 looks CORRECT (reasonable latency: %d ms)\n", lagSeconds)
				}

				fmt.Println()
			}

		case "error":
			fmt.Printf("❌ Error: %+v\n", genericMsg.Payload)

		case "complete":
			fmt.Printf("✓ Subscription completed\n")

		case "ka":
			// Keep-alive, ignore
			continue

		default:
			continue
		}
	}
}
