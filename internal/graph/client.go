package graph

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// GraphName is the name of the AGE graph
const GraphName = "associate"

// RetryOptions configures the connection retry behavior
type RetryOptions struct {
	// MaxAttempts is the maximum number of connection attempts (default: 30)
	MaxAttempts int
	// InitialDelay is the delay before the first retry (default: 1s)
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries (default: 10s)
	MaxDelay time.Duration
}

// DefaultRetryOptions returns sensible defaults for waiting on PostgreSQL startup
func DefaultRetryOptions() RetryOptions {
	return RetryOptions{
		MaxAttempts:  30,
		InitialDelay: 1 * time.Second,
		MaxDelay:     10 * time.Second,
	}
}

// Client wraps the PostgreSQL/AGE connection with application-specific configuration
type Client struct {
	db        *sql.DB
	graphName string
}

// Config holds PostgreSQL/AGE connection configuration
type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
}

// ConfigFromEnv creates a Config from environment variables
func ConfigFromEnv() Config {
	return Config{
		Host:     getEnvOrDefault("DB_HOST", "localhost"),
		Port:     getEnvOrDefault("DB_PORT", "5432"),
		Username: getEnvOrDefault("DB_USERNAME", "associate"),
		Password: getEnvOrDefault("DB_PASSWORD", "password"),
		Database: getEnvOrDefault("DB_DATABASE", "associate"),
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// DSN returns the PostgreSQL connection string
func (c Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.Host, c.Port, c.Username, c.Password, c.Database,
	)
}

// NewClient creates a new AGE client
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connectivity
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	client := &Client{
		db:        db,
		graphName: GraphName,
	}

	// Initialize AGE extension and graph
	if err := client.initAGE(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize AGE: %w", err)
	}

	// Initialize schema (indexes)
	if err := client.initSchema(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return client, nil
}

// NewClientWithRetry creates a new AGE client with retry logic
func NewClientWithRetry(ctx context.Context, cfg Config, opts *RetryOptions) (*Client, error) {
	if opts == nil {
		defaultOpts := DefaultRetryOptions()
		opts = &defaultOpts
	}

	var client *Client
	var lastErr error

	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
		client, lastErr = NewClient(ctx, cfg)
		if lastErr == nil {
			return client, nil
		}

		if attempt == opts.MaxAttempts {
			break
		}

		delay := opts.InitialDelay * time.Duration(1<<(attempt-1))
		if delay > opts.MaxDelay {
			delay = opts.MaxDelay
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
		case <-time.After(delay):
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", opts.MaxAttempts, lastErr)
}

// Close closes the database connection
func (c *Client) Close(ctx context.Context) error {
	return c.db.Close()
}

// DB returns the underlying database connection for direct queries
func (c *Client) DB() *sql.DB {
	return c.db
}

// GraphName returns the graph name
func (c *Client) Graph() string {
	return c.graphName
}

// BeginTx starts a new transaction
func (c *Client) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return c.db.BeginTx(ctx, nil)
}

// initAGE initializes the AGE extension and creates the graph if needed
func (c *Client) initAGE(ctx context.Context) error {
	// Load the AGE extension
	_, err := c.db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS age")
	if err != nil {
		return fmt.Errorf("failed to create AGE extension: %w", err)
	}

	// Load AGE into the search path
	_, err = c.db.ExecContext(ctx, "SET search_path = ag_catalog, \"$user\", public")
	if err != nil {
		return fmt.Errorf("failed to set search path: %w", err)
	}

	// Create the graph if it doesn't exist
	// First check if it exists
	var exists bool
	err = c.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM ag_catalog.ag_graph WHERE name = $1)",
		c.graphName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check graph existence: %w", err)
	}

	if !exists {
		_, err = c.db.ExecContext(ctx, fmt.Sprintf("SELECT create_graph('%s')", c.graphName))
		if err != nil {
			return fmt.Errorf("failed to create graph: %w", err)
		}
	}

	return nil
}

// initSchema creates indexes for the graph on AGE's internal label tables.
// Note: AGE uses agtype columns, which require special handling for indexes.
// For now, we ensure label tables exist but skip indexes (they're an optimization).
func (c *Client) initSchema(ctx context.Context) error {
	// Ensure pg_trgm extension is available for full-text search
	if _, err := c.db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS pg_trgm"); err != nil {
		return fmt.Errorf("failed to create pg_trgm extension: %w", err)
	}

	// Ensure label tables exist by creating and deleting a dummy vertex for each type
	seedLabels := []string{"Memory", "Plan", "Task"}
	for _, label := range seedLabels {
		// Create a seed node
		createQuery := fmt.Sprintf(
			`SELECT * FROM cypher('%s', $$ CREATE (n:%s {id: '__seed__'}) RETURN n $$) as (v agtype)`,
			c.graphName, label)
		if _, err := c.db.ExecContext(ctx, createQuery); err != nil {
			return fmt.Errorf("failed to seed %s label table: %w", label, err)
		}

		// Delete the seed node
		deleteQuery := fmt.Sprintf(
			`SELECT * FROM cypher('%s', $$ MATCH (n:%s {id: '__seed__'}) DELETE n $$) as (v agtype)`,
			c.graphName, label)
		if _, err := c.db.ExecContext(ctx, deleteQuery); err != nil {
			return fmt.Errorf("failed to delete seed %s node: %w", label, err)
		}
	}

	// AGE uses agtype columns, not JSONB. Creating indexes on agtype requires
	// special functions like agtype_access_operator(). For simplicity, we skip
	// index creation - the graph queries will still work, just without index optimization.
	// TODO: Add AGE-compatible indexes if performance becomes an issue

	return nil
}

// execCypher executes a Cypher query and returns the result rows.
// The cypher query should NOT include RETURN if you don't expect results.
// For queries with RETURN, specify the appropriate column definitions.
func (c *Client) execCypher(ctx context.Context, tx *sql.Tx, cypher string, returnCols string) (*sql.Rows, error) {
	query := fmt.Sprintf(
		`SELECT * FROM cypher('%s', $$ %s $$) as (%s)`,
		c.graphName, cypher, returnCols,
	)

	if tx != nil {
		return tx.QueryContext(ctx, query)
	}
	return c.db.QueryContext(ctx, query)
}

// execCypherNoReturn executes a Cypher query that doesn't return rows (e.g., CREATE, DELETE).
func (c *Client) execCypherNoReturn(ctx context.Context, tx *sql.Tx, cypher string) error {
	// For mutations, we use a RETURN with a dummy result since AGE requires it
	query := fmt.Sprintf(
		`SELECT * FROM cypher('%s', $$ %s $$) as (v agtype)`,
		c.graphName, cypher,
	)

	var rows *sql.Rows
	var err error
	if tx != nil {
		rows, err = tx.QueryContext(ctx, query)
	} else {
		rows, err = c.db.QueryContext(ctx, query)
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	// Consume all rows
	for rows.Next() {
	}
	return rows.Err()
}
