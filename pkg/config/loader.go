// Package config provides configuration management for the API server.
package config

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zabbix-workshop/nlld/internal/models"

	"gopkg.in/yaml.v3"
)

//go:embed default.yaml
var embeddedConfig embed.FS

const (
	DefaultConfigPath = "config/default.yaml"
	ConfigPathEnv     = "CONFIG_PATH"
)

// LoadDataConfig loads the data configuration from a YAML file.
// It checks for CONFIG_PATH environment variable first, then falls back to DefaultConfigPath,
// and finally uses embedded default config if no file is found.
func LoadDataConfig() (*models.DataConfig, error) {
	configPath := getConfigPath()

	// Try to load from specified path
	if configPath != "" && configPath != DefaultConfigPath {
		data, err := os.ReadFile(configPath)
		if err == nil {
			return parseDataConfig(data)
		}
		fmt.Printf("Warning: Could not read custom config at %s: %v\n", configPath, err)
	}

	// Try default path
	if _, err := os.Stat(DefaultConfigPath); err == nil {
		data, err := os.ReadFile(DefaultConfigPath)
		if err == nil {
			return parseDataConfig(data)
		}
		fmt.Printf("Warning: Could not read default config at %s: %v\n", DefaultConfigPath, err)
	}

	// Fall back to embedded config
	data, err := embeddedConfig.ReadFile("default.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded config: %w", err)
	}
	fmt.Println("Using embedded default configuration")
	return parseDataConfig(data)
}

// getConfigPath returns the path to the configuration file from environment or default
func getConfigPath() string {
	if path := os.Getenv(ConfigPathEnv); path != "" {
		return path
	}
	return DefaultConfigPath
}

// parseDataConfig parses YAML data into DataConfig struct
func parseDataConfig(data []byte) (*models.DataConfig, error) {
	var cfg models.DataConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// GetConfigDir returns the directory containing the configuration file
func GetConfigDir() string {
	configPath := getConfigPath()
	if filepath.IsAbs(configPath) {
		return filepath.Dir(configPath)
	}
	// For relative paths, use current working directory
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, filepath.Dir(configPath))
}
