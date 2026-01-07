package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetGlobalConfigDir(t *testing.T) {
	dir := GetGlobalConfigDir()
	
	// Should be in home directory
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}
	
	expected := filepath.Join(home, ".associate")
	if dir != expected {
		t.Errorf("expected %s, got %s", expected, dir)
	}
}

func TestEnsureGlobalConfigDir(t *testing.T) {
	// Create a temporary home for testing
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)
	
	err := EnsureGlobalConfigDir()
	if err != nil {
		t.Fatalf("failed to ensure global config dir: %v", err)
	}
	
	// Check that directory was created
	expectedDir := filepath.Join(tempHome, ".associate")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("global config directory was not created at %s", expectedDir)
	}
}

func TestLoadGlobalConfig_WithValidFile(t *testing.T) {
	// Create a temporary home for testing
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)
	
	// Create config directory
	configDir := filepath.Join(tempHome, ".associate")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	
	// Write global config
	configPath := filepath.Join(configDir, "config")
	content := `NEO4J_URI=neo4j://test:7687
NEO4J_USERNAME=testuser
NEO4J_PASSWORD=testpass
NEO4J_DATABASE=testdb
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	
	// Load global config
	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("failed to load global config: %v", err)
	}
	
	// Verify values
	if cfg.Neo4jURI != "neo4j://test:7687" {
		t.Errorf("expected neo4j://test:7687, got %s", cfg.Neo4jURI)
	}
	if cfg.Neo4jUsername != "testuser" {
		t.Errorf("expected testuser, got %s", cfg.Neo4jUsername)
	}
	if cfg.Neo4jPassword != "testpass" {
		t.Errorf("expected testpass, got %s", cfg.Neo4jPassword)
	}
}

func TestLoadGlobalConfig_WithMissingFile(t *testing.T) {
	// Create a temporary home for testing
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)
	
	// Attempt to load (file doesn't exist)
	cfg, err := LoadGlobalConfig()
	
	// Should return config with defaults, but fail validation
	if err == nil {
		t.Error("expected error for missing password, got nil")
	}
	
	if cfg == nil {
		t.Error("expected config struct even with error, got nil")
	}
}

func TestSetGlobalConfig(t *testing.T) {
	// Create a temporary home for testing
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)
	
	// Set a value
	err := SetGlobalConfig("NEO4J_PASSWORD", "newsecret")
	if err != nil {
		t.Fatalf("failed to set global config: %v", err)
	}
	
	// Load and verify
	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("failed to load global config after set: %v", err)
	}
	
	if cfg.Neo4jPassword != "newsecret" {
		t.Errorf("expected newsecret, got %s", cfg.Neo4jPassword)
	}
}

func TestGetGlobalConfig(t *testing.T) {
	// Create a temporary home for testing
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)
	
	// Set a value
	err := SetGlobalConfig("NEO4J_URI", "neo4j://global:7687")
	if err != nil {
		t.Fatalf("failed to set global config: %v", err)
	}
	
	// Get the value
	value, err := GetGlobalConfig("NEO4J_URI")
	if err != nil {
		t.Fatalf("failed to get global config: %v", err)
	}
	
	if value != "neo4j://global:7687" {
		t.Errorf("expected neo4j://global:7687, got %s", value)
	}
}

func TestGetGlobalConfig_NonExistentKey(t *testing.T) {
	// Create a temporary home for testing
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)
	
	// Get non-existent key
	_, err := GetGlobalConfig("DOES_NOT_EXIST")
	if err == nil {
		t.Error("expected error for non-existent key, got nil")
	}
}

func TestLoadWithGlobalFallback(t *testing.T) {
	// Create a temporary home for testing
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)
	
	// Set global password
	err := SetGlobalConfig("NEO4J_PASSWORD", "globalpass")
	if err != nil {
		t.Fatalf("failed to set global config: %v", err)
	}
	
	// Create a temp directory for local config (without password)
	localDir := t.TempDir()
	localEnv := filepath.Join(localDir, ".env")
	localContent := `NEO4J_URI=neo4j://local:7687`
	if err := os.WriteFile(localEnv, []byte(localContent), 0644); err != nil {
		t.Fatalf("failed to write local config: %v", err)
	}
	
	// Load config - should merge local + global
	cfg, err := Load(localDir)
	if err != nil {
		t.Fatalf("failed to load config with global fallback: %v", err)
	}
	
	// Local URI should be used
	if cfg.Neo4jURI != "neo4j://local:7687" {
		t.Errorf("expected local URI, got %s", cfg.Neo4jURI)
	}
	
	// Global password should be used
	if cfg.Neo4jPassword != "globalpass" {
		t.Errorf("expected global password, got %s", cfg.Neo4jPassword)
	}
}
