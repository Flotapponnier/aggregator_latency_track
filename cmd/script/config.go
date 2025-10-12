package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	CoinGeckoAPIKey string
	MobulaAPIKey    string
}

func loadEnv() (*Config, error) {
	file, err := os.Open(".env")
	if err != nil {
		return nil, fmt.Errorf("error opening .env file: %w", err)
	}
	defer file.Close()

	config := &Config{}

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
		switch key {
		case "COINGECKO_API_KEY":
			config.CoinGeckoAPIKey = value
		case "MOBULA_API_KEY":
			config.MobulaAPIKey = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading .env file: %w", err)
	}

	return config, nil
}
