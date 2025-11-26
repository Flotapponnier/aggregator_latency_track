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
		OnEventsCreated struct {
			Address   string       `json:"address"`
			NetworkID int          `json:"networkId"`
			Events    []CodexEvent `json:"events"`
		} `json:"onEventsCreated"`
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

	fmt.Println("✓ Connection acknowledged by Codex")

	return conn, nil
}

func subscribeToSolanaPool(conn *websocket.Conn, poolAddress string) error {
	// Test avec paramètre confirmed: false
	query := `subscription OnPoolEvents($address: String!, $networkId: Int!, $confirmed: Boolean) {
		onEventsCreated(address: $address, networkId: $networkId, confirmed: $confirmed) {
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

	subscribeMsg := CodexSubscribe{
		Type: "subscribe",
		ID:   "solana_test",
		Payload: map[string]interface{}{
			"query": query,
			"variables": map[string]interface{}{
				"address":   poolAddress,
				"networkId": 1399811149,
				"confirmed": false, // Tester avec unconfirmed
			},
		},
	}

	if err := conn.WriteJSON(subscribeMsg); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return nil
}

func main() {
	fmt.Println("🧪 Test Codex Solana - Paramètre confirmed:false")
	fmt.Println("=================================================")
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
	if err := subscribeToSolanaPool(conn, solanaPool); err != nil {
		log.Fatalf("❌ Failed to subscribe: %v", err)
	}

	fmt.Printf("✓ Subscribed to Solana pool (%s)\n", solanaPool)
	fmt.Println("   Testing with parameter: confirmed=false")
	fmt.Println()
	fmt.Println("📊 Waiting for trades...")
	fmt.Println()

	timeout := time.After(30 * time.Second)
	eventsReceived := 0

	for {
		select {
		case <-timeout:
			fmt.Printf("\n⏱️  Timeout après 30s - Events reçus: %d\n", eventsReceived)
			if eventsReceived == 0 {
				fmt.Println("❌ Aucun event reçu - paramètre confirmed:false ne marche pas")
			}
			return

		default:
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			_, messageBytes, err := conn.ReadMessage()
			if err != nil {
				if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
					continue
				}
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

				eventsOutput := eventData.Data.OnEventsCreated
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

					fmt.Printf("✅ Event #%d - Latency: %dms - Tx: %s\n",
						eventsReceived, lag, event.TransactionHash[:16]+"...")
				}

			case "error":
				fmt.Printf("❌ Error: %+v\n", genericMsg.Payload)

			case "ka":
				continue
			}
		}
	}
}
