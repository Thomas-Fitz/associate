# Migration Plan: Neo4j to Apache AGE

This document outlines the complete migration of the Associate MCP server from Neo4j graph database to Apache AGE (PostgreSQL extension).

## Overview

| Aspect | Current (Neo4j) | Target (Apache AGE) |
|--------|-----------------|---------------------|
| Database | Neo4j 5 Community | PostgreSQL 17 + AGE 1.6.0 |
| Protocol | Bolt (7687) | PostgreSQL (5432) |
| Driver | `neo4j/neo4j-go-driver/v5` | `apache/age/drivers/golang/age` |
| Query Format | Direct Cypher | Cypher wrapped in `cypher()` SQL function |
| Full-Text Search | Native FULLTEXT INDEX | CONTAINS (simple string matching) |
| Visualization | Neo4j Browser (:7474) | Apache AGE Viewer (:3000) |
| Graph Name | N/A (database-level) | `associate` |

## Decisions

- **PostgreSQL version**: 17 with AGE 1.6.0
- **Full-text search**: CONTAINS (upgrade to pg_trgm later if needed)
- **Go driver**: Official AGE driver (fallback to lib/pq if problematic)
- **Migration approach**: Full cutover (drop Neo4j entirely)
- **Package naming**: Generic (`internal/graph/`)
- **Environment variables**: Generic (`DB_HOST`, `DB_PORT`, `DB_USERNAME`, `DB_PASSWORD`, `DB_DATABASE`)
- **Graph name**: `associate`
- **Visualization**: Apache AGE Viewer included in docker-compose

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

// initSchema creates indexes for the graph
// Note: AGE stores vertices in label-specific tables, indexes are PostgreSQL indexes
func (c *Client) initSchema(ctx context.Context) error {
    // AGE automatically creates tables for labels when vertices are created
    // We can add PostgreSQL indexes on the properties JSONB column later if needed
    // For now, the basic graph structure is sufficient
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

## Phase 4: Repository Layer Migration

### Task 4.1: Migrate Memory Repository

**File**: `internal/graph/repository.go`

Key query translations:

#### Search (CONTAINS fallback)

**Neo4j**:
```cypher
CALL db.index.fulltext.queryNodes('memory_content', $query) 
YIELD node, score
```

**AGE**:
```go
func (r *Repository) Search(ctx context.Context, query string, limit int) ([]models.SearchResult, error) {
    tx, err := r.client.BeginTx(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    escapedQuery := EscapeCypherString(query)
    
    cursor, err := ExecCypher(tx, r.client.GraphName(), 1,
        "MATCH (m:Memory) WHERE toLower(m.content) CONTAINS toLower('%s') RETURN m LIMIT %d",
        escapedQuery, limit)
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
        results = append(results, models.SearchResult{
            Memory: mem,
            Score:  1.0, // CONTAINS doesn't provide scoring
        })
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

    if mem.Type == "" {
        mem.Type = models.TypeGeneral
    }

    metadataJSON := metadataToJSON(mem.Metadata)
    tagsJSON := tagsToJSON(mem.Tags)

    _, err = ExecCypher(tx, r.client.GraphName(), 0,
        `CREATE (m:Memory {
            id: '%s',
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

#### Create Relationship

**Neo4j**:
```cypher
MATCH (a) WHERE a.id = $from_id
MATCH (b) WHERE b.id = $to_id
MERGE (a)-[r:RELATES_TO]->(b)
```

**AGE**:
```go
func (r *Repository) createRelationship(tx *sql.Tx, fromID, toID string, relType models.RelationType) error {
    _, err := ExecCypher(tx, r.client.GraphName(), 0,
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

### Task 4.2: Migrate Plan Repository

**File**: `internal/graph/plan_repository.go`

Similar pattern to Memory repository. Key differences:

- Plan nodes have `name` and `description` instead of just `content`
- Plans have relationships to Tasks via `PART_OF`
- Cascade delete logic for orphan tasks

#### GetWithTasks

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

    // Get tasks with positions
    cursor, err = ExecCypher(tx, r.client.GraphName(), 2,
        `MATCH (t:Task)-[r:PART_OF]->(p:Plan {id: '%s'})
         RETURN t, r.position
         ORDER BY r.position ASC`,
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
        position := getFloat64Prop(row[1].(map[string]interface{}), "position")
        
        tasks = append(tasks, models.TaskInPlan{
            Task:     task,
            Position: position,
        })
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

    // Create task vertex
    _, err = ExecCypher(tx, r.client.GraphName(), 0,
        `CREATE (t:Task {
            id: '%s',
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
        position, err := r.calculatePosition(tx, planID, afterTaskID, beforeTaskID)
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

    // Create other relationships
    for _, rel := range relationships {
        _, err = ExecCypher(tx, r.client.GraphName(), 0,
            `MATCH (a:Task {id: '%s'}), (b {id: '%s'})
             CREATE (a)-[r:%s]->(b)`,
            EscapeCypherString(task.ID),
            EscapeCypherString(rel.ToID),
            rel.Type,
        )
        if err != nil {
            fmt.Fprintf(os.Stderr, "warning: failed to create relationship: %v\n", err)
        }
    }

    if err := tx.Commit(); err != nil {
        return nil, err
    }

    return &task, nil
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
| 4 | Create new client.go | 2 | Medium |
| 5 | Create helpers.go | 4 | Medium |
| 6 | Migrate repository.go (Memory) | 5 | Large |
| 7 | Update repository tests | 6 | Medium |
| 8 | Migrate plan_repository.go | 5 | Large |
| 9 | Update plan repository tests | 8 | Medium |
| 10 | Migrate task_repository.go | 5 | Large |
| 11 | Update task repository tests | 10 | Medium |
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
