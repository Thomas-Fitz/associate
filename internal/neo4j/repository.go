package neo4j

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fitz/associate/internal/models"
	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Repository provides CRUD operations for memories.
type Repository struct {
	client *Client
}

// NewRepository creates a new repository
func NewRepository(client *Client) *Repository {
	return &Repository{client: client}
}

// Search finds memories matching the query
func (r *Repository) Search(ctx context.Context, query string, limit int) ([]models.SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	session := r.client.Session(ctx)
	defer session.Close(ctx)

	// Use full-text search with fallback to CONTAINS
	cypher := `
CALL db.index.fulltext.queryNodes('memory_content', $query) 
YIELD node, score
OPTIONAL MATCH (node)-[:RELATES_TO|PART_OF|REFERENCES|DEPENDS_ON|BLOCKS|FOLLOWS|IMPLEMENTS]-(related:Memory)
RETURN node, score, collect(DISTINCT related.id) as related_ids
ORDER BY score DESC
LIMIT $limit
`

	result, err := session.Run(ctx, cypher, map[string]any{
		"query": query,
		"limit": limit,
	})
	if err != nil {
		// Fallback to simple CONTAINS search if full-text fails
		return r.searchFallback(ctx, session, query, limit)
	}

	var results []models.SearchResult
	for result.Next(ctx) {
		record := result.Record()
		node, _ := record.Get("node")
		score, _ := record.Get("score")
		relatedIDs, _ := record.Get("related_ids")

		mem := nodeToMemory(node.(neo4j.Node))
		sr := models.SearchResult{
			Memory: mem,
			Score:  score.(float64),
		}

		if ids, ok := relatedIDs.([]any); ok {
			for _, id := range ids {
				if s, ok := id.(string); ok && s != "" {
					sr.Related = append(sr.Related, s)
				}
			}
		}
		results = append(results, sr)
	}

	return results, result.Err()
}

func (r *Repository) searchFallback(ctx context.Context, session neo4j.SessionWithContext, query string, limit int) ([]models.SearchResult, error) {
	cypher := `
MATCH (m:Memory)
WHERE m.content CONTAINS $query OR m.id CONTAINS $query
OPTIONAL MATCH (m)-[:RELATES_TO|PART_OF|REFERENCES|DEPENDS_ON|BLOCKS|FOLLOWS|IMPLEMENTS]-(related:Memory)
RETURN m, 1.0 as score, collect(DISTINCT related.id) as related_ids
LIMIT $limit
`

	result, err := session.Run(ctx, cypher, map[string]any{
		"query": query,
		"limit": limit,
	})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var results []models.SearchResult
	for result.Next(ctx) {
		record := result.Record()
		node, _ := record.Get("m")
		score, _ := record.Get("score")
		relatedIDs, _ := record.Get("related_ids")

		mem := nodeToMemory(node.(neo4j.Node))
		sr := models.SearchResult{
			Memory: mem,
			Score:  score.(float64),
		}

		if ids, ok := relatedIDs.([]any); ok {
			for _, id := range ids {
				if s, ok := id.(string); ok && s != "" {
					sr.Related = append(sr.Related, s)
				}
			}
		}
		results = append(results, sr)
	}

	return results, result.Err()
}

// Add creates a new memory and optional relationships
func (r *Repository) Add(ctx context.Context, mem models.Memory, relationships []models.Relationship) (*models.Memory, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	if mem.ID == "" {
		mem.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	mem.CreatedAt = now
	mem.UpdatedAt = now

	if mem.Type == "" {
		mem.Type = models.TypeGeneral
	}

	// Create the memory node - store metadata as JSON string
	metadataJSON := metadataToJSON(mem.Metadata)
	tags := mem.Tags
	if tags == nil {
		tags = []string{}
	}

	params := map[string]any{
		"id":         mem.ID,
		"type":       string(mem.Type),
		"content":    mem.Content,
		"metadata":   metadataJSON,
		"tags":       tags,
		"created_at": mem.CreatedAt.Format(time.RFC3339),
		"updated_at": mem.UpdatedAt.Format(time.RFC3339),
	}

	cypher := `
CREATE (m:Memory {
id: $id,
type: $type,
content: $content,
metadata: $metadata,
tags: $tags,
created_at: datetime($created_at),
updated_at: datetime($updated_at)
})
RETURN m
`

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory: %w", err)
	}

	if !result.Next(ctx) {
		if err := result.Err(); err != nil {
			return nil, fmt.Errorf("result iteration error: %w", err)
		}
		return nil, fmt.Errorf("no result returned from create")
	}

	// Create relationships
	for _, rel := range relationships {
		if err := r.createRelationship(ctx, session, mem.ID, rel.ToID, rel.Type); err != nil {
			// Log but don't fail - the node was created
			fmt.Printf("warning: failed to create relationship: %v\n", err)
		}
	}

	return &mem, nil
}

// Update modifies an existing memory
func (r *Repository) Update(ctx context.Context, id string, content *string, metadata map[string]string, tags []string, newRelationships []models.Relationship) (*models.Memory, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	// Build dynamic SET clause
	setClauses := []string{"m.updated_at = datetime($updated_at)"}
	params := map[string]any{
		"id":         id,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}

	if content != nil {
		setClauses = append(setClauses, "m.content = $content")
		params["content"] = *content
	}
	if metadata != nil {
		setClauses = append(setClauses, "m.metadata = $metadata")
		params["metadata"] = metadataToJSON(metadata)
	}
	if tags != nil {
		setClauses = append(setClauses, "m.tags = $tags")
		params["tags"] = tags
	}

	cypher := fmt.Sprintf(`
MATCH (m:Memory {id: $id})
SET %s
RETURN m
`, joinStrings(setClauses, ", "))

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update memory: %w", err)
	}

	if !result.Next(ctx) {
		return nil, fmt.Errorf("memory not found: %s", id)
	}

	node, _ := result.Record().Get("m")
	mem := nodeToMemory(node.(neo4j.Node))

	// Create new relationships
	for _, rel := range newRelationships {
		if err := r.createRelationship(ctx, session, id, rel.ToID, rel.Type); err != nil {
			fmt.Printf("warning: failed to create relationship: %v\n", err)
		}
	}

	return &mem, nil
}

// GetByID retrieves a memory by ID
func (r *Repository) GetByID(ctx context.Context, id string) (*models.Memory, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	cypher := `MATCH (m:Memory {id: $id}) RETURN m`
	result, err := session.Run(ctx, cypher, map[string]any{"id": id})
	if err != nil {
		return nil, err
	}

	if !result.Next(ctx) {
		return nil, nil // Not found
	}

	node, _ := result.Record().Get("m")
	mem := nodeToMemory(node.(neo4j.Node))
	return &mem, nil
}

func (r *Repository) createRelationship(ctx context.Context, session neo4j.SessionWithContext, fromID, toID string, relType models.RelationType) error {
	cypher := fmt.Sprintf(`
MATCH (a:Memory {id: $from_id})
MATCH (b:Memory {id: $to_id})
MERGE (a)-[r:%s]->(b)
RETURN r
`, relType)

	_, err := session.Run(ctx, cypher, map[string]any{
		"from_id": fromID,
		"to_id":   toID,
	})
	return err
}

// nodeToMemory converts a Neo4j node to a Memory struct
func nodeToMemory(node neo4j.Node) models.Memory {
	props := node.Props

	mem := models.Memory{
		ID:      getString(props, "id"),
		Type:    models.MemoryType(getString(props, "type")),
		Content: getString(props, "content"),
	}

	// Metadata is stored as JSON string
	if metadataStr, ok := props["metadata"].(string); ok && metadataStr != "" {
		mem.Metadata = jsonToMetadata(metadataStr)
	}

	if tags, ok := props["tags"].([]any); ok {
		for _, t := range tags {
			if s, ok := t.(string); ok {
				mem.Tags = append(mem.Tags, s)
			}
		}
	}

	if createdAt, ok := props["created_at"].(time.Time); ok {
		mem.CreatedAt = createdAt
	}
	if updatedAt, ok := props["updated_at"].(time.Time); ok {
		mem.UpdatedAt = updatedAt
	}

	return mem
}

func getString(props map[string]any, key string) string {
	if v, ok := props[key].(string); ok {
		return v
	}
	return ""
}

func metadataToJSON(m map[string]string) string {
	if m == nil || len(m) == 0 {
		return ""
	}
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}

func jsonToMetadata(s string) map[string]string {
	if s == "" {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	return m
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// GetByIDWithRelated retrieves a memory by ID along with its direct relationships
func (r *Repository) GetByIDWithRelated(ctx context.Context, id string) (*models.Memory, []models.RelatedInfo, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	cypher := `
MATCH (m:Memory {id: $id})
OPTIONAL MATCH (m)-[r]->(outgoing:Memory)
OPTIONAL MATCH (incoming:Memory)-[r2]->(m)
RETURN m, 
       collect(DISTINCT {id: outgoing.id, type: outgoing.type, rel_type: type(r), direction: 'outgoing'}) as outgoing_rels,
       collect(DISTINCT {id: incoming.id, type: incoming.type, rel_type: type(r2), direction: 'incoming'}) as incoming_rels
`

	result, err := session.Run(ctx, cypher, map[string]any{"id": id})
	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %w", err)
	}

	if !result.Next(ctx) {
		return nil, nil, nil // Not found
	}

	record := result.Record()
	node, _ := record.Get("m")
	outgoingRels, _ := record.Get("outgoing_rels")
	incomingRels, _ := record.Get("incoming_rels")

	mem := nodeToMemory(node.(neo4j.Node))

	var related []models.RelatedInfo

	// Process outgoing relationships
	if rels, ok := outgoingRels.([]any); ok {
		for _, rel := range rels {
			if m, ok := rel.(map[string]any); ok {
				if relID, ok := m["id"].(string); ok && relID != "" {
					related = append(related, models.RelatedInfo{
						ID:           relID,
						Type:         models.MemoryType(getString(m, "type")),
						RelationType: getString(m, "rel_type"),
						Direction:    "outgoing",
					})
				}
			}
		}
	}

	// Process incoming relationships
	if rels, ok := incomingRels.([]any); ok {
		for _, rel := range rels {
			if m, ok := rel.(map[string]any); ok {
				if relID, ok := m["id"].(string); ok && relID != "" {
					related = append(related, models.RelatedInfo{
						ID:           relID,
						Type:         models.MemoryType(getString(m, "type")),
						RelationType: getString(m, "rel_type"),
						Direction:    "incoming",
					})
				}
			}
		}
	}

	return &mem, related, nil
}

// Delete removes a memory and all its relationships
func (r *Repository) Delete(ctx context.Context, id string) error {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	cypher := `
MATCH (m:Memory {id: $id})
DETACH DELETE m
RETURN count(m) as deleted
`

	result, err := session.Run(ctx, cypher, map[string]any{"id": id})
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	if !result.Next(ctx) {
		return fmt.Errorf("memory not found: %s", id)
	}

	return nil
}

// GetRelated retrieves memories related to the given ID with optional filtering
func (r *Repository) GetRelated(ctx context.Context, id string, relationType string, direction string, depth int) ([]models.RelatedMemoryResult, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	// Build the relationship pattern based on direction and type
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
MATCH (a:Memory {id: $id})%s(b:Memory)
WHERE a <> b
WITH DISTINCT b, r, 
     CASE WHEN startNode(r[-1]) = a OR (size(r) > 1 AND startNode(r[-1]).id = $id) THEN 'outgoing' ELSE 'incoming' END as direction,
     size(r) as depth,
     type(r[-1]) as rel_type
RETURN b, rel_type, direction, depth
ORDER BY depth ASC
`, relPattern)

	result, err := session.Run(ctx, cypher, map[string]any{"id": id})
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	var results []models.RelatedMemoryResult
	seen := make(map[string]bool) // Prevent duplicates

	for result.Next(ctx) {
		record := result.Record()
		node, _ := record.Get("b")
		relType, _ := record.Get("rel_type")
		dir, _ := record.Get("direction")
		d, _ := record.Get("depth")

		mem := nodeToMemory(node.(neo4j.Node))

		// Skip duplicates
		if seen[mem.ID] {
			continue
		}
		seen[mem.ID] = true

		relTypeStr := ""
		if s, ok := relType.(string); ok {
			relTypeStr = s
		}

		dirStr := "both"
		if s, ok := dir.(string); ok {
			dirStr = s
		}

		depthVal := 1
		if i, ok := d.(int64); ok {
			depthVal = int(i)
		}

		results = append(results, models.RelatedMemoryResult{
			Memory:       mem,
			RelationType: relTypeStr,
			Direction:    dirStr,
			Depth:        depthVal,
		})
	}

	return results, result.Err()
}
