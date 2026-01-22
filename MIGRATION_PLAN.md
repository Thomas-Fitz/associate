# Migration Plan: Neo4j to Apache AGE

This document outlines the complete migration of the Associate MCP server from Neo4j graph database to Apache AGE (PostgreSQL extension).

## Overview

| Aspect | Current (Neo4j) | Target (Apache AGE) |
|--------|-----------------|---------------------|
| Database | Neo4j 5 Community | PostgreSQL 17 + AGE 1.6.0 |
| Protocol | Bolt (7687) | PostgreSQL (5432) |
| Driver | `neo4j/neo4j-go-driver/v5` | `apache/age/drivers/golang/age` |
| Query Format | Direct Cypher | Cypher wrapped in `cypher()` SQL function |
| Full-Text Search | Native FULLTEXT INDEX | pg_trgm (trigram-based fuzzy search) |
| Visualization | Neo4j Browser (:7474) | Apache AGE Viewer (:3000) |
| Graph Name | N/A (database-level) | `associate` |

## Decisions

- **PostgreSQL version**: 17 with AGE 1.6.0
- **Full-text search**: pg_trgm extension for all node types (Memory, Plan, Task). Hard dependency — no fallback.
- **Go driver**: Official AGE driver (fallback to lib/pq if problematic)
- **Migration approach**: Full cutover (drop Neo4j entirely)
- **Package naming**: Generic (`internal/graph/`)
- **Environment variables**: Generic (`DB_HOST`, `DB_PORT`, `DB_USERNAME`, `DB_PASSWORD`, `DB_DATABASE`)
- **Graph name**: `associate`
- **Visualization**: Apache AGE Viewer included in docker-compose
- **Timestamps**: RFC3339 strings (lexicographically sortable, no native datetime needed)
- **Idempotent relationships**: Check-then-create pattern (accept small race condition risk; duplicates are harmless due to query-level dedup)
- **Node type detection**: Store explicit `node_type` property on every node (do not rely on AGE `labels()` function)
- **Transaction rollback**: `defer tx.Rollback()` pattern — AGE transactions handle atomicity natively, no best-effort delete needed
- **Indexes**: Created on AGE internal label tables in `initSchema()` using raw SQL

---

## Phase 1: Infrastructure & Configuration

### Task 1.1: Update Docker Compose

**File**: `docker-compose.yml`

Replace Neo4j service with PostgreSQL/AGE and add AGE Viewer:

```yaml
services:
  postgres:
    image: apache/age:PG17_latest
    container_name: associate-postgres
    environment:
      - POSTGRES_USER=associate
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=associate
    ports:
      - "5432:5432"
    volumes:
      - associate_postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U associate -d associate"]
      interval: 10s
      timeout: 5s
      retries: 5

  age-viewer:
    image: apache/age-viewer:latest
    container_name: associate-age-viewer
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=production

  associate:
    build: .
    container_name: associate-mcp
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USERNAME=associate
      - DB_PASSWORD=password
      - DB_DATABASE=associate
    ports:
      - "8080:8080"
    command: ["-http"]
    restart: unless-stopped

volumes:
  associate_postgres_data:
    name: associate_postgres_data
```

### Task 1.2: Update Dockerfile

**File**: `Dockerfile`

Update environment variable defaults:

```dockerfile
# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git for go mod download
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /associate ./cmd/associate

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

COPY --from=builder /associate /associate

# Default environment for PostgreSQL/AGE connection
ENV DB_HOST=postgres
ENV DB_PORT=5432
ENV DB_USERNAME=associate
ENV DB_PASSWORD=password
ENV DB_DATABASE=associate

EXPOSE 8080

ENTRYPOINT ["/associate"]
```

### Task 1.3: Update Go Dependencies

**File**: `go.mod`

```go
module github.com/fitz/associate

go 1.25.5

require (
    github.com/apache/age/drivers/golang v0.0.0  // Use latest release tag
    github.com/google/uuid v1.6.0
    github.com/lib/pq v1.10.9
    github.com/modelcontextprotocol/go-sdk v1.2.0
)

// Remove: github.com/neo4j/neo4j-go-driver/v5
```

Run:
```bash
go get github.com/apache/age/drivers/golang@latest
go get github.com/lib/pq@latest
go mod tidy
```

---

## Phase 2: Rename and Restructure Package

### Task 2.1: Rename Package Directory

```bash
git mv internal/neo4j internal/graph
```

### Task 2.2: Update All Import Paths

Update imports in all files that reference `internal/neo4j`:

**Files to update**:
- `cmd/associate/main.go`
- `internal/mcp/server.go`
- `internal/mcp/plan_task_test.go`
- Any other files importing the neo4j package

Change:
```go
"github.com/fitz/associate/internal/neo4j"
```
To:
```go
"github.com/fitz/associate/internal/graph"
```

---

## Phase 3: Client Layer Rewrite

### Task 3.1: Create New Client

**File**: `internal/graph/client.go`

```go
package graph

import (
    "context"
    "database/sql"
    "fmt"
    "os"
    "time"

    "github.com/apache/age/drivers/golang/age"
    _ "github.com/lib/pq"
)

const GraphName = "associate"

// RetryOptions configures the connection retry behavior
type RetryOptions struct {
    MaxAttempts  int
    InitialDelay time.Duration
    MaxDelay     time.Duration
}

// DefaultRetryOptions returns sensible defaults for waiting on PostgreSQL startup
func DefaultRetryOptions() RetryOptions {
    return RetryOptions{
        MaxAttempts:  30,
        InitialDelay: 1 * time.Second,
        MaxDelay:     10 * time.Second,
    }
}

// Client wraps the AGE connection with application-specific configuration
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

    // Initialize AGE extension and graph
    if _, err := age.GetReady(db, GraphName); err != nil {
        db.Close()
        return nil, fmt.Errorf("failed to initialize AGE: %w", err)
    }

    client := &Client{
        db:        db,
        graphName: GraphName,
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
func (c *Client) GraphName() string {
    return c.graphName
}

// BeginTx starts a new transaction
func (c *Client) BeginTx(ctx context.Context) (*sql.Tx, error) {
    return c.db.BeginTx(ctx, nil)
}

// initSchema creates indexes for the graph on AGE's internal label tables.
// AGE stores vertices in label-specific tables (e.g., associate."Memory") with a
// `properties` JSONB column. We create PostgreSQL indexes on these tables for
// efficient lookups and pg_trgm indexes for full-text search.
//
// Prerequisites: pg_trgm extension must be available (hard dependency).
func (c *Client) initSchema(ctx context.Context) error {
    // Ensure pg_trgm extension is available
    if _, err := c.db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS pg_trgm"); err != nil {
        return fmt.Errorf("failed to create pg_trgm extension: %w", err)
    }

    // Ensure label tables exist by creating and deleting a dummy vertex for each type
    // (AGE creates the table on first vertex creation for a label)
    seedQueries := []string{
        fmt.Sprintf(`SELECT * FROM cypher('%s', $$ CREATE (n:Memory {id: '__seed__'}) RETURN n $$) as (v agtype)`, GraphName),
        fmt.Sprintf(`SELECT * FROM cypher('%s', $$ MATCH (n:Memory {id: '__seed__'}) DELETE n $$) as (v agtype)`, GraphName),
        fmt.Sprintf(`SELECT * FROM cypher('%s', $$ CREATE (n:Plan {id: '__seed__'}) RETURN n $$) as (v agtype)`, GraphName),
        fmt.Sprintf(`SELECT * FROM cypher('%s', $$ MATCH (n:Plan {id: '__seed__'}) DELETE n $$) as (v agtype)`, GraphName),
        fmt.Sprintf(`SELECT * FROM cypher('%s', $$ CREATE (n:Task {id: '__seed__'}) RETURN n $$) as (v agtype)`, GraphName),
        fmt.Sprintf(`SELECT * FROM cypher('%s', $$ MATCH (n:Task {id: '__seed__'}) DELETE n $$) as (v agtype)`, GraphName),
    }

    for _, q := range seedQueries {
        if _, err := c.db.ExecContext(ctx, q); err != nil {
            return fmt.Errorf("failed to seed label table: %w", err)
        }
    }

    // Create B-tree indexes on id and status properties for fast lookups
    // Create GIN pg_trgm indexes on content/name fields for search
    indexQueries := []string{
        // Memory indexes
        fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_memory_id ON %s."Memory" ((properties->>'id'))`, GraphName),
        fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_memory_type ON %s."Memory" ((properties->>'type'))`, GraphName),
        fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_memory_content_trgm ON %s."Memory" USING GIN ((properties->>'content') gin_trgm_ops)`, GraphName),
        // Plan indexes
        fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_plan_id ON %s."Plan" ((properties->>'id'))`, GraphName),
        fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_plan_status ON %s."Plan" ((properties->>'status'))`, GraphName),
        fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_plan_name_trgm ON %s."Plan" USING GIN ((properties->>'name') gin_trgm_ops)`, GraphName),
        fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_plan_desc_trgm ON %s."Plan" USING GIN ((properties->>'description') gin_trgm_ops)`, GraphName),
        // Task indexes
        fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_task_id ON %s."Task" ((properties->>'id'))`, GraphName),
        fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_task_status ON %s."Task" ((properties->>'status'))`, GraphName),
        fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_task_content_trgm ON %s."Task" USING GIN ((properties->>'content') gin_trgm_ops)`, GraphName),
    }

    for _, q := range indexQueries {
        if _, err := c.db.ExecContext(ctx, q); err != nil {
            return fmt.Errorf("failed to create index: %w", err)
        }
    }

    return nil
}
```

### Task 3.2: Create Query Helper Functions

**File**: `internal/graph/helpers.go`

```go
package graph

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "github.com/apache/age/drivers/golang/age"
    "github.com/fitz/associate/internal/models"
)

// ExecCypher is a convenience wrapper for age.ExecCypher
func ExecCypher(tx *sql.Tx, graphName string, numResults int, cypher string, args ...interface{}) (*age.Cursor, error) {
    return age.ExecCypher(tx, graphName, numResults, cypher, args...)
}

// NOTE: All nodes store a `node_type` property ('Memory', 'Plan', 'Task') for reliable
// type detection in cross-type queries (e.g., GetRelated). This avoids depending on
// AGE's labels() function which may have compatibility issues. The Vertex-to-struct
// converters below ignore this property since it's only needed for type dispatch.

// VertexToMemory converts an AGE Vertex to a Memory struct
func VertexToMemory(v *age.Vertex) models.Memory {
    props := v.Props()
    
    mem := models.Memory{
        ID:      getStringProp(props, "id"),
        Type:    models.MemoryType(getStringProp(props, "type")),
        Content: getStringProp(props, "content"),
    }

    // Parse metadata from JSON string
    if metaStr := getStringProp(props, "metadata"); metaStr != "" {
        var meta map[string]string
        if err := json.Unmarshal([]byte(metaStr), &meta); err == nil {
            mem.Metadata = meta
        }
    }

    // Parse tags from JSON array
    if tagsRaw, ok := props["tags"]; ok {
        if tagsArr, ok := tagsRaw.([]interface{}); ok {
            for _, t := range tagsArr {
                if s, ok := t.(string); ok {
                    mem.Tags = append(mem.Tags, s)
                }
            }
        }
    }

    // Parse timestamps
    if createdStr := getStringProp(props, "created_at"); createdStr != "" {
        if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
            mem.CreatedAt = t
        }
    }
    if updatedStr := getStringProp(props, "updated_at"); updatedStr != "" {
        if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
            mem.UpdatedAt = t
        }
    }

    return mem
}

// VertexToPlan converts an AGE Vertex to a Plan struct
func VertexToPlan(v *age.Vertex) models.Plan {
    props := v.Props()
    
    plan := models.Plan{
        ID:          getStringProp(props, "id"),
        Name:        getStringProp(props, "name"),
        Description: getStringProp(props, "description"),
        Status:      models.PlanStatus(getStringProp(props, "status")),
    }

    // Parse metadata
    if metaStr := getStringProp(props, "metadata"); metaStr != "" {
        var meta map[string]string
        if err := json.Unmarshal([]byte(metaStr), &meta); err == nil {
            plan.Metadata = meta
        }
    }

    // Parse tags
    if tagsRaw, ok := props["tags"]; ok {
        if tagsArr, ok := tagsRaw.([]interface{}); ok {
            for _, t := range tagsArr {
                if s, ok := t.(string); ok {
                    plan.Tags = append(plan.Tags, s)
                }
            }
        }
    }

    // Parse timestamps
    if createdStr := getStringProp(props, "created_at"); createdStr != "" {
        if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
            plan.CreatedAt = t
        }
    }
    if updatedStr := getStringProp(props, "updated_at"); updatedStr != "" {
        if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
            plan.UpdatedAt = t
        }
    }

    return plan
}

// VertexToTask converts an AGE Vertex to a Task struct
func VertexToTask(v *age.Vertex) models.Task {
    props := v.Props()
    
    task := models.Task{
        ID:      getStringProp(props, "id"),
        Content: getStringProp(props, "content"),
        Status:  models.TaskStatus(getStringProp(props, "status")),
    }

    // Parse metadata
    if metaStr := getStringProp(props, "metadata"); metaStr != "" {
        var meta map[string]string
        if err := json.Unmarshal([]byte(metaStr), &meta); err == nil {
            task.Metadata = meta
        }
    }

    // Parse tags
    if tagsRaw, ok := props["tags"]; ok {
        if tagsArr, ok := tagsRaw.([]interface{}); ok {
            for _, t := range tagsArr {
                if s, ok := t.(string); ok {
                    task.Tags = append(task.Tags, s)
                }
            }
        }
    }

    // Parse timestamps
    if createdStr := getStringProp(props, "created_at"); createdStr != "" {
        if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
            task.CreatedAt = t
        }
    }
    if updatedStr := getStringProp(props, "updated_at"); updatedStr != "" {
        if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
            task.UpdatedAt = t
        }
    }

    return task
}

// Helper functions

func getStringProp(props map[string]interface{}, key string) string {
    if v, ok := props[key].(string); ok {
        return v
    }
    return ""
}

func getFloat64Prop(props map[string]interface{}, key string) float64 {
    if v, ok := props[key].(float64); ok {
        return v
    }
    return 0
}

func metadataToJSON(m map[string]string) string {
    if len(m) == 0 {
        return ""
    }
    b, err := json.Marshal(m)
    if err != nil {
        return ""
    }
    return string(b)
}

func tagsToJSON(tags []string) string {
    if len(tags) == 0 {
        return "[]"
    }
    b, err := json.Marshal(tags)
    if err != nil {
        return "[]"
    }
    return string(b)
}

func joinStrings(strs []string, sep string) string {
    return strings.Join(strs, sep)
}

// EscapeCypherString escapes a string for use in Cypher queries
// This prevents injection attacks when using string formatting
func EscapeCypherString(s string) string {
    s = strings.ReplaceAll(s, "\\", "\\\\")
    s = strings.ReplaceAll(s, "'", "\\'")
    s = strings.ReplaceAll(s, "\"", "\\\"")
    return s
}
```

---

## Phase 3.5: AGE Cypher Compatibility Validation

Before implementing the repository layer, validate that all required Cypher features work correctly in AGE. Create a test file that exercises each feature against a running AGE instance.

### Task 3.5.1: Create Validation Test

**File**: `internal/graph/age_compat_test.go`

This test should validate the following Cypher features used by the application:

| Feature | Example | Used In |
|---------|---------|---------|
| `labels(node)` | `labels(b) as nodeLabels` | GetRelated (fallback only, we use node_type property) |
| `startNode(rel)` | `startNode(r[-1]) = a` | GetRelated direction detection |
| `type(rel)` | `type(r[-1]) as rel_type` | GetRelated, GetByIDWithRelated |
| `r[-1]` (list indexing) | `r[-1]` on variable-length path | GetRelated |
| `size(list)` | `size(r) as depth` | GetRelated |
| `COALESCE()` | `COALESCE(r.position, 0.0)` | Task position queries |
| `collect(DISTINCT ...)` | `collect(DISTINCT dep.id)` | GetWithTasks, Search |
| Variable-length paths | `[r*1..N]` | GetRelated |
| `OPTIONAL MATCH` | Multiple patterns | GetByIDWithRelated, GetWithTasks |
| Map literals in RETURN | `{id: n.id, type: n.type}` | GetByIDWithRelated |
| `ILIKE` (via pg_trgm) | `m.content ILIKE '%query%'` | Search |
| Relationship properties | `r.position` on PART_OF | Task ordering |
| `SET` on relationship | `SET r.position = $pos` | UpdatePositions |
| `FOREACH` | `FOREACH (x IN list \| ...)` | Plan cascade delete |
| `EXISTS {}` subquery | `NOT EXISTS { MATCH ... }` | Plan cascade delete |

```go
func TestAGECypherCompatibility(t *testing.T) {
    client, ctx, cancel := getTestClient(t)
    defer cancel()
    defer client.Close(ctx)

    tx, err := client.BeginTx(ctx)
    require.NoError(t, err)
    defer tx.Rollback()

    // Test 1: Variable-length paths with list indexing
    // Create: A -> B -> C, then query with r[-1] and size(r)
    
    // Test 2: startNode() function
    
    // Test 3: type() function on relationships
    
    // Test 4: COALESCE with relationship properties
    
    // Test 5: collect(DISTINCT ...) with map literals
    
    // Test 6: OPTIONAL MATCH returning nulls
    
    // Test 7: ILIKE operator (pg_trgm)
    
    // Test 8: SET on relationship properties
    
    // Test 9: FOREACH with DETACH DELETE
    
    // Test 10: NOT EXISTS {} subquery pattern

    // If any test fails, document the alternative approach needed
}
```

### Task 3.5.2: Document Alternatives

For any feature that fails validation, document the alternative approach:

- **If `startNode(r[-1])` fails**: Use separate queries for outgoing and incoming, merge results in Go
- **If `r[-1]` list indexing fails**: Use `last(r)` or unwind the path
- **If `FOREACH` fails**: Use multiple individual DELETE statements in a loop
- **If `NOT EXISTS {}` fails**: Use `OPTIONAL MATCH ... WHERE other IS NULL` pattern
- **If `ILIKE` fails in Cypher**: Use raw SQL query against the AGE label table directly
- **If map literals in collect fail**: Return individual columns and assemble in Go code

---

## Phase 4: Repository Layer Migration

### Task 4.1: Migrate Memory Repository

**File**: `internal/graph/repository.go`

Key query translations:

#### Search (pg_trgm with related IDs)

**Neo4j**:
```cypher
CALL db.index.fulltext.queryNodes('memory_content', $query) 
YIELD node, score
OPTIONAL MATCH (node)-[:RELATES_TO|PART_OF|REFERENCES|DEPENDS_ON|BLOCKS|FOLLOWS|IMPLEMENTS]-(related:Memory)
RETURN node, score, collect(DISTINCT related.id) as related_ids
```

**AGE**:

> **Important**: Search uses pg_trgm for case-insensitive fuzzy matching with similarity scoring.
> This requires pg_trgm indexes on the AGE label tables (created in `initSchema()`).
> Also searches by ID (preserving current fallback behavior).
> No fallback mechanism — pg_trgm is a hard dependency.

```go
func (r *Repository) Search(ctx context.Context, query string, limit int) ([]models.SearchResult, error) {
    if limit <= 0 {
        limit = 10
    }

    tx, err := r.client.BeginTx(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    escapedQuery := EscapeCypherString(query)
    
    // Search content and ID using pg_trgm similarity (via ILIKE for trigram index usage)
    // Also collect related Memory IDs (preserves API response shape)
    cursor, err := ExecCypher(tx, r.client.GraphName(), 2,
        `MATCH (m:Memory) 
         WHERE m.content ILIKE '%%%s%%' OR m.id ILIKE '%%%s%%'
         OPTIONAL MATCH (m)-[:RELATES_TO|PART_OF|REFERENCES|DEPENDS_ON|BLOCKS|FOLLOWS|IMPLEMENTS]-(related:Memory)
         RETURN m, collect(DISTINCT related.id) as related_ids
         LIMIT %d`,
        escapedQuery, escapedQuery, limit)
    if err != nil {
        return nil, fmt.Errorf("search failed: %w", err)
    }

    var results []models.SearchResult
    for cursor.Next() {
        row, err := cursor.GetRow()
        if err != nil {
            return nil, err
        }
        vertex := row[0].(*age.Vertex)
        mem := VertexToMemory(vertex)
        
        sr := models.SearchResult{
            Memory: mem,
            Score:  1.0, // pg_trgm ILIKE doesn't provide scoring; use similarity() for ranked results if needed
        }
        
        // Extract related IDs
        if relatedIDs, ok := row[1].([]interface{}); ok {
            for _, id := range relatedIDs {
                if s, ok := id.(string); ok && s != "" {
                    sr.Related = append(sr.Related, s)
                }
            }
        }
        
        results = append(results, sr)
    }

    tx.Commit()
    return results, nil
}
```

#### Add Memory

**Neo4j**:
```cypher
CREATE (m:Memory {id: $id, type: $type, content: $content, ...})
RETURN m
```

**AGE**:
```go
func (r *Repository) Add(ctx context.Context, mem models.Memory, relationships []models.Relationship) (*models.Memory, error) {
    tx, err := r.client.BeginTx(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    if mem.ID == "" {
        mem.ID = uuid.New().String()
    }
    now := time.Now().UTC()
    mem.CreatedAt = now
    mem.UpdatedAt = now

    // Default type if not specified
    if mem.Type == "" {
        mem.Type = models.TypeGeneral
    }

    metadataJSON := metadataToJSON(mem.Metadata)
    tagsJSON := tagsToJSON(mem.Tags)

    _, err = ExecCypher(tx, r.client.GraphName(), 0,
        `CREATE (m:Memory {
            id: '%s',
            node_type: 'Memory',
            type: '%s',
            content: '%s',
            metadata: '%s',
            tags: %s,
            created_at: '%s',
            updated_at: '%s'
        })`,
        EscapeCypherString(mem.ID),
        EscapeCypherString(string(mem.Type)),
        EscapeCypherString(mem.Content),
        EscapeCypherString(metadataJSON),
        tagsJSON,
        mem.CreatedAt.Format(time.RFC3339),
        mem.UpdatedAt.Format(time.RFC3339),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create memory: %w", err)
    }

    // Create relationships
    for _, rel := range relationships {
        if err := r.createRelationship(tx, mem.ID, rel.ToID, rel.Type); err != nil {
            fmt.Fprintf(os.Stderr, "warning: failed to create relationship: %v\n", err)
        }
    }

    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("failed to commit: %w", err)
    }

    return &mem, nil
}
```

#### GetByID

**Neo4j**:
```cypher
MATCH (m:Memory {id: $id}) RETURN m
```

**AGE**:
```go
func (r *Repository) GetByID(ctx context.Context, id string) (*models.Memory, error) {
    tx, err := r.client.BeginTx(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    cursor, err := ExecCypher(tx, r.client.GraphName(), 1,
        "MATCH (m:Memory {id: '%s'}) RETURN m",
        EscapeCypherString(id))
    if err != nil {
        return nil, err
    }

    if !cursor.Next() {
        return nil, nil // Not found
    }

    row, err := cursor.GetRow()
    if err != nil {
        return nil, err
    }

    vertex := row[0].(*age.Vertex)
    mem := VertexToMemory(vertex)
    
    tx.Commit()
    return &mem, nil
}
```

#### Delete

**Neo4j**:
```cypher
MATCH (m:Memory {id: $id}) DETACH DELETE m
```

**AGE**:
```go
func (r *Repository) Delete(ctx context.Context, id string) error {
    tx, err := r.client.BeginTx(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    _, err = ExecCypher(tx, r.client.GraphName(), 0,
        "MATCH (m:Memory {id: '%s'}) DETACH DELETE m",
        EscapeCypherString(id))
    if err != nil {
        return fmt.Errorf("delete failed: %w", err)
    }

    return tx.Commit()
}
```

#### Create Relationship (Idempotent via Check-Then-Create)

**Neo4j**:
```cypher
MATCH (a) WHERE a.id = $from_id
MATCH (b) WHERE b.id = $to_id
MERGE (a)-[r:RELATES_TO]->(b)
```

**AGE**:

> **Important**: AGE does not support MERGE for relationships. We use a check-then-create
> pattern instead. This has a theoretical race condition under concurrent writes, but:
> - Duplicate edges are functionally harmless (queries use `DISTINCT` and Go-level dedup)
> - Concurrent writes to the same relationship pair are extremely unlikely in MCP usage
> - The simplicity trade-off is worthwhile vs. SERIALIZABLE isolation or UNIQUE constraints

```go
func (r *Repository) createRelationship(tx *sql.Tx, fromID, toID string, relType models.RelationType) error {
    // Check if relationship already exists to avoid duplicates (AGE doesn't support MERGE)
    cursor, err := ExecCypher(tx, r.client.GraphName(), 1,
        `MATCH (a)-[r:%s]->(b) 
         WHERE a.id = '%s' AND b.id = '%s' 
         RETURN r`,
        relType,
        EscapeCypherString(fromID),
        EscapeCypherString(toID),
    )
    if err != nil {
        return err
    }
    
    // If relationship exists, nothing to do
    if cursor.Next() {
        return nil
    }
    
    // Create the relationship
    _, err = ExecCypher(tx, r.client.GraphName(), 0,
        `MATCH (a), (b) 
         WHERE a.id = '%s' AND b.id = '%s' 
         CREATE (a)-[r:%s]->(b)`,
        EscapeCypherString(fromID),
        EscapeCypherString(toID),
        relType, // Relationship type is safe (enum)
    )
    return err
}
```

#### GetByIDWithRelated

Retrieves a memory with its direct relationships (both incoming and outgoing).

**Neo4j**:
```cypher
MATCH (m:Memory {id: $id})
OPTIONAL MATCH (m)-[r]->(outgoing:Memory)
OPTIONAL MATCH (incoming:Memory)-[r2]->(m)
RETURN m, 
       collect(DISTINCT {id: outgoing.id, type: outgoing.type, rel_type: type(r), direction: 'outgoing'}) as outgoing_rels,
       collect(DISTINCT {id: incoming.id, type: incoming.type, rel_type: type(r2), direction: 'incoming'}) as incoming_rels
```

**AGE**:
```go
func (r *Repository) GetByIDWithRelated(ctx context.Context, id string) (*models.Memory, []models.RelatedInfo, error) {
    tx, err := r.client.BeginTx(ctx)
    if err != nil {
        return nil, nil, err
    }
    defer tx.Rollback()

    // Get memory with outgoing relationships
    cursor, err := ExecCypher(tx, r.client.GraphName(), 3,
        `MATCH (m:Memory {id: '%s'})
         OPTIONAL MATCH (m)-[r]->(outgoing:Memory)
         OPTIONAL MATCH (incoming:Memory)-[r2]->(m)
         RETURN m, 
                collect(DISTINCT {id: outgoing.id, type: outgoing.type, rel_type: type(r), direction: 'outgoing'}) as outgoing_rels,
                collect(DISTINCT {id: incoming.id, type: incoming.type, rel_type: type(r2), direction: 'incoming'}) as incoming_rels`,
        EscapeCypherString(id))
    if err != nil {
        return nil, nil, fmt.Errorf("query failed: %w", err)
    }

    if !cursor.Next() {
        return nil, nil, nil // Not found
    }

    row, err := cursor.GetRow()
    if err != nil {
        return nil, nil, err
    }

    mem := VertexToMemory(row[0].(*age.Vertex))
    var related []models.RelatedInfo

    // Process outgoing relationships
    if rels, ok := row[1].([]interface{}); ok {
        for _, rel := range rels {
            if m, ok := rel.(map[string]interface{}); ok {
                if relID := getStringProp(m, "id"); relID != "" {
                    related = append(related, models.RelatedInfo{
                        ID:           relID,
                        Type:         models.MemoryType(getStringProp(m, "type")),
                        RelationType: getStringProp(m, "rel_type"),
                        Direction:    "outgoing",
                    })
                }
            }
        }
    }

    // Process incoming relationships
    if rels, ok := row[2].([]interface{}); ok {
        for _, rel := range rels {
            if m, ok := rel.(map[string]interface{}); ok {
                if relID := getStringProp(m, "id"); relID != "" {
                    related = append(related, models.RelatedInfo{
                        ID:           relID,
                        Type:         models.MemoryType(getStringProp(m, "type")),
                        RelationType: getStringProp(m, "rel_type"),
                        Direction:    "incoming",
                    })
                }
            }
        }
    }

    tx.Commit()
    return &mem, related, nil
}
```

#### GetRelated (Depth Traversal)

Retrieves nodes related to a given ID with optional filtering by relationship type, direction, and depth.

**Neo4j**:
```cypher
MATCH (a) WHERE a.id = $id AND (a:Memory OR a:Plan OR a:Task)
MATCH (a)-[r*1..$depth]-(b)
WHERE a <> b AND (b:Memory OR b:Plan OR b:Task)
RETURN b, type(r[-1]) as rel_type, ...
```

**AGE**:
```go
func (r *Repository) GetRelated(ctx context.Context, id string, relationType string, direction string, depth int) ([]models.RelatedMemoryResult, error) {
    tx, err := r.client.BeginTx(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    // Build relationship pattern based on direction and type
    var relPattern string
    if relationType != "" {
        switch direction {
        case "outgoing":
            relPattern = fmt.Sprintf("-[r:%s*1..%d]->", relationType, depth)
        case "incoming":
            relPattern = fmt.Sprintf("<-[r:%s*1..%d]-", relationType, depth)
        default: // "both"
            relPattern = fmt.Sprintf("-[r:%s*1..%d]-", relationType, depth)
        }
    } else {
        switch direction {
        case "outgoing":
            relPattern = fmt.Sprintf("-[r*1..%d]->", depth)
        case "incoming":
            relPattern = fmt.Sprintf("<-[r*1..%d]-", depth)
        default: // "both"
            relPattern = fmt.Sprintf("-[r*1..%d]-", depth)
        }
    }

    cypher := fmt.Sprintf(`
        MATCH (a) WHERE a.id = '%s' AND (a:Memory OR a:Plan OR a:Task)
        MATCH (a)%s(b)
        WHERE a <> b AND (b:Memory OR b:Plan OR b:Task)
        WITH DISTINCT b, r,
             CASE WHEN startNode(r[-1]) = a OR (size(r) > 1 AND startNode(r[-1]).id = '%s') THEN 'outgoing' ELSE 'incoming' END as direction,
             size(r) as depth,
             type(r[-1]) as rel_type
        RETURN b, rel_type, direction, depth, b.node_type as nodeType
        ORDER BY depth ASC`,
        EscapeCypherString(id), relPattern, EscapeCypherString(id))

    cursor, err := ExecCypher(tx, r.client.GraphName(), 5, cypher)
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }

    var results []models.RelatedMemoryResult
    seen := make(map[string]bool)

    for cursor.Next() {
        row, err := cursor.GetRow()
        if err != nil {
            return nil, err
        }

        vertex := row[0].(*age.Vertex)
        props := vertex.Props()
        nodeID := getStringProp(props, "id")

        // Skip duplicates
        if seen[nodeID] {
            continue
        }
        seen[nodeID] = true

        // Determine node type from stored node_type property
        // (does not rely on AGE labels() function)
        nodeType := "Memory"
        if nt, ok := row[4].(string); ok && nt != "" {
            nodeType = nt
        }

        mem := models.Memory{
            ID:   nodeID,
            Type: models.MemoryType(nodeType),
        }
        if nodeType == "Plan" {
            mem.Content = getStringProp(props, "name")
        } else {
            mem.Content = getStringProp(props, "content")
        }

        // Parse metadata
        if metaStr := getStringProp(props, "metadata"); metaStr != "" {
            var meta map[string]string
            if err := json.Unmarshal([]byte(metaStr), &meta); err == nil {
                mem.Metadata = meta
            }
        }

        // Parse tags
        if tagsRaw, ok := props["tags"]; ok {
            if tagsArr, ok := tagsRaw.([]interface{}); ok {
                for _, t := range tagsArr {
                    if s, ok := t.(string); ok {
                        mem.Tags = append(mem.Tags, s)
                    }
                }
            }
        }

        // Parse timestamps
        if createdStr := getStringProp(props, "created_at"); createdStr != "" {
            if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
                mem.CreatedAt = t
            }
        }
        if updatedStr := getStringProp(props, "updated_at"); updatedStr != "" {
            if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
                mem.UpdatedAt = t
            }
        }

        relTypeStr, _ := row[1].(string)
        dirStr, _ := row[2].(string)
        depthVal := 1
        if d, ok := row[3].(int64); ok {
            depthVal = int(d)
        }

        results = append(results, models.RelatedMemoryResult{
            Memory:       mem,
            RelationType: relTypeStr,
            Direction:    dirStr,
            Depth:        depthVal,
        })
    }

    tx.Commit()
    return results, nil
}
```

### Task 4.2: Migrate Plan Repository

**File**: `internal/graph/plan_repository.go`

Similar pattern to Memory repository. Key differences:

- Plan nodes have `name` and `description` instead of just `content`
- Plans have relationships to Tasks via `PART_OF`
- Cascade delete logic for orphan tasks
- **Must include `node_type: 'Plan'` property** on every created Plan node
- **Default status**: If `plan.Status` is empty, set to `models.PlanStatusActive`

#### Add Plan

```go
func (r *PlanRepository) Add(ctx context.Context, plan models.Plan, relationships []models.Relationship) (*models.Plan, error) {
    tx, err := r.client.BeginTx(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    if plan.ID == "" {
        plan.ID = uuid.New().String()
    }
    now := time.Now().UTC()
    plan.CreatedAt = now
    plan.UpdatedAt = now

    // Default status if not specified
    if plan.Status == "" {
        plan.Status = models.PlanStatusActive
    }

    metadataJSON := metadataToJSON(plan.Metadata)
    tagsJSON := tagsToJSON(plan.Tags)

    // Create plan vertex with node_type property for type detection
    _, err = ExecCypher(tx, r.client.GraphName(), 0,
        `CREATE (p:Plan {
            id: '%s',
            node_type: 'Plan',
            name: '%s',
            description: '%s',
            status: '%s',
            metadata: '%s',
            tags: %s,
            created_at: '%s',
            updated_at: '%s'
        })`,
        EscapeCypherString(plan.ID),
        EscapeCypherString(plan.Name),
        EscapeCypherString(plan.Description),
        EscapeCypherString(string(plan.Status)),
        EscapeCypherString(metadataJSON),
        tagsJSON,
        plan.CreatedAt.Format(time.RFC3339),
        plan.UpdatedAt.Format(time.RFC3339),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create plan: %w", err)
    }

    // Create relationships
    for _, rel := range relationships {
        if err := r.createRelationshipFromPlan(tx, plan.ID, rel.ToID, rel.Type); err != nil {
            fmt.Fprintf(os.Stderr, "warning: failed to create relationship: %v\n", err)
        }
    }

    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("failed to commit: %w", err)
    }

    return &plan, nil
}
```

#### GetWithTasks

Retrieves a plan with all tasks ordered by position, including dependency information
(DEPENDS_ON and BLOCKS relationships between tasks in the same plan).

```go
func (r *PlanRepository) GetWithTasks(ctx context.Context, planID string) (*models.Plan, []models.TaskInPlan, error) {
    tx, err := r.client.BeginTx(ctx)
    if err != nil {
        return nil, nil, err
    }
    defer tx.Rollback()

    // Get plan
    cursor, err := ExecCypher(tx, r.client.GraphName(), 1,
        "MATCH (p:Plan {id: '%s'}) RETURN p",
        EscapeCypherString(planID))
    if err != nil {
        return nil, nil, err
    }

    if !cursor.Next() {
        return nil, nil, nil
    }

    row, err := cursor.GetRow()
    if err != nil {
        return nil, nil, err
    }
    plan := VertexToPlan(row[0].(*age.Vertex))

    // Get tasks with positions and dependency information
    cursor, err = ExecCypher(tx, r.client.GraphName(), 4,
        `MATCH (p:Plan {id: '%s'})
         OPTIONAL MATCH (t:Task)-[r:PART_OF]->(p)
         OPTIONAL MATCH (t)-[:DEPENDS_ON]->(dep:Task)-[:PART_OF]->(p)
         OPTIONAL MATCH (t)-[:BLOCKS]->(blk:Task)-[:PART_OF]->(p)
         WITH t, r, collect(DISTINCT dep.id) as depends_on, collect(DISTINCT blk.id) as blocks
         WHERE t IS NOT NULL
         RETURN t, COALESCE(r.position, 0.0) as position, depends_on, blocks
         ORDER BY position ASC`,
        EscapeCypherString(planID))
    if err != nil {
        return &plan, nil, err
    }

    var tasks []models.TaskInPlan
    for cursor.Next() {
        row, err := cursor.GetRow()
        if err != nil {
            return &plan, nil, err
        }
        task := VertexToTask(row[0].(*age.Vertex))
        position := toFloat64(row[1])

        taskInPlan := models.TaskInPlan{
            Task:     task,
            Position: position,
        }

        // Extract depends_on IDs
        if deps, ok := row[2].([]interface{}); ok {
            for _, d := range deps {
                if s, ok := d.(string); ok && s != "" {
                    taskInPlan.DependsOn = append(taskInPlan.DependsOn, s)
                }
            }
        }

        // Extract blocks IDs
        if blks, ok := row[3].([]interface{}); ok {
            for _, b := range blks {
                if s, ok := b.(string); ok && s != "" {
                    taskInPlan.Blocks = append(taskInPlan.Blocks, s)
                }
            }
        }

        tasks = append(tasks, taskInPlan)
    }

    tx.Commit()
    return &plan, tasks, nil
}
```

### Task 4.3: Migrate Task Repository

**File**: `internal/graph/task_repository.go`

Key considerations:
- Tasks must belong to at least one Plan
- Position management for ordering within plans
- Dependencies (DEPENDS_ON, BLOCKS, FOLLOWS relationships)

#### Position Constants and Utilities

Port these position management utilities from the Neo4j implementation:

```go
// Position constants for task ordering
const (
    // DefaultPositionIncrement is the default spacing between positions
    DefaultPositionIncrement = 1000.0
)

// appendPosition returns a position value for appending after maxPos.
// Uses nanosecond timestamp plus small random jitter to ensure uniqueness
// in concurrent scenarios.
func appendPosition(maxPos float64) float64 {
    nanoComponent := float64(time.Now().UnixNano()%1e9) / 1e9
    jitter := rand.Float64() * 0.0001
    return maxPos + DefaultPositionIncrement + nanoComponent + jitter
}

// CalculateInsertPositions calculates position values for inserting tasks between
// afterPos and beforePos. Returns a slice of positions for the tasks to be inserted.
func CalculateInsertPositions(afterPos, beforePos float64, count int) []float64 {
    if count <= 0 {
        return nil
    }

    positions := make([]float64, count)

    switch {
    case afterPos == 0 && beforePos == 0:
        // Empty plan or no reference - start at increment
        for i := 0; i < count; i++ {
            positions[i] = DefaultPositionIncrement * float64(i+1)
        }
    case beforePos == 0:
        // Inserting after a task (at end)
        for i := 0; i < count; i++ {
            positions[i] = afterPos + DefaultPositionIncrement*float64(i+1)
        }
    case afterPos == 0:
        // Inserting before a task (at start)
        gap := beforePos / float64(count+1)
        for i := 0; i < count; i++ {
            positions[i] = gap * float64(i+1)
        }
    default:
        // Inserting between two tasks
        gap := (beforePos - afterPos) / float64(count+1)
        for i := 0; i < count; i++ {
            positions[i] = afterPos + gap*float64(i+1)
        }
    }

    return positions
}
```

#### Position Query Helpers

```go
// getMaxPosition returns the maximum position value for tasks in a plan.
func (r *TaskRepository) getMaxPosition(ctx context.Context, tx *sql.Tx, planID string) (float64, error) {
    cursor, err := ExecCypher(tx, r.client.GraphName(), 1,
        `MATCH (t:Task)-[r:PART_OF]->(p:Plan {id: '%s'})
         RETURN COALESCE(max(r.position), 0.0) as max_pos`,
        EscapeCypherString(planID))
    if err != nil {
        return 0, fmt.Errorf("failed to get max position: %w", err)
    }

    if !cursor.Next() {
        return 0, nil
    }

    row, _ := cursor.GetRow()
    return toFloat64(row[0]), nil
}

// getTaskPosition returns the position of a task within a specific plan.
func (r *TaskRepository) getTaskPosition(ctx context.Context, tx *sql.Tx, taskID, planID string) (float64, error) {
    cursor, err := ExecCypher(tx, r.client.GraphName(), 1,
        `MATCH (t:Task {id: '%s'})-[r:PART_OF]->(p:Plan {id: '%s'})
         RETURN COALESCE(r.position, 0.0) as position`,
        EscapeCypherString(taskID),
        EscapeCypherString(planID))
    if err != nil {
        return 0, fmt.Errorf("failed to get task position: %w", err)
    }

    if !cursor.Next() {
        return 0, nil
    }

    row, _ := cursor.GetRow()
    return toFloat64(row[0]), nil
}

// getAdjacentPositions returns the positions of tasks immediately before and after
// the task with the given ID in a plan.
func (r *TaskRepository) getAdjacentPositions(ctx context.Context, tx *sql.Tx, taskID, planID string) (before, after float64, err error) {
    currentPos, err := r.getTaskPosition(ctx, tx, taskID, planID)
    if err != nil {
        return 0, 0, err
    }

    // Get position before
    cursor, err := ExecCypher(tx, r.client.GraphName(), 1,
        `MATCH (t:Task)-[r:PART_OF]->(p:Plan {id: '%s'})
         WHERE r.position < %f
         RETURN r.position as position
         ORDER BY r.position DESC
         LIMIT 1`,
        EscapeCypherString(planID), currentPos)
    if err != nil {
        return 0, 0, err
    }
    if cursor.Next() {
        row, _ := cursor.GetRow()
        before = toFloat64(row[0])
    }

    // Get position after
    cursor, err = ExecCypher(tx, r.client.GraphName(), 1,
        `MATCH (t:Task)-[r:PART_OF]->(p:Plan {id: '%s'})
         WHERE r.position > %f
         RETURN r.position as position
         ORDER BY r.position ASC
         LIMIT 1`,
        EscapeCypherString(planID), currentPos)
    if err != nil {
        return 0, 0, err
    }
    if cursor.Next() {
        row, _ := cursor.GetRow()
        after = toFloat64(row[0])
    }

    return before, after, nil
}

// calculatePosition calculates the position for a new task based on afterTaskID/beforeTaskID
func (r *TaskRepository) calculatePosition(ctx context.Context, tx *sql.Tx, planID string, afterTaskID, beforeTaskID *string) (float64, error) {
    var afterPos, beforePos float64
    var err error

    if afterTaskID != nil && *afterTaskID != "" {
        afterPos, err = r.getTaskPosition(ctx, tx, *afterTaskID, planID)
        if err != nil {
            return 0, err
        }
        if beforeTaskID == nil || *beforeTaskID == "" {
            _, afterPos2, err := r.getAdjacentPositions(ctx, tx, *afterTaskID, planID)
            if err != nil {
                return 0, err
            }
            beforePos = afterPos2
        }
    }

    if beforeTaskID != nil && *beforeTaskID != "" {
        beforePos, err = r.getTaskPosition(ctx, tx, *beforeTaskID, planID)
        if err != nil {
            return 0, err
        }
        if afterTaskID == nil || *afterTaskID == "" {
            beforePos2, _, err := r.getAdjacentPositions(ctx, tx, *beforeTaskID, planID)
            if err != nil {
                return 0, err
            }
            afterPos = beforePos2
        }
    }

    // If neither specified, append to end
    if (afterTaskID == nil || *afterTaskID == "") && (beforeTaskID == nil || *beforeTaskID == "") {
        maxPos, err := r.getMaxPosition(ctx, tx, planID)
        if err != nil {
            return 0, err
        }
        return appendPosition(maxPos), nil
    }

    positions := CalculateInsertPositions(afterPos, beforePos, 1)
    if len(positions) == 0 {
        return DefaultPositionIncrement, nil
    }
    return positions[0], nil
}

func toFloat64(v interface{}) float64 {
    switch val := v.(type) {
    case float64:
        return val
    case float32:
        return float64(val)
    case int64:
        return float64(val)
    case int:
        return float64(val)
    default:
        return 0
    }
}
```

#### Add Task with Position

```go
func (r *TaskRepository) Add(ctx context.Context, task models.Task, planIDs []string, relationships []models.Relationship, afterTaskID, beforeTaskID *string) (*models.Task, error) {
    if len(planIDs) == 0 {
        return nil, fmt.Errorf("task must belong to at least one plan")
    }

    tx, err := r.client.BeginTx(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    // Verify all plans exist
    for _, planID := range planIDs {
        cursor, err := ExecCypher(tx, r.client.GraphName(), 1,
            "MATCH (p:Plan {id: '%s'}) RETURN p",
            EscapeCypherString(planID))
        if err != nil {
            return nil, err
        }
        if !cursor.Next() {
            return nil, fmt.Errorf("plan not found: %s", planID)
        }
    }

    // Generate ID and timestamps
    if task.ID == "" {
        task.ID = uuid.New().String()
    }
    now := time.Now().UTC()
    task.CreatedAt = now
    task.UpdatedAt = now

    // Default status if not specified
    if task.Status == "" {
        task.Status = models.TaskStatusPending
    }

    // Create task vertex with node_type property for type detection
    _, err = ExecCypher(tx, r.client.GraphName(), 0,
        `CREATE (t:Task {
            id: '%s',
            node_type: 'Task',
            content: '%s',
            status: '%s',
            metadata: '%s',
            tags: %s,
            created_at: '%s',
            updated_at: '%s'
        })`,
        EscapeCypherString(task.ID),
        EscapeCypherString(task.Content),
        EscapeCypherString(string(task.Status)),
        EscapeCypherString(metadataToJSON(task.Metadata)),
        tagsToJSON(task.Tags),
        task.CreatedAt.Format(time.RFC3339),
        task.UpdatedAt.Format(time.RFC3339),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create task: %w", err)
    }

    // Create PART_OF relationships to plans with positions
    for _, planID := range planIDs {
        position, err := r.calculatePosition(ctx, tx, planID, afterTaskID, beforeTaskID)
        if err != nil {
            return nil, err
        }

        _, err = ExecCypher(tx, r.client.GraphName(), 0,
            `MATCH (t:Task {id: '%s'}), (p:Plan {id: '%s'})
             CREATE (t)-[r:PART_OF {position: %f}]->(p)`,
            EscapeCypherString(task.ID),
            EscapeCypherString(planID),
            position,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to link task to plan: %w", err)
        }
    }

    // Create other relationships (use idempotent pattern)
    for _, rel := range relationships {
        if err := r.createRelationshipFromTask(tx, task.ID, rel.ToID, rel.Type); err != nil {
            fmt.Fprintf(os.Stderr, "warning: failed to create relationship: %v\n", err)
        }
    }

    if err := tx.Commit(); err != nil {
        return nil, err
    }

    return &task, nil
}

// createRelationshipFromTask creates a relationship from a Task to another node (idempotent)
func (r *TaskRepository) createRelationshipFromTask(tx *sql.Tx, fromID, toID string, relType models.RelationType) error {
    // Check if relationship already exists
    cursor, err := ExecCypher(tx, r.client.GraphName(), 1,
        `MATCH (a:Task {id: '%s'})-[r:%s]->(b {id: '%s'})
         RETURN r`,
        EscapeCypherString(fromID),
        relType,
        EscapeCypherString(toID),
    )
    if err != nil {
        return err
    }
    if cursor.Next() {
        return nil // Already exists
    }

    _, err = ExecCypher(tx, r.client.GraphName(), 0,
        `MATCH (a:Task {id: '%s'}), (b) WHERE b.id = '%s' AND (b:Memory OR b:Plan OR b:Task)
         CREATE (a)-[r:%s]->(b)`,
        EscapeCypherString(fromID),
        EscapeCypherString(toID),
        relType,
    )
    return err
}
```

#### List Tasks

```go
// List retrieves tasks with optional filtering.
// When planID is provided, tasks are ordered by position.
// When planID is not provided, tasks are ordered by updated_at DESC.
func (r *TaskRepository) List(ctx context.Context, planID string, status string, tags []string, limit int) ([]models.TaskListResult, error) {
    tx, err := r.client.BeginTx(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    if limit <= 0 {
        limit = 50
    }

    var cypher string
    hasPosition := false

    if planID != "" {
        hasPosition = true
        whereClauses := []string{}
        if status != "" {
            whereClauses = append(whereClauses, fmt.Sprintf("t.status = '%s'", EscapeCypherString(status)))
        }
        if len(tags) > 0 {
            // Build tag filter
            tagConditions := make([]string, len(tags))
            for i, tag := range tags {
                tagConditions[i] = fmt.Sprintf("'%s' IN t.tags", EscapeCypherString(tag))
            }
            whereClauses = append(whereClauses, "("+strings.Join(tagConditions, " OR ")+")")
        }

        whereClause := ""
        if len(whereClauses) > 0 {
            whereClause = "AND " + strings.Join(whereClauses, " AND ")
        }

        cypher = fmt.Sprintf(`
            MATCH (t:Task)-[r:PART_OF]->(p:Plan {id: '%s'})
            WHERE true %s
            RETURN t, r.position as position
            ORDER BY r.position ASC
            LIMIT %d`,
            EscapeCypherString(planID), whereClause, limit)
    } else {
        whereClauses := []string{}
        if status != "" {
            whereClauses = append(whereClauses, fmt.Sprintf("t.status = '%s'", EscapeCypherString(status)))
        }
        if len(tags) > 0 {
            tagConditions := make([]string, len(tags))
            for i, tag := range tags {
                tagConditions[i] = fmt.Sprintf("'%s' IN t.tags", EscapeCypherString(tag))
            }
            whereClauses = append(whereClauses, "("+strings.Join(tagConditions, " OR ")+")")
        }

        whereClause := ""
        if len(whereClauses) > 0 {
            whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
        }

        cypher = fmt.Sprintf(`
            MATCH (t:Task)
            %s
            RETURN t, null as position
            ORDER BY t.updated_at DESC
            LIMIT %d`,
            whereClause, limit)
    }

    cursor, err := ExecCypher(tx, r.client.GraphName(), 2, cypher)
    if err != nil {
        return nil, fmt.Errorf("list failed: %w", err)
    }

    var tasks []models.TaskListResult
    for cursor.Next() {
        row, err := cursor.GetRow()
        if err != nil {
            return nil, err
        }
        task := VertexToTask(row[0].(*age.Vertex))
        taskResult := models.TaskListResult{Task: task}
        
        if hasPosition && row[1] != nil {
            pos := toFloat64(row[1])
            taskResult.Position = &pos
        }
        tasks = append(tasks, taskResult)
    }

    tx.Commit()
    return tasks, nil
}
```

#### GetByID, GetWithPlans, Update, Delete

```go
func (r *TaskRepository) GetByID(ctx context.Context, id string) (*models.Task, error) {
    tx, err := r.client.BeginTx(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    cursor, err := ExecCypher(tx, r.client.GraphName(), 1,
        "MATCH (t:Task {id: '%s'}) RETURN t",
        EscapeCypherString(id))
    if err != nil {
        return nil, err
    }

    if !cursor.Next() {
        return nil, nil
    }

    row, _ := cursor.GetRow()
    task := VertexToTask(row[0].(*age.Vertex))
    tx.Commit()
    return &task, nil
}

func (r *TaskRepository) GetWithPlans(ctx context.Context, id string) (*models.Task, []models.Plan, error) {
    tx, err := r.client.BeginTx(ctx)
    if err != nil {
        return nil, nil, err
    }
    defer tx.Rollback()

    cursor, err := ExecCypher(tx, r.client.GraphName(), 2,
        `MATCH (t:Task {id: '%s'})
         OPTIONAL MATCH (t)-[:PART_OF]->(p:Plan)
         RETURN t, collect(p) as plans`,
        EscapeCypherString(id))
    if err != nil {
        return nil, nil, err
    }

    if !cursor.Next() {
        return nil, nil, nil
    }

    row, _ := cursor.GetRow()
    task := VertexToTask(row[0].(*age.Vertex))

    var plans []models.Plan
    if planNodes, ok := row[1].([]interface{}); ok {
        for _, pn := range planNodes {
            if pn != nil {
                if vertex, ok := pn.(*age.Vertex); ok {
                    plans = append(plans, VertexToPlan(vertex))
                }
            }
        }
    }

    tx.Commit()
    return &task, plans, nil
}

func (r *TaskRepository) Delete(ctx context.Context, id string) error {
    tx, err := r.client.BeginTx(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    _, err = ExecCypher(tx, r.client.GraphName(), 0,
        "MATCH (t:Task {id: '%s'}) DETACH DELETE t",
        EscapeCypherString(id))
    if err != nil {
        return fmt.Errorf("delete failed: %w", err)
    }

    return tx.Commit()
}
```

#### UpdatePositions (Batch Reorder)

```go
// UpdatePositions batch updates task positions within a plan.
// Used by the reorder operation to move multiple tasks at once.
func (r *TaskRepository) UpdatePositions(ctx context.Context, planID string, taskPositions map[string]float64) error {
    tx, err := r.client.BeginTx(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    for taskID, position := range taskPositions {
        _, err := ExecCypher(tx, r.client.GraphName(), 0,
            `MATCH (t:Task {id: '%s'})-[r:PART_OF]->(p:Plan {id: '%s'})
             SET r.position = %f`,
            EscapeCypherString(taskID),
            EscapeCypherString(planID),
            position,
        )
        if err != nil {
            return fmt.Errorf("failed to update position for task %s: %w", taskID, err)
        }
    }

    return tx.Commit()
}
```

---

## Phase 5: Update Tests

### Task 5.1: Update Integration Test Helper

**File**: `internal/graph/plan_task_integration_test.go`

```go
func getTestClient(t *testing.T) (*Client, context.Context, context.CancelFunc) {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

    cfg := Config{
        Host:     getEnvOrDefault("DB_HOST", "localhost"),
        Port:     getEnvOrDefault("DB_PORT", "5432"),
        Username: getEnvOrDefault("DB_USERNAME", "associate"),
        Password: getEnvOrDefault("DB_PASSWORD", "password"),
        Database: getEnvOrDefault("DB_DATABASE", "associate"),
    }

    client, err := NewClient(ctx, cfg)
    if err != nil {
        cancel()
        t.Fatalf("Failed to connect to PostgreSQL/AGE: %v", err)
    }

    return client, ctx, cancel
}

func cleanupTestData(ctx context.Context, client *Client, ids ...string) {
    tx, err := client.BeginTx(ctx)
    if err != nil {
        return
    }
    defer tx.Rollback()

    for _, id := range ids {
        ExecCypher(tx, client.GraphName(), 0,
            "MATCH (n {id: '%s'}) DETACH DELETE n",
            EscapeCypherString(id))
    }
    tx.Commit()
}
```

### Task 5.2: Update Unit Tests

Update test files to use new package name and connection method:
- `internal/graph/client_test.go`
- `internal/graph/persistence_test.go`
- `internal/mcp/plan_task_test.go`
- `internal/mcp/server_test.go`

---

## Phase 6: Update Application Entry Point

### Task 6.1: Update Main

**File**: `cmd/associate/main.go`

```go
import (
    "github.com/fitz/associate/internal/graph"
    // ... other imports
)

func main() {
    // ...
    
    cfg := graph.ConfigFromEnv()
    
    client, err := graph.NewClientWithRetry(ctx, cfg, nil)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer client.Close(ctx)
    
    // ...
}
```

---

## Phase 7: Update Documentation

### Task 7.1: Update README.md

- Replace Neo4j references with PostgreSQL/AGE
- Update Docker instructions
- Add AGE Viewer access instructions (http://localhost:3000)
- Update environment variable documentation
- Remove Neo4j Browser references

### Task 7.2: Update docs/memory.md

- Update query examples to show AGE format
- Note that Cypher syntax is largely the same

---

## Phase 8: Cleanup

### Task 8.1: Remove Neo4j Artifacts

- Delete any Neo4j-specific configuration files
- Remove old volume definitions
- Update .gitignore if needed

### Task 8.2: Final Testing

1. Run all unit tests: `go test ./...`
2. Run integration tests: `go test -tags=integration ./internal/graph/...`
3. Manual testing with AGE Viewer:
   - Connect to http://localhost:3000
   - Connection URL: `localhost`
   - Port: `5432`
   - Database: `associate`
   - Username: `associate`
   - Password: `password`
   - Verify graph visualization works

---

## Task Execution Order

| # | Task | Depends On | Est. Effort |
|---|------|------------|-------------|
| 1 | Update go.mod dependencies | - | Small |
| 2 | Rename neo4j package to graph | 1 | Small |
| 3 | Update all import paths | 2 | Small |
| 4 | Create new client.go (with initSchema + pg_trgm indexes) | 2 | Medium |
| 4a | **AGE Cypher compatibility validation tests** | 4 | Medium |
| 4b | **Document alternatives for any failing Cypher features** | 4a | Small |
| 5 | Create helpers.go | 4 | Medium |
| 6 | Migrate repository.go (Memory): Add, GetByID, Delete, Update (with node_type property) | 5, 4b | Medium |
| 6a | Migrate repository.go: Search with pg_trgm + ID search | 6 | Medium |
| 6b | Migrate repository.go: GetByIDWithRelated | 6 | Medium |
| 6c | Migrate repository.go: GetRelated (depth traversal, full direction logic, node_type) | 6 | Large |
| 6d | Migrate repository.go: createRelationship (check-then-create) | 6 | Small |
| 7 | Update repository tests | 6c | Medium |
| 8 | Migrate plan_repository.go (with node_type, default status, DependsOn/Blocks in GetWithTasks) | 5, 4b | Large |
| 9 | Update plan repository tests | 8 | Medium |
| 10 | Migrate task_repository.go: Add (with node_type, default status), GetByID, GetWithPlans, Delete | 5, 4b | Medium |
| 10a | Migrate task_repository.go: Position utilities (appendPosition, CalculateInsertPositions, etc.) | 10 | Medium |
| 10b | Migrate task_repository.go: List with position ordering | 10a | Medium |
| 10c | Migrate task_repository.go: UpdatePositions (batch reorder) | 10a | Medium |
| 10d | Migrate task_repository.go: createRelationshipFromTask (check-then-create) | 10 | Small |
| 11 | Update task repository tests | 10c | Medium |
| 12 | Update integration tests | 7, 9, 11 | Medium |
| 13 | Update main.go | 4 | Small |
| 14 | Update docker-compose.yml | - | Small |
| 15 | Update Dockerfile | - | Small |
| 16 | Update README.md | 14, 15 | Small |
| 17 | Update docs/memory.md | 6 | Small |
| 18 | Full integration testing | All | Medium |
| 19 | Cleanup old Neo4j artifacts | 18 | Small |

---

## Rollback Plan

Since this is an early development project with no data migration requirement, rollback is straightforward:

1. `git checkout` to previous commit
2. `docker-compose down -v` to remove AGE volumes
3. `docker-compose up` with old Neo4j configuration

---

## AGE Viewer Usage

After migration, access the graph visualization at http://localhost:3000

**Connection Settings**:
- Connect URL: `postgres` (or `localhost` if accessing from host)
- Connect Port: `5432`
- Database Name: `associate`
- User Name: `associate`
- Password: `password`
- Graph Name: `associate`

**Example Queries**:
```cypher
-- View all memories
MATCH (m:Memory) RETURN m

-- View all plans with their tasks
MATCH (t:Task)-[:PART_OF]->(p:Plan) RETURN t, p

-- View relationship graph
MATCH (a)-[r]->(b) RETURN a, r, b LIMIT 100
```

---

## Notes and Considerations

1. **AGE Driver Fallback**: If the official AGE Go driver proves problematic, fall back to using `lib/pq` directly with manual agtype parsing using the `database/sql` interface.

2. **Full-Text Search Upgrade Path**: If CONTAINS becomes a performance bottleneck, add `pg_trgm` extension:
   ```sql
   CREATE EXTENSION pg_trgm;
   -- Create trigram index on a supporting table
   ```

3. **Transaction Handling**: AGE requires explicit transaction management. Every Cypher query must be wrapped in a transaction.

4. **Parameter Passing**: The AGE Go driver uses string formatting for parameters (not parameterized queries like Neo4j). Always use `EscapeCypherString()` to prevent injection.

5. **Label Tables**: AGE creates PostgreSQL tables for each label (e.g., `associate."Memory"`). These can be queried directly with SQL if needed.

6. **Idempotent Relationship Creation**: AGE does not support `MERGE` for relationships. Use an existence check before `CREATE` to avoid duplicate relationships. This is critical for maintaining data integrity.

7. **Position Management**: Task ordering uses fractional positioning with these key functions:
   - `appendPosition`: Appends to end with timestamp-based uniqueness for concurrent inserts
   - `CalculateInsertPositions`: Calculates positions for inserting between existing tasks
   - `UpdatePositions`: Batch updates for reordering operations
   
   These must be ported accurately to preserve task ordering behavior.

8. **Search API Shape**: The Search function must return `related_ids` to preserve API compatibility. Even though scoring is lost (returns 1.0), the related memories collection must be maintained.
