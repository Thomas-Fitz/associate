// Package config manages application configuration from environment variables and .env files.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds the application configuration.
type Config struct {
	Neo4jURI       string
	Neo4jUsername  string
	Neo4jPassword  string
	Neo4jDatabase  string
	Neo4jImage     string
	ContainerName  string
	CopilotToken   string
}

// Load reads configuration from a .env file in the specified directory.
// If the .env file doesn't exist, it falls back to global config (~/.associate/config),
// then to environment variables and defaults.
func Load(dir string) (*Config, error) {
	envPath := GetConfigPath(dir)
	
	// Read local .env file if it exists
	localEnvMap, err := godotenv.Read(envPath)
	if err != nil {
		// If file doesn't exist, use empty map
		localEnvMap = make(map[string]string)
	}
	
	// Read global config file
	globalEnvMap, err := godotenv.Read(GetGlobalConfigPath())
	if err != nil {
		// If file doesn't exist, use empty map
		globalEnvMap = make(map[string]string)
	}

	// Helper to get value with precedence: local > global > env > default
	getValueWithFallback := func(key, defaultValue string) string {
		// Check local first
		if value, ok := localEnvMap[key]; ok && value != "" {
			return value
		}
		// Check global
		if value, ok := globalEnvMap[key]; ok && value != "" {
			return value
		}
		// Check environment
		if value := os.Getenv(key); value != "" {
			return value
		}
		// Return default
		return defaultValue
	}
	
	getValueWithFallbackNoDefault := func(key string) string {
		// Check local first
		if value, ok := localEnvMap[key]; ok && value != "" {
			return value
		}
		// Check global
		if value, ok := globalEnvMap[key]; ok && value != "" {
			return value
		}
		// Check environment
		return os.Getenv(key)
	}

	cfg := &Config{
		Neo4jURI:      getValueWithFallback("NEO4J_URI", "neo4j://localhost:7687"),
		Neo4jUsername: getValueWithFallback("NEO4J_USERNAME", "neo4j"),
		Neo4jPassword: getValueWithFallbackNoDefault("NEO4J_PASSWORD"),
		Neo4jDatabase: getValueWithFallback("NEO4J_DATABASE", "neo4j"),
		Neo4jImage:    getValueWithFallback("NEO4J_IMAGE", "neo4j:5.25-community"),
		ContainerName: getValueWithFallback("NEO4J_CONTAINER_NAME", "associate-neo4j"),
		CopilotToken:  getValueWithFallbackNoDefault("GITHUB_COPILOT_TOKEN"),
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that all required configuration fields are set.
func (c *Config) Validate() error {
	var missing []string

	if c.Neo4jURI == "" {
		missing = append(missing, "NEO4J_URI")
	}
	if c.Neo4jUsername == "" {
		missing = append(missing, "NEO4J_USERNAME")
	}
	if c.Neo4jPassword == "" {
		missing = append(missing, "NEO4J_PASSWORD")
	}
	if c.Neo4jDatabase == "" {
		missing = append(missing, "NEO4J_DATABASE")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration fields: %s", strings.Join(missing, ", "))
	}

	return nil
}

// GetConfigPath returns the full path to the .env file in the given directory.
func GetConfigPath(dir string) string {
	return filepath.Join(dir, ".env")
}

// Set updates or creates a configuration value in the .env file.
func Set(dir, key, value string) error {
	envPath := GetConfigPath(dir)
	
	// Load existing config
	envMap, err := godotenv.Read(envPath)
	if err != nil {
		// If file doesn't exist, create new map
		envMap = make(map[string]string)
	}

	// Update the value
	envMap[key] = value

	// Write back to file
	return godotenv.Write(envMap, envPath)
}

// Get retrieves a configuration value from the .env file.
func Get(dir, key string) (string, error) {
	envPath := GetConfigPath(dir)
	
	// Load config
	envMap, err := godotenv.Read(envPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config: %w", err)
	}

	value, ok := envMap[key]
	if !ok {
		return "", fmt.Errorf("key '%s' not found in configuration", key)
	}

	return value, nil
}

// getValue gets a value from the env map, falling back to system env var.
func getValue(envMap map[string]string, key string) string {
	if value, ok := envMap[key]; ok && value != "" {
		return value
	}
	return os.Getenv(key)
}

// getValueOrDefault gets a value from env map, falling back to system env var, then default.
func getValueOrDefault(envMap map[string]string, key, defaultValue string) string {
	if value, ok := envMap[key]; ok && value != "" {
		return value
	}
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
