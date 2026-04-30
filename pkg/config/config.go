// Package config provides configuration management for the API server.
package config

import (
	"os"
)

// Config holds all application configuration
type Config struct {
	Port     int
	Host     string
	Debug    bool
	DataSeed int64 // For reproducible random data generation
}

// DefaultConfig returns the default configuration values
func DefaultConfig() *Config {
	return &Config{
		Port:     8080,
		Host:     "0.0.0.0",
		Debug:    false,
		DataSeed: 0, // Random by default
	}
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	cfg := DefaultConfig()

	if port := os.Getenv("API_PORT"); port != "" {
		if p, err := parsePort(port); err == nil {
			cfg.Port = p
		}
	}

	if host := os.Getenv("API_HOST"); host != "" {
		cfg.Host = host
	}

	if debug := os.Getenv("DEBUG"); debug != "" {
		cfg.Debug = debug == "true" || debug == "1"
	}

	if seed := os.Getenv("DATA_SEED"); seed != "" {
		if s, err := parseInt64(seed); err == nil {
			cfg.DataSeed = s
		}
	}

	return cfg
}

// parsePort parses a port string to an integer
func parsePort(s string) (int, error) {
	var p int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil // Invalid, return zero
		}
		p = p*10 + int(c-'0')
	}
	if p < 1 || p > 65535 {
		return 0, nil // Out of range
	}
	return p, nil
}

// parseInt64 parses a string to an int64
func parseInt64(s string) (int64, error) {
	var n int64 = 0
	negative := false
	for i, c := range s {
		if i == 0 && c == '-' {
			negative = true
			continue
		}
		if c < '0' || c > '9' {
			return 0, nil // Invalid
		}
		n = n*10 + int64(c-'0')
	}
	if negative {
		n = -n
	}
	return n, nil
}
