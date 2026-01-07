package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// GetGlobalConfigDir returns the path to the global configuration directory.
// This is typically ~/.associate/
func GetGlobalConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home cannot be determined
		return ".associate"
	}
	return filepath.Join(home, ".associate")
}

// GetGlobalConfigPath returns the path to the global configuration file.
func GetGlobalConfigPath() string {
	return filepath.Join(GetGlobalConfigDir(), "config")
}

// EnsureGlobalConfigDir ensures that the global configuration directory exists.
func EnsureGlobalConfigDir() error {
	dir := GetGlobalConfigDir()
	return os.MkdirAll(dir, 0755)
}

// LoadGlobalConfig loads configuration from the global config file.
func LoadGlobalConfig() (*Config, error) {
	configPath := GetGlobalConfigPath()
	
	// Read global config file if it exists
	envMap, err := godotenv.Read(configPath)
	if err != nil {
		// If file doesn't exist, use empty map
		envMap = make(map[string]string)
	}
	
	cfg := &Config{
		Neo4jURI:      getValueOrDefault(envMap, "NEO4J_URI", "neo4j://localhost:7687"),
		Neo4jUsername: getValueOrDefault(envMap, "NEO4J_USERNAME", "neo4j"),
		Neo4jPassword: getValue(envMap, "NEO4J_PASSWORD"),
		Neo4jDatabase: getValueOrDefault(envMap, "NEO4J_DATABASE", "neo4j"),
		Neo4jImage:    getValueOrDefault(envMap, "NEO4J_IMAGE", "neo4j:5.25-community"),
		ContainerName: getValueOrDefault(envMap, "NEO4J_CONTAINER_NAME", "associate-neo4j"),
	}
	
	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	
	return cfg, nil
}

// SetGlobalConfig sets a configuration value in the global config file.
func SetGlobalConfig(key, value string) error {
	// Ensure directory exists
	if err := EnsureGlobalConfigDir(); err != nil {
		return fmt.Errorf("failed to create global config directory: %w", err)
	}
	
	configPath := GetGlobalConfigPath()
	
	// Load existing config
	envMap, err := godotenv.Read(configPath)
	if err != nil {
		// If file doesn't exist, create new map
		envMap = make(map[string]string)
	}
	
	// Update the value
	envMap[key] = value
	
	// Write back to file
	return godotenv.Write(envMap, configPath)
}

// GetGlobalConfig retrieves a configuration value from the global config file.
func GetGlobalConfig(key string) (string, error) {
	configPath := GetGlobalConfigPath()
	
	// Load config
	envMap, err := godotenv.Read(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read global config: %w", err)
	}
	
	value, ok := envMap[key]
	if !ok {
		return "", fmt.Errorf("key '%s' not found in global configuration", key)
	}
	
	return value, nil
}
