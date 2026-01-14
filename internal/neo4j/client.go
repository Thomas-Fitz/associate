package neo4j

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// RetryOptions configures the connection retry behavior
type RetryOptions struct {
	// MaxAttempts is the maximum number of connection attempts (default: 30)
	MaxAttempts int
	// InitialDelay is the delay before the first retry (default: 1s)
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries (default: 10s)
	MaxDelay time.Duration
}

// DefaultRetryOptions returns sensible defaults for waiting on Neo4j startup
// These defaults allow up to ~60 seconds for Neo4j to become available
func DefaultRetryOptions() RetryOptions {
	return RetryOptions{
		MaxAttempts:  30,
		InitialDelay: 1 * time.Second,
		MaxDelay:     10 * time.Second,
	}
}

// calculateBackoff returns the delay for the given attempt using exponential backoff
func calculateBackoff(attempt int, initialDelay, maxDelay time.Duration) time.Duration {
	// Exponential backoff: delay = initialDelay * 2^(attempt-1)
	delay := initialDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay > maxDelay {
			return maxDelay
		}
	}
	return delay
}

// retryWithBackoff executes fn with exponential backoff retry logic
func retryWithBackoff(ctx context.Context, opts RetryOptions, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// Don't wait after the last attempt
		if attempt == opts.MaxAttempts {
			break
		}

		delay := calculateBackoff(attempt, opts.InitialDelay, opts.MaxDelay)

		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while retrying: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", opts.MaxAttempts, lastErr)
}

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

// NewClientWithRetry creates a new Neo4j client with retry logic.
// This is useful when the application starts before Neo4j is ready,
// such as in Docker containers without health check dependencies.
func NewClientWithRetry(ctx context.Context, cfg Config, opts *RetryOptions) (*Client, error) {
	if opts == nil {
		defaultOpts := DefaultRetryOptions()
		opts = &defaultOpts
	}

	var client *Client
	err := retryWithBackoff(ctx, *opts, func() error {
		var connectErr error
		client, connectErr = NewClient(ctx, cfg)
		return connectErr
	})

	if err != nil {
		return nil, err
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
