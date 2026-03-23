package config

import (
	"fmt"
	"os"
	"strconv"
)


type Config struct {
	P2P  P2PConfig  `json:"p2p"`
	HTTP HTTPConfig `json:"http"`
}

type P2PConfig struct {
	Port int `json:"port"`
}

type HTTPConfig struct {
	Port int `json:"port"`
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	cfg := &Config{
		P2P:  P2PConfig{Port: 0},
		HTTP: HTTPConfig{Port: 8080},
	}

	if portStr := os.Getenv("P2P_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid P2P_PORT: %w", err)
		}
		cfg.P2P.Port = port
	}

	if portStr := os.Getenv("HTTP_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid HTTP_PORT: %w", err)
		}
		cfg.HTTP.Port = port
	}

	return cfg, nil
}

