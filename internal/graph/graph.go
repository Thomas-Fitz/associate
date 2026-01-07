// Package graph provides Neo4j graph database operations for code memory management.
package graph

import (
"context"
"fmt"
"strings"

"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Client wraps a Neo4j driver for graph operations.
type Client struct {
driver   neo4j.DriverWithContext
database string
}

// NewClient creates a new Neo4j client.
func NewClient(uri, username, password, database string) (*Client, error) {
driver, err := neo4j.NewDriverWithContext(
uri,
neo4j.BasicAuth(username, password, ""),
)
if err != nil {
return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
}

return &Client{
driver:   driver,
database: database,
}, nil
}

// Close closes the Neo4j driver connection.
func (c *Client) Close(ctx context.Context) error {
return c.driver.Close(ctx)
}

// VerifyConnectivity checks if the connection to Neo4j is working.
func (c *Client) VerifyConnectivity(ctx context.Context) error {
return c.driver.VerifyConnectivity(ctx)
}

// RepoNode represents a repository node in the graph.
type RepoNode struct {
Path        string            `json:"path"`
Name        string            `json:"name"`
Description string            `json:"description,omitempty"`
Language    string            `json:"language,omitempty"`
Metadata    map[string]string `json:"metadata,omitempty"`
}

// Validate checks that all required fields are set.
func (r *RepoNode) Validate() error {
var missing []string

if r.Path == "" {
missing = append(missing, "Path")
}
if r.Name == "" {
missing = append(missing, "Name")
}

if len(missing) > 0 {
return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
}

return nil
}

// CodeNode represents a code structure (function, class, module, etc.) in the graph.
type CodeNode struct {
Type        string            `json:"type"`        // function, class, struct, interface, etc.
Name        string            `json:"name"`        // Name of the code element
FilePath    string            `json:"file_path"`   // Relative path from repo root
Description string            `json:"description"` // What this code does
Signature   string            `json:"signature,omitempty"`
LineStart   int               `json:"line_start,omitempty"`
LineEnd     int               `json:"line_end,omitempty"`
Metadata    map[string]string `json:"metadata,omitempty"`
}

// Validate checks that all required fields are set.
func (c *CodeNode) Validate() error {
var missing []string

if c.Type == "" {
missing = append(missing, "Type")
}
if c.Name == "" {
missing = append(missing, "Name")
}
if c.FilePath == "" {
missing = append(missing, "FilePath")
}

if len(missing) > 0 {
return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
}

return nil
}

// CreateRepo creates or updates a repository node in the graph.
func (c *Client) CreateRepo(ctx context.Context, repo *RepoNode) error {
if err := repo.Validate(); err != nil {
return fmt.Errorf("invalid repo node: %w", err)
}

session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
defer session.Close(ctx)

query := `
MERGE (r:Repo {path: $path})
SET r.name = $name,
    r.description = $description,
    r.language = $language,
    r.updated_at = datetime()
RETURN r
`

params := map[string]interface{}{
"path":        repo.Path,
"name":        repo.Name,
"description": repo.Description,
"language":    repo.Language,
}

_, err := session.Run(ctx, query, params)
if err != nil {
return fmt.Errorf("failed to create repo node: %w", err)
}

return nil
}

// GetRepo retrieves a repository node by path.
func (c *Client) GetRepo(ctx context.Context, path string) (*RepoNode, error) {
session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
defer session.Close(ctx)

query := `
MATCH (r:Repo {path: $path})
RETURN r.path AS path, r.name AS name, r.description AS description, r.language AS language
`

result, err := session.Run(ctx, query, map[string]interface{}{"path": path})
if err != nil {
return nil, fmt.Errorf("failed to query repo: %w", err)
}

if result.Next(ctx) {
record := result.Record()
repo := &RepoNode{
Path:        record.Values[0].(string),
Name:        record.Values[1].(string),
Description: getString(record.Values[2]),
Language:    getString(record.Values[3]),
}
return repo, nil
}

return nil, fmt.Errorf("repo not found: %s", path)
}

// DeleteRepo deletes a repository and all its associated nodes.
func (c *Client) DeleteRepo(ctx context.Context, repoPath string) error {
session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
defer session.Close(ctx)

query := `
MATCH (r:Repo {path: $path})
OPTIONAL MATCH (r)-[*]-(n)
DETACH DELETE r, n
`

_, err := session.Run(ctx, query, map[string]interface{}{"path": repoPath})
if err != nil {
return fmt.Errorf("failed to delete repo: %w", err)
}

return nil
}

// CreateCodeNode creates a code node and links it to a repository.
func (c *Client) CreateCodeNode(ctx context.Context, repoPath string, node *CodeNode) error {
if err := node.Validate(); err != nil {
return fmt.Errorf("invalid code node: %w", err)
}

session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
defer session.Close(ctx)

query := `
MATCH (r:Repo {path: $repoPath})
MERGE (c:Code {repo_path: $repoPath, file_path: $filePath, name: $name})
SET c.type = $type,
    c.description = $description,
    c.signature = $signature,
    c.line_start = $lineStart,
    c.line_end = $lineEnd,
    c.updated_at = datetime()
MERGE (r)-[:CONTAINS]->(c)
RETURN c
`

params := map[string]interface{}{
"repoPath":    repoPath,
"filePath":    node.FilePath,
"name":        node.Name,
"type":        node.Type,
"description": node.Description,
"signature":   node.Signature,
"lineStart":   node.LineStart,
"lineEnd":     node.LineEnd,
}

_, err := session.Run(ctx, query, params)
if err != nil {
return fmt.Errorf("failed to create code node: %w", err)
}

return nil
}

// QueryCodeNodes retrieves code nodes for a specific repository.
func (c *Client) QueryCodeNodes(ctx context.Context, repoPath string, nodeType string) ([]*CodeNode, error) {
session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
defer session.Close(ctx)

var query string
var params map[string]interface{}

if nodeType != "" {
query = `
MATCH (r:Repo {path: $repoPath})-[:CONTAINS]->(c:Code {type: $type})
RETURN c.type, c.name, c.file_path, c.description, c.signature, c.line_start, c.line_end
`
params = map[string]interface{}{
"repoPath": repoPath,
"type":     nodeType,
}
} else {
query = `
MATCH (r:Repo {path: $repoPath})-[:CONTAINS]->(c:Code)
RETURN c.type, c.name, c.file_path, c.description, c.signature, c.line_start, c.line_end
`
params = map[string]interface{}{
"repoPath": repoPath,
}
}

result, err := session.Run(ctx, query, params)
if err != nil {
return nil, fmt.Errorf("failed to query code nodes: %w", err)
}

var nodes []*CodeNode
for result.Next(ctx) {
record := result.Record()
node := &CodeNode{
Type:        record.Values[0].(string),
Name:        record.Values[1].(string),
FilePath:    record.Values[2].(string),
Description: getString(record.Values[3]),
Signature:   getString(record.Values[4]),
LineStart:   getInt(record.Values[5]),
LineEnd:     getInt(record.Values[6]),
}
nodes = append(nodes, node)
}

return nodes, nil
}

// getString safely converts an interface{} to string.
func getString(v interface{}) string {
if v == nil {
return ""
}
if s, ok := v.(string); ok {
return s
}
return ""
}

// getInt safely converts an interface{} to int.
func getInt(v interface{}) int {
if v == nil {
return 0
}
if i, ok := v.(int64); ok {
return int(i)
}
if i, ok := v.(int); ok {
return i
}
return 0
}

// MemoryNode represents a contextual memory or note stored by an AI agent.
type MemoryNode struct {
Content     string            `json:"content"`      // The actual memory content
ContextType string            `json:"context_type"` // Type: architectural_decision, bug_fix, performance, etc.
Tags        []string          `json:"tags"`         // Tags for categorization
RelatedTo   string            `json:"related_to,omitempty"` // Optional: related code path
Metadata    map[string]string `json:"metadata,omitempty"`
}

// Validate checks that all required fields are set.
func (m *MemoryNode) Validate() error {
var missing []string

if m.Content == "" {
missing = append(missing, "Content")
}
if m.ContextType == "" {
missing = append(missing, "ContextType")
}

if len(missing) > 0 {
return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
}

return nil
}

// LearningNode represents an architectural pattern or learning specific to a repository.
type LearningNode struct {
Pattern     string            `json:"pattern"`     // The pattern or learning
Category    string            `json:"category"`    // Category: architectural_pattern, best_practice, anti_pattern, etc.
Description string            `json:"description"` // Detailed description
Examples    []string          `json:"examples,omitempty"` // Code examples
Metadata    map[string]string `json:"metadata,omitempty"`
}

// Validate checks that all required fields are set.
func (l *LearningNode) Validate() error {
var missing []string

if l.Pattern == "" {
missing = append(missing, "Pattern")
}
if l.Category == "" {
missing = append(missing, "Category")
}

if len(missing) > 0 {
return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
}

return nil
}

// SaveMemory creates or updates a memory node and links it to a repository.
func (c *Client) SaveMemory(ctx context.Context, repoPath string, memory *MemoryNode) error {
if err := memory.Validate(); err != nil {
return fmt.Errorf("invalid memory node: %w", err)
}

session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
defer session.Close(ctx)

// Create unique ID based on content hash (simple approach)
memoryID := fmt.Sprintf("%x", memory.Content[:min(50, len(memory.Content))])

query := `
MATCH (r:Repo {path: $repoPath})
MERGE (m:Memory {id: $memoryID, repo_path: $repoPath})
SET m.content = $content,
    m.context_type = $contextType,
    m.tags = $tags,
    m.related_to = $relatedTo,
    m.updated_at = datetime()
MERGE (r)-[:HAS_MEMORY]->(m)
RETURN m
`

params := map[string]interface{}{
"repoPath":    repoPath,
"memoryID":    memoryID,
"content":     memory.Content,
"contextType": memory.ContextType,
"tags":        memory.Tags,
"relatedTo":   memory.RelatedTo,
}

_, err := session.Run(ctx, query, params)
if err != nil {
return fmt.Errorf("failed to save memory: %w", err)
}

return nil
}

// SearchMemory searches for memories in a repository matching the query.
func (c *Client) SearchMemory(ctx context.Context, repoPath string, query string, contextType string, tags []string, limit int) ([]*MemoryNode, error) {
session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
defer session.Close(ctx)

if limit == 0 {
limit = 10
}

// Build query based on filters
cypher := `
MATCH (r:Repo {path: $repoPath})-[:HAS_MEMORY]->(m:Memory)
WHERE 1=1
`
params := map[string]interface{}{
"repoPath": repoPath,
"limit":    limit,
}

// Add content search
if query != "" {
cypher += " AND (m.content CONTAINS $query OR m.related_to CONTAINS $query)"
params["query"] = query
}

// Add context type filter
if contextType != "" {
cypher += " AND m.context_type = $contextType"
params["contextType"] = contextType
}

// Add tags filter
if len(tags) > 0 {
cypher += " AND ANY(tag IN $tags WHERE tag IN m.tags)"
params["tags"] = tags
}

cypher += `
RETURN m.content, m.context_type, m.tags, m.related_to
ORDER BY m.updated_at DESC
LIMIT $limit
`

result, err := session.Run(ctx, cypher, params)
if err != nil {
return nil, fmt.Errorf("failed to search memory: %w", err)
}

var memories []*MemoryNode
for result.Next(ctx) {
record := result.Record()
memory := &MemoryNode{
Content:     getString(record.Values[0]),
ContextType: getString(record.Values[1]),
Tags:        getStringSlice(record.Values[2]),
RelatedTo:   getString(record.Values[3]),
}
memories = append(memories, memory)
}

return memories, nil
}

// SaveLearning creates or updates a learning/pattern node and links it to a repository.
func (c *Client) SaveLearning(ctx context.Context, repoPath string, learning *LearningNode) error {
if err := learning.Validate(); err != nil {
return fmt.Errorf("invalid learning node: %w", err)
}

session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
defer session.Close(ctx)

// Create unique ID based on pattern
learningID := fmt.Sprintf("%x", learning.Pattern[:min(50, len(learning.Pattern))])

query := `
MATCH (r:Repo {path: $repoPath})
MERGE (l:Learning {id: $learningID, repo_path: $repoPath})
SET l.pattern = $pattern,
    l.category = $category,
    l.description = $description,
    l.examples = $examples,
    l.updated_at = datetime()
MERGE (r)-[:HAS_LEARNING]->(l)
RETURN l
`

params := map[string]interface{}{
"repoPath":    repoPath,
"learningID":  learningID,
"pattern":     learning.Pattern,
"category":    learning.Category,
"description": learning.Description,
"examples":    learning.Examples,
}

_, err := session.Run(ctx, query, params)
if err != nil {
return fmt.Errorf("failed to save learning: %w", err)
}

return nil
}

// SearchLearnings searches for learnings/patterns in a repository.
func (c *Client) SearchLearnings(ctx context.Context, repoPath string, query string, category string, limit int) ([]*LearningNode, error) {
session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
defer session.Close(ctx)

if limit == 0 {
limit = 10
}

// Build query based on filters
cypher := `
MATCH (r:Repo {path: $repoPath})-[:HAS_LEARNING]->(l:Learning)
WHERE 1=1
`
params := map[string]interface{}{
"repoPath": repoPath,
"limit":    limit,
}

// Add search query
if query != "" {
cypher += " AND (l.pattern CONTAINS $query OR l.description CONTAINS $query)"
params["query"] = query
}

// Add category filter
if category != "" {
cypher += " AND l.category = $category"
params["category"] = category
}

cypher += `
RETURN l.pattern, l.category, l.description, l.examples
ORDER BY l.updated_at DESC
LIMIT $limit
`

result, err := session.Run(ctx, cypher, params)
if err != nil {
return nil, fmt.Errorf("failed to search learnings: %w", err)
}

var learnings []*LearningNode
for result.Next(ctx) {
record := result.Record()
learning := &LearningNode{
Pattern:     getString(record.Values[0]),
Category:    getString(record.Values[1]),
Description: getString(record.Values[2]),
Examples:    getStringSlice(record.Values[3]),
}
learnings = append(learnings, learning)
}

return learnings, nil
}

// getStringSlice safely converts an interface{} to []string.
func getStringSlice(v interface{}) []string {
if v == nil {
return nil
}
if slice, ok := v.([]interface{}); ok {
result := make([]string, 0, len(slice))
for _, item := range slice {
if s, ok := item.(string); ok {
result = append(result, s)
}
}
return result
}
if slice, ok := v.([]string); ok {
return slice
}
return nil
}

// min returns the minimum of two integers.
func min(a, b int) int {
if a < b {
return a
}
return b
}
