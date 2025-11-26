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

// Structure pour onUnconfirmedEventsCreated (différente de onEventsCreated!)
type UnconfirmedEvent struct {
	Address          string `json:"address"`
	BlockHash        string `json:"blockHash"`
	BlockNumber      int64  `json:"blockNumber"`
	EventType        string `json:"eventType"`
	ID               string `json:"id"`
	LogIndex         int    `json:"logIndex"`
	Maker            string `json:"maker"`
	NetworkID        int    `json:"networkId"`
	Timestamp        int64  `json:"timestamp"`
	TransactionHash  string `json:"transactionHash"`
	TransactionIndex int    `json:"transactionIndex"`
}

type UnconfirmedEventData struct {
	Data struct {
		OnUnconfirmedEventsCreated struct {
			Address   string             `json:"address"`
			NetworkID int                `json:"networkId"`
			ID        string             `json:"id"`
			Events    []UnconfirmedEvent `json:"events"`
		} `json:"onUnconfirmedEventsCreated"`
	} `json:"data"`
}

func loadAPIKey() (string, error) {
	apiKey := os.Getenv("CODEX_API_KEY")
	if apiKey != "" {
		return apiKey, nil
	}

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

	return conn, nil
}

func subscribeToSolanaPoolUnconfirmed(conn *websocket.Conn, poolAddress string) error {
	// Subscription avec les champs CORRECTS selon la doc
	query := `subscription OnUnconfirmedPoolEvents($id: String!) {
		onUnconfirmedEventsCreated(id: $id) {
			address
			networkId
			id
			events {
				address
				blockHash
				blockNumber
				eventType
				id
				logIndex
				maker
				networkId
				timestamp
				transactionHash
				transactionIndex
			}
		}
	}`

	pairID := fmt.Sprintf("%s:%d", poolAddress, 1399811149) // Solana network ID

	subscribeMsg := CodexSubscribe{
		Type: "subscribe",
		ID:   "unconfirmed_test",
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
	fmt.Println("🧪 Test Codex onUnconfirmedEventsCreated (avec bon format)")
	fmt.Println("============================================================")
	fmt.Println()

	apiKey, err := loadAPIKey()
	if err != nil {
		log.Fatalf("❌ Error loading API key: %v", err)
	}

	conn, err := connectCodexWebSocket(apiKey)
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Println("✓ Connected to Codex WebSocket")

	solanaPool := "7qbRF6YsyGuLUVs6Y1q64bdVrfe4ZcUUz1JRdoVNUJnm"
	if err := subscribeToSolanaPoolUnconfirmed(conn, solanaPool); err != nil {
		log.Fatalf("❌ Failed to subscribe: %v", err)
	}

	pairID := fmt.Sprintf("%s:%d", solanaPool, 1399811149)
	fmt.Printf("✓ Subscribed to pair ID: %s\n", pairID)
	fmt.Println("   Using onUnconfirmedEventsCreated")
	fmt.Println()
	fmt.Println("📊 Waiting for UNCONFIRMED trades...")
	fmt.Println()

	eventsReceived := 0

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
			var eventData UnconfirmedEventData
			if err := json.Unmarshal(payloadBytes, &eventData); err != nil {
				fmt.Printf("⚠️  Parse error: %v\n", err)
				fmt.Printf("Raw payload: %s\n", string(payloadBytes))
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

				eventsReceived++

				tradeTime := time.Unix(event.Timestamp, 0)
				lag := receiveTime.Sub(tradeTime).Milliseconds()

				txHashShort := event.TransactionHash
				if len(txHashShort) > 8 {
					txHashShort = txHashShort[:8]
				}

				fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
				fmt.Printf("✅ UNCONFIRMED Event #%d\n", eventsReceived)
				fmt.Printf("   Tx:        %s...\n", txHashShort)
				fmt.Printf("   Block:     %d\n", event.BlockNumber)
				fmt.Printf("   Trade:     %s\n", tradeTime.Format("15:04:05.000"))
				fmt.Printf("   Received:  %s\n", receiveTime.Format("15:04:05.000"))
				fmt.Printf("   🚀 LATENCY: %d ms\n", lag)
				fmt.Println()
			}

		case "error":
			fmt.Printf("❌ Error: %+v\n", genericMsg.Payload)
			return

		case "complete":
			fmt.Printf("✓ Subscription completed\n")
			return

		case "ka":
			// Keep-alive
			continue
		}
	}
}
