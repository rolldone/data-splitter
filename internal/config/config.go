package config

import (
	"fmt"
	"os"

	"data-splitter/pkg/types"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// LoadConfig loads configuration from config.yaml file
func LoadConfig(configPath string) (*types.Config, error) {
	// If no path provided, use default
	if configPath == "" {
		configPath = "config.yaml"
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Optional: load .env file if present, but do not overwrite existing env vars.
	// godotenv.Load will set variables; to avoid overwriting we only load into
	// a map and set missing keys ourselves.
	if _, err := os.Stat(".env"); err == nil {
		if m, err := godotenv.Read(".env"); err == nil {
			for k, v := range m {
				if os.Getenv(k) == "" {
					os.Setenv(k, v)
				}
			}
		}
	}

	// Expand environment variables in the config content so config.yaml can
	// contain placeholders like ${DATABASE_USER} that are populated from the
	// process environment (possibly loaded from .env above).
	expanded := os.Expand(string(data), func(key string) string {
		return os.Getenv(key)
	})

	// Parse YAML
	var config types.Config
	if err := yaml.Unmarshal([]byte(expanded), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// validateConfig performs basic validation on the configuration
func validateConfig(config *types.Config) error {
	if config.Database.Type == "" {
		return fmt.Errorf("database.type is required")
	}

	if config.Database.Host == "" {
		return fmt.Errorf("database.host is required")
	}

	if config.Database.SourceDB == "" {
		return fmt.Errorf("database.source_db is required")
	}

	if len(config.Tables) == 0 {
		return fmt.Errorf("at least one table must be configured")
	}

	if len(config.Archive.Years) == 0 {
		return fmt.Errorf("at least one year must be specified in archive.years")
	}

	// Validate each table
	for i, table := range config.Tables {
		if table.Name == "" {
			return fmt.Errorf("table[%d].name is required", i)
		}
		if table.Enabled && table.SplitColumn == "" {
			return fmt.Errorf("table[%d].split_column is required when enabled", i)
		}
	}

	return nil
}
