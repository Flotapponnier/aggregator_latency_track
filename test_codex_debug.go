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

func main() {
	fmt.Println("🔍 Codex Debug - Affiche TOUS les messages")
	fmt.Println("==========================================")
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

	// Test avec le format correct selon la doc
	poolAddress := "7qbRF6YsyGuLUVs6Y1q64bdVrfe4ZcUUz1JRdoVNUJnm"
	networkID := 1399811149
	pairID := fmt.Sprintf("%s:%d", poolAddress, networkID)

	fmt.Printf("📝 Pair ID: %s\n", pairID)
	fmt.Println()

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
			}
		}
	}`

	subscribeMsg := CodexSubscribe{
		Type: "subscribe",
		ID:   "debug_test",
		Payload: map[string]interface{}{
			"query": query,
			"variables": map[string]interface{}{
				"id": pairID,
			},
		},
	}

	fmt.Println("📤 Sending subscription:")
	prettyJSON, _ := json.MarshalIndent(subscribeMsg, "", "  ")
	fmt.Println(string(prettyJSON))
	fmt.Println()

	if err := conn.WriteJSON(subscribeMsg); err != nil {
		log.Fatalf("❌ Failed to subscribe: %v", err)
	}

	fmt.Println("✓ Subscription sent")
	fmt.Println()
	fmt.Println("📥 Waiting for messages (will display ALL messages)...")
	fmt.Println()

	messageCount := 0
	timeout := time.After(30 * time.Second)

	// Read immediate response (might be an error)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, immediateMsg, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("❌ Failed to read immediate response: %v", err)
	}

	fmt.Println("📩 IMMEDIATE RESPONSE AFTER SUBSCRIPTION:")
	var prettyImmediate map[string]interface{}
	if err := json.Unmarshal(immediateMsg, &prettyImmediate); err == nil {
		pretty, _ := json.MarshalIndent(prettyImmediate, "", "  ")
		fmt.Println(string(pretty))
	} else {
		fmt.Println(string(immediateMsg))
	}
	fmt.Println()

	// Check if it's an error
	var immediateType CodexWSMessage
	json.Unmarshal(immediateMsg, &immediateType)
	if immediateType.Type == "error" {
		fmt.Println("❌ SUBSCRIPTION ERROR - Connection will likely close")
		return
	}

	for {
		select {
		case <-timeout:
			fmt.Printf("\n⏱️  Timeout après 30s - Messages reçus: %d\n", messageCount)
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

			messageCount++
			fmt.Printf("━━━━━━━━━━━━━━ Message #%d ━━━━━━━━━━━━━━\n", messageCount)

			// Print raw JSON
			var prettyMsg map[string]interface{}
			if err := json.Unmarshal(messageBytes, &prettyMsg); err == nil {
				pretty, _ := json.MarshalIndent(prettyMsg, "", "  ")
				fmt.Println(string(pretty))
			} else {
				fmt.Println(string(messageBytes))
			}
			fmt.Println()

			// Parse type
			var genericMsg CodexWSMessage
			if err := json.Unmarshal(messageBytes, &genericMsg); err != nil {
				fmt.Printf("⚠️  Failed to parse message: %v\n", err)
				continue
			}

			fmt.Printf("Type: %s\n", genericMsg.Type)
			if genericMsg.ID != "" {
				fmt.Printf("ID: %s\n", genericMsg.ID)
			}

			switch genericMsg.Type {
			case "error":
				fmt.Println("❌ ERROR MESSAGE RECEIVED!")
				if genericMsg.Payload != nil {
					prettyPayload, _ := json.MarshalIndent(genericMsg.Payload, "", "  ")
					fmt.Println(string(prettyPayload))
				}

			case "next":
				fmt.Println("✅ DATA MESSAGE RECEIVED!")
				if genericMsg.Payload != nil {
					prettyPayload, _ := json.MarshalIndent(genericMsg.Payload, "", "  ")
					fmt.Println(string(prettyPayload))
				}

			case "complete":
				fmt.Println("✓ Subscription completed")

			case "ka":
				fmt.Println("💓 Keep-alive")
			}

			fmt.Println()
		}
	}
}
