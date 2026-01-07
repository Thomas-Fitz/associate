package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fitz/associate/internal/config"
)

func TestLoad_WithValidEnvFile(t *testing.T) {
	// Setup: Create temporary .env file
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	envContent := `NEO4J_URI=neo4j://localhost:7687
NEO4J_USERNAME=testuser
NEO4J_PASSWORD=testpass
NEO4J_DATABASE=testdb
`
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}

	// Test: Load configuration
	cfg, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify: Check loaded values
	if cfg.Neo4jURI != "neo4j://localhost:7687" {
		t.Errorf("Expected Neo4jURI to be 'neo4j://localhost:7687', got '%s'", cfg.Neo4jURI)
	}
	if cfg.Neo4jUsername != "testuser" {
		t.Errorf("Expected Neo4jUsername to be 'testuser', got '%s'", cfg.Neo4jUsername)
	}
	if cfg.Neo4jPassword != "testpass" {
		t.Errorf("Expected Neo4jPassword to be 'testpass', got '%s'", cfg.Neo4jPassword)
	}
	if cfg.Neo4jDatabase != "testdb" {
		t.Errorf("Expected Neo4jDatabase to be 'testdb', got '%s'", cfg.Neo4jDatabase)
	}
}

func TestLoad_WithMissingEnvFile(t *testing.T) {
	// Test: Load from directory without .env file but with required env vars
	tmpDir := t.TempDir()
	
	// Set required environment variable for this test
	oldPassword := os.Getenv("NEO4J_PASSWORD")
	os.Setenv("NEO4J_PASSWORD", "testpassword")
	defer func() {
		if oldPassword != "" {
			os.Setenv("NEO4J_PASSWORD", oldPassword)
		} else {
			os.Unsetenv("NEO4J_PASSWORD")
		}
	}()
	
	cfg, err := config.Load(tmpDir)

	// Verify: Should use defaults with system env vars
	if err != nil {
		t.Fatalf("Load() should not error when password is in env: %v", err)
	}

	// Check defaults are applied
	if cfg.Neo4jURI == "" {
		t.Error("Expected default Neo4jURI to be set")
	}
	if cfg.Neo4jDatabase == "" {
		t.Error("Expected default Neo4jDatabase to be set")
	}
	if cfg.Neo4jPassword != "testpassword" {
		t.Errorf("Expected password from env 'testpassword', got '%s'", cfg.Neo4jPassword)
	}
}

func TestLoad_WithMissingRequiredFields(t *testing.T) {
	// Setup: Create .env file missing required password
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	envContent := `NEO4J_URI=neo4j://localhost:7687
NEO4J_USERNAME=testuser
`
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}

	// Test: Load configuration
	_, err := config.Load(tmpDir)

	// Verify: Should return validation error
	if err == nil {
		t.Fatal("Expected validation error for missing NEO4J_PASSWORD")
	}
}

func TestValidate_AllFieldsPresent(t *testing.T) {
	cfg := &config.Config{
		Neo4jURI:      "neo4j://localhost:7687",
		Neo4jUsername: "neo4j",
		Neo4jPassword: "password",
		Neo4jDatabase: "neo4j",
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() should not return error for valid config: %v", err)
	}
}

func TestValidate_MissingURI(t *testing.T) {
	cfg := &config.Config{
		Neo4jUsername: "neo4j",
		Neo4jPassword: "password",
		Neo4jDatabase: "neo4j",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should return error for missing Neo4jURI")
	}
}

func TestValidate_MissingUsername(t *testing.T) {
	cfg := &config.Config{
		Neo4jURI:      "neo4j://localhost:7687",
		Neo4jPassword: "password",
		Neo4jDatabase: "neo4j",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should return error for missing Neo4jUsername")
	}
}

func TestValidate_MissingPassword(t *testing.T) {
	cfg := &config.Config{
		Neo4jURI:      "neo4j://localhost:7687",
		Neo4jUsername: "neo4j",
		Neo4jDatabase: "neo4j",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should return error for missing Neo4jPassword")
	}
}

func TestGetConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	expectedPath := filepath.Join(tmpDir, ".env")

	path := config.GetConfigPath(tmpDir)
	if path != expectedPath {
		t.Errorf("Expected config path '%s', got '%s'", expectedPath, path)
	}
}

func TestSet_UpdatesEnvFile(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	
	// Initially empty
	if err := os.WriteFile(envFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}

	// Test: Set a value
	err := config.Set(tmpDir, "NEO4J_PASSWORD", "newsecret")
	if err != nil {
		t.Fatalf("Set() failed: %v", err)
	}

	// Verify: Value is persisted
	cfg, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() failed after Set(): %v", err)
	}

	if cfg.Neo4jPassword != "newsecret" {
		t.Errorf("Expected password 'newsecret', got '%s'", cfg.Neo4jPassword)
	}
}

func TestGet_RetrievesValue(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	envContent := `NEO4J_USERNAME=myuser`
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}

	// Test: Get a value
	value, err := config.Get(tmpDir, "NEO4J_USERNAME")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	// Verify
	if value != "myuser" {
		t.Errorf("Expected value 'myuser', got '%s'", value)
	}
}

func TestGet_NonExistentKey(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Test: Get non-existent key
	_, err := config.Get(tmpDir, "DOES_NOT_EXIST")
	
	// Verify: Should return error
	if err == nil {
		t.Error("Get() should return error for non-existent key")
	}
}
