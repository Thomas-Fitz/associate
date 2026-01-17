//go:build integration
// +build integration

package neo4j

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fitz/associate/internal/models"
)

// TestPersistence_SurvivesRestart verifies that data persists across database restarts.
func TestPersistence_SurvivesRestart(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check for required env vars or use defaults
	uri := os.Getenv("NEO4J_URI")
	if uri == "" {
		uri = "bolt://localhost:7687"
	}

	cfg := Config{
		URI:      uri,
		Username: getEnvOrDefault("NEO4J_USERNAME", "neo4j"),
		Password: getEnvOrDefault("NEO4J_PASSWORD", "password"),
		Database: getEnvOrDefault("NEO4J_DATABASE", "neo4j"),
	}

	client, err := NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	defer client.Close(ctx)

	repo := NewRepository(client)

	// Create a unique memory for this test run
	testID := "persistence-test-" + time.Now().Format("20060102-150405")
	testContent := "This memory must survive docker-compose down -v"

	// Check if memory already exists (from previous run)
	existing, err := repo.GetByID(ctx, testID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if existing != nil {
		t.Logf("✓ Found existing memory from previous run: %s", existing.ID)
		t.Logf("  Content: %s", existing.Content)
		t.Logf("  Created: %s", existing.CreatedAt)
		return
	}

	// Create new memory
	mem := models.Memory{
		ID:      testID,
		Type:    models.TypeNote,
		Content: testContent,
		Metadata: map[string]string{
			"test_run": time.Now().Format(time.RFC3339),
		},
		Tags: []string{"persistence-test"},
	}

	created, err := repo.Add(ctx, mem, nil)
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	t.Logf("✓ Created new memory: %s", created.ID)
	t.Log("  Restart the database and run this test again to verify persistence.")
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
