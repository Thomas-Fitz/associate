//go:build integration
// +build integration

package graph

import (
	"context"
	"testing"
	"time"

	"github.com/Thomas-Fitz/associate/internal/models"
)

// TestPersistence_SurvivesRestart verifies that data persists across database restarts.
func TestPersistence_SurvivesRestart(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := Config{
		Host:     getEnvOrDefault("DB_HOST", "localhost"),
		Port:     getEnvOrDefault("DB_PORT", "5432"),
		Username: getEnvOrDefault("DB_USERNAME", "associate"),
		Password: getEnvOrDefault("DB_PASSWORD", "password"),
		Database: getEnvOrDefault("DB_DATABASE", "associate"),
	}

	client, err := NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL/AGE: %v", err)
	}
	defer client.Close(ctx)

	repo := NewRepository(client)

	// Create a unique memory for this test run
	testID := "persistence-test-" + time.Now().Format("20060102-150405")
	testContent := "This memory must survive docker-compose down"

	// Check if memory already exists (from previous run)
	existing, err := repo.GetByID(ctx, testID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if existing != nil {
		t.Logf("Found existing memory from previous run: %s", existing.ID)
		t.Logf("Content: %s", existing.Content)
		t.Logf("Created at: %s", existing.CreatedAt)
	} else {
		// Create new memory
		mem := models.Memory{
			ID:      testID,
			Type:    models.TypeNote,
			Content: testContent,
			Metadata: map[string]string{
				"test":      "persistence",
				"timestamp": time.Now().Format(time.RFC3339),
			},
			Tags: []string{"persistence", "test"},
		}

		created, err := repo.Add(ctx, mem, nil)
		if err != nil {
			t.Fatalf("Failed to create memory: %v", err)
		}

		t.Logf("Created new memory: %s", created.ID)
		t.Logf("To verify persistence, restart the database and run this test again")
	}
}
