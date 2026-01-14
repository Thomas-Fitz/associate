package tools

import (
"context"
"encoding/json"
"log/slog"

"github.com/fitz/associate/internal/models"
"github.com/fitz/associate/internal/neo4j"
"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Handler provides the dependencies needed by tool handlers.
type Handler struct {
Repo   *neo4j.Repository
Logger *slog.Logger
}

// NewHandler creates a new Handler with the given dependencies.
func NewHandler(repo *neo4j.Repository, logger *slog.Logger) *Handler {
return &Handler{
Repo:   repo,
Logger: logger,
}
}

// RelatedMemory contains summary info about a related memory.
type RelatedMemory struct {
ID           string `json:"id"`
Type         string `json:"type"`
RelationType string `json:"relationship_type"`
Direction    string `json:"direction"` // "incoming" or "outgoing"
}

// RelatedMemoryFull contains full info about a related memory.
type RelatedMemoryFull struct {
ID           string            `json:"id"`
Type         string            `json:"type"`
Content      string            `json:"content"`
Metadata     map[string]string `json:"metadata,omitempty"`
Tags         []string          `json:"tags,omitempty"`
RelationType string            `json:"relationship_type"`
Direction    string            `json:"direction"`
Depth        int               `json:"depth"`
CreatedAt    string            `json:"created_at"`
UpdatedAt    string            `json:"updated_at"`
}

// DerefSlice safely dereferences a pointer to a slice, returning nil if the pointer is nil.
// Exported for testing purposes.
func DerefSlice(p *[]string) []string {
	return derefSlice(p)
}

// derefSlice safely dereferences a pointer to a slice, returning nil if the pointer is nil.
func derefSlice(p *[]string) []string {
if p == nil {
return nil
}
return *p
}

// ConvertMetadata converts a map[string]any to map[string]string.
// Exported for testing purposes.
func ConvertMetadata(m map[string]any) map[string]string {
	return convertMetadata(m)
}

// convertMetadata converts a map[string]any to map[string]string.
// String values are kept as-is; other values are JSON-serialized.
func convertMetadata(m map[string]any) map[string]string {
if m == nil {
return nil
}
result := make(map[string]string, len(m))
for k, v := range m {
switch val := v.(type) {
case string:
result[k] = val
case nil:
// Skip nil values
default:
// Serialize non-string values as JSON
if b, err := json.Marshal(val); err == nil {
result[k] = string(b)
}
}
}
return result
}

// buildRelationships builds a slice of relationships from the input pointer slices.
func buildRelationships(
relatedTo, partOf, references, dependsOn, blocks, follows, implements *[]string,
) []models.Relationship {
var rels []models.Relationship
for _, id := range derefSlice(relatedTo) {
rels = append(rels, models.Relationship{ToID: id, Type: models.RelRelatesTo})
}
for _, id := range derefSlice(partOf) {
rels = append(rels, models.Relationship{ToID: id, Type: models.RelPartOf})
}
for _, id := range derefSlice(references) {
rels = append(rels, models.Relationship{ToID: id, Type: models.RelReferences})
}
for _, id := range derefSlice(dependsOn) {
rels = append(rels, models.Relationship{ToID: id, Type: models.RelDependsOn})
}
for _, id := range derefSlice(blocks) {
rels = append(rels, models.Relationship{ToID: id, Type: models.RelBlocks})
}
for _, id := range derefSlice(follows) {
rels = append(rels, models.Relationship{ToID: id, Type: models.RelFollows})
}
for _, id := range derefSlice(implements) {
rels = append(rels, models.Relationship{ToID: id, Type: models.RelImplements})
}
return rels
}

// Unused import placeholders to satisfy compiler during incremental development.
var (
_ context.Context
_ *mcp.CallToolRequest
)
