package neo4j

import (
	"context"
	"fmt"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Client wraps the Neo4j driver with application-specific configuration
type Client struct {
	driver neo4j.DriverWithContext
	db     string
}

// Config holds Neo4j connection configuration
type Config struct {
	URI      string
	Username string
	Password string
	Database string
}

// ConfigFromEnv creates a Config from environment variables
func ConfigFromEnv() Config {
	uri := os.Getenv("NEO4J_URI")
	if uri == "" {
		uri = "bolt://localhost:7687"
	}
	username := os.Getenv("NEO4J_USERNAME")
	if username == "" {
		username = "neo4j"
	}
	password := os.Getenv("NEO4J_PASSWORD")
	if password == "" {
		password = "password"
	}
	database := os.Getenv("NEO4J_DATABASE")
	if database == "" {
		database = "neo4j"
	}
	return Config{
		URI:      uri,
		Username: username,
		Password: password,
		Database: database,
	}
}

// NewClient creates a new Neo4j client
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	driver, err := neo4j.NewDriverWithContext(
		cfg.URI,
		neo4j.BasicAuth(cfg.Username, cfg.Password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create driver: %w", err)
	}

	// Verify connectivity
	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(ctx)
		return nil, fmt.Errorf("failed to connect to Neo4j: %w", err)
	}

	client := &Client{
		driver: driver,
		db:     cfg.Database,
	}

	// Initialize schema
	if err := client.initSchema(ctx); err != nil {
		driver.Close(ctx)
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return client, nil
}

// Close closes the Neo4j driver
func (c *Client) Close(ctx context.Context) error {
	return c.driver.Close(ctx)
}

// Session returns a new Neo4j session
func (c *Client) Session(ctx context.Context) neo4j.SessionWithContext {
	return c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.db,
	})
}

// initSchema creates indexes and constraints for the graph
func (c *Client) initSchema(ctx context.Context) error {
	session := c.Session(ctx)
	defer session.Close(ctx)

	// Create full-text index for content search
	queries := []string{
		`CREATE INDEX memory_id IF NOT EXISTS FOR (m:Memory) ON (m.id)`,
		`CREATE INDEX memory_type IF NOT EXISTS FOR (m:Memory) ON (m.type)`,
		`CREATE FULLTEXT INDEX memory_content IF NOT EXISTS FOR (m:Memory) ON EACH [m.content]`,
	}

	for _, query := range queries {
		_, err := session.Run(ctx, query, nil)
		if err != nil {
			return fmt.Errorf("failed to run schema query %q: %w", query, err)
		}
	}

	return nil
}
