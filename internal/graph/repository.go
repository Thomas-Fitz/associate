package graph

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fitz/associate/internal/models"
	"github.com/google/uuid"
)

// Repository provides CRUD operations for memories.
type Repository struct {
	client *Client
}

// NewRepository creates a new repository
func NewRepository(client *Client) *Repository {
	return &Repository{client: client}
}

// Search finds memories matching the query using Cypher string matching.
// Note: AGE uses agtype columns, not JSONB, so we can't use pg_trgm directly.
// Instead, we use Cypher's CONTAINS for case-insensitive matching.
func (r *Repository) Search(ctx context.Context, query string, limit int) ([]models.SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Use Cypher for search - CONTAINS for substring matching
	// toLower for case-insensitive matching
	escapedQuery := EscapeCypherString(strings.ToLower(query))
	cypher := fmt.Sprintf(
		`MATCH (m:Memory)
		 WHERE toLower(m.content) CONTAINS '%s' OR toLower(m.id) CONTAINS '%s'
		 RETURN m
		 LIMIT %d`,
		escapedQuery, escapedQuery, limit)

	rows, err := r.client.execCypher(ctx, tx, cypher, "m agtype")
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	type searchHit struct {
		mem models.Memory
	}
	var hits []searchHit

	for rows.Next() {
		var agtypeStr string
		if err := rows.Scan(&agtypeStr); err != nil {
			continue
		}

		props, err := parseAGTypeProperties(agtypeStr)
		if err != nil {
			continue
		}

		mem := propsToMemory(props)
		hits = append(hits, searchHit{mem: mem})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Phase 2: For each hit, fetch related Memory IDs via Cypher.
	var results []models.SearchResult
	for _, hit := range hits {
		sr := models.SearchResult{
			Memory: hit.mem,
			Score:  1.0, // Cypher CONTAINS doesn't provide similarity scores
		}

		// Fetch related IDs (best-effort)
		cypher := fmt.Sprintf(
			`MATCH (m:Memory {id: '%s'})-[:RELATES_TO|PART_OF|REFERENCES|DEPENDS_ON|BLOCKS|FOLLOWS|IMPLEMENTS]-(related:Memory)
			 RETURN related.id`,
			EscapeCypherString(hit.mem.ID))

		relRows, err := r.client.execCypher(ctx, tx, cypher, "related_id agtype")
		if err == nil {
			defer relRows.Close()
			seen := make(map[string]bool)
			for relRows.Next() {
				var relatedID string
				if err := relRows.Scan(&relatedID); err == nil {
					// Strip quotes from agtype string
					relatedID = strings.Trim(relatedID, "\"")
					if relatedID != "" && !seen[relatedID] {
						seen[relatedID] = true
						sr.Related = append(sr.Related, relatedID)
					}
				}
			}
		}

		results = append(results, sr)
	}

	tx.Commit()
	return results, nil
}

// Add creates a new memory and optional relationships
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
	tagsList := tagsToCypherList(mem.Tags)

	cypher := fmt.Sprintf(
		`CREATE (m:Memory {
			id: '%s',
			node_type: 'Memory',
			type: '%s',
			content: '%s',
			metadata: '%s',
			tags: %s,
			created_at: '%s',
			updated_at: '%s'
		}) RETURN m`,
		EscapeCypherString(mem.ID),
		EscapeCypherString(string(mem.Type)),
		EscapeCypherString(mem.Content),
		EscapeCypherString(metadataJSON),
		tagsList,
		mem.CreatedAt.Format(time.RFC3339),
		mem.UpdatedAt.Format(time.RFC3339),
	)

	rows, err := r.client.execCypher(ctx, tx, cypher, "m agtype")
	if err != nil {
		return nil, fmt.Errorf("failed to create memory: %w", err)
	}
	rows.Close()

	// Create relationships
	for _, rel := range relationships {
		if err := r.createRelationship(ctx, tx, mem.ID, rel.ToID, rel.Type); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to create relationship: %v\n", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return &mem, nil
}

// Update modifies an existing memory
func (r *Repository) Update(ctx context.Context, id string, content *string, metadata map[string]string, tags []string, newRelationships []models.Relationship) (*models.Memory, error) {
	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Build dynamic SET clause
	setClauses := []string{fmt.Sprintf("m.updated_at = '%s'", time.Now().UTC().Format(time.RFC3339))}

	if content != nil {
		setClauses = append(setClauses, fmt.Sprintf("m.content = '%s'", EscapeCypherString(*content)))
	}
	if metadata != nil {
		setClauses = append(setClauses, fmt.Sprintf("m.metadata = '%s'", EscapeCypherString(metadataToJSON(metadata))))
	}
	if tags != nil {
		setClauses = append(setClauses, fmt.Sprintf("m.tags = %s", tagsToCypherList(tags)))
	}

	cypher := fmt.Sprintf(`
		MATCH (m:Memory {id: '%s'})
		SET %s
		RETURN m`,
		EscapeCypherString(id),
		joinStrings(setClauses, ", "))

	rows, err := r.client.execCypher(ctx, tx, cypher, "m agtype")
	if err != nil {
		return nil, fmt.Errorf("failed to update memory: %w", err)
	}

	var mem *models.Memory
	if rows.Next() {
		var agtypeStr string
		if err := rows.Scan(&agtypeStr); err == nil {
			props, err := parseAGTypeProperties(agtypeStr)
			if err == nil {
				m := propsToMemory(props)
				mem = &m
			}
		}
	}
	rows.Close()

	if mem == nil {
		return nil, fmt.Errorf("memory not found: %s", id)
	}

	// Create new relationships
	for _, rel := range newRelationships {
		if err := r.createRelationship(ctx, tx, id, rel.ToID, rel.Type); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to create relationship: %v\n", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return mem, nil
}

// GetByID retrieves a memory by ID
func (r *Repository) GetByID(ctx context.Context, id string) (*models.Memory, error) {
	cypher := fmt.Sprintf(`MATCH (m:Memory {id: '%s'}) RETURN m`, EscapeCypherString(id))

	rows, err := r.client.execCypher(ctx, nil, cypher, "m agtype")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil // Not found
	}

	var agtypeStr string
	if err := rows.Scan(&agtypeStr); err != nil {
		return nil, err
	}

	props, err := parseAGTypeProperties(agtypeStr)
	if err != nil {
		return nil, err
	}

	mem := propsToMemory(props)
	return &mem, nil
}

// GetByIDWithRelated retrieves a memory by ID along with its direct relationships
func (r *Repository) GetByIDWithRelated(ctx context.Context, id string) (*models.Memory, []models.RelatedInfo, error) {
	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	// Get the memory
	cypher := fmt.Sprintf(`MATCH (m:Memory {id: '%s'}) RETURN m`, EscapeCypherString(id))
	rows, err := r.client.execCypher(ctx, tx, cypher, "m agtype")
	if err != nil {
		return nil, nil, err
	}

	var mem *models.Memory
	if rows.Next() {
		var agtypeStr string
		if err := rows.Scan(&agtypeStr); err == nil {
			props, err := parseAGTypeProperties(agtypeStr)
			if err == nil {
				m := propsToMemory(props)
				mem = &m
			}
		}
	}
	rows.Close()

	if mem == nil {
		return nil, nil, nil // Not found
	}

	var related []models.RelatedInfo

	// Get outgoing relationships
	outCypher := fmt.Sprintf(
		`MATCH (m:Memory {id: '%s'})-[r]->(out:Memory)
		 RETURN out.id, out.type, type(r)`,
		EscapeCypherString(id))
	outRows, err := r.client.execCypher(ctx, tx, outCypher, "out_id agtype, out_type agtype, rel_type agtype")
	if err == nil {
		for outRows.Next() {
			var outID, outType, relType string
			if err := outRows.Scan(&outID, &outType, &relType); err == nil {
				outID = strings.Trim(outID, "\"")
				outType = strings.Trim(outType, "\"")
				relType = strings.Trim(relType, "\"")
				if outID != "" {
					related = append(related, models.RelatedInfo{
						ID:           outID,
						Type:         models.MemoryType(outType),
						RelationType: relType,
						Direction:    "outgoing",
					})
				}
			}
		}
		outRows.Close()
	}

	// Get incoming relationships
	inCypher := fmt.Sprintf(
		`MATCH (inc:Memory)-[r]->(m:Memory {id: '%s'})
		 RETURN inc.id, inc.type, type(r)`,
		EscapeCypherString(id))
	inRows, err := r.client.execCypher(ctx, tx, inCypher, "inc_id agtype, inc_type agtype, rel_type agtype")
	if err == nil {
		for inRows.Next() {
			var incID, incType, relType string
			if err := inRows.Scan(&incID, &incType, &relType); err == nil {
				incID = strings.Trim(incID, "\"")
				incType = strings.Trim(incType, "\"")
				relType = strings.Trim(relType, "\"")
				if incID != "" {
					related = append(related, models.RelatedInfo{
						ID:           incID,
						Type:         models.MemoryType(incType),
						RelationType: relType,
						Direction:    "incoming",
					})
				}
			}
		}
		inRows.Close()
	}

	tx.Commit()
	return mem, related, nil
}

// Delete removes a memory and all its relationships
func (r *Repository) Delete(ctx context.Context, id string) error {
	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	cypher := fmt.Sprintf(`MATCH (m:Memory {id: '%s'}) DETACH DELETE m RETURN true`, EscapeCypherString(id))
	rows, err := r.client.execCypher(ctx, tx, cypher, "result agtype")
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}
	rows.Close()

	return tx.Commit()
}

// GetRelated retrieves nodes related to the given ID with optional filtering.
// Uses iterative depth expansion since AGE has limited support for variable-length path features.
func (r *Repository) GetRelated(ctx context.Context, id string, relationType string, direction string, depth int) ([]models.RelatedMemoryResult, error) {
	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var results []models.RelatedMemoryResult
	seen := map[string]bool{id: true}
	frontier := []string{id}

	for d := 1; d <= depth && len(frontier) > 0; d++ {
		var nextFrontier []string
		for _, currentID := range frontier {
			// Build relationship pattern for this hop
			var relPattern string
			if relationType != "" {
				switch direction {
				case "outgoing":
					relPattern = fmt.Sprintf("-[r:%s]->", relationType)
				case "incoming":
					relPattern = fmt.Sprintf("<-[r:%s]-", relationType)
				default:
					relPattern = fmt.Sprintf("-[r:%s]-", relationType)
				}
			} else {
				switch direction {
				case "outgoing":
					relPattern = "-[r]->"
				case "incoming":
					relPattern = "<-[r]-"
				default:
					relPattern = "-[r]-"
				}
			}

			cypher := fmt.Sprintf(
				`MATCH (a {id: '%s'})%s(b)
				 WHERE %s
				 RETURN b, type(r), b.node_type`,
				EscapeCypherString(currentID), relPattern, NodeLabelPredicate("b"))

			rows, err := r.client.execCypher(ctx, tx, cypher, "b agtype, rel_type agtype, node_type agtype")
			if err != nil {
				continue
			}

			for rows.Next() {
				var bStr, relTypeStr, nodeTypeStr string
				if err := rows.Scan(&bStr, &relTypeStr, &nodeTypeStr); err != nil {
					continue
				}

				props, err := parseAGTypeProperties(bStr)
				if err != nil {
					continue
				}

				nodeID := getString(props, "id")
				if seen[nodeID] {
					continue
				}
				seen[nodeID] = true
				nextFrontier = append(nextFrontier, nodeID)

				// Build result
				nodeType := strings.Trim(nodeTypeStr, "\"")
				if nodeType == "" {
					nodeType = "Memory"
				}

				mem := models.Memory{
					ID:   nodeID,
					Type: models.MemoryType(nodeType),
				}
				if nodeType == "Plan" {
					mem.Content = getString(props, "name")
				} else {
					mem.Content = getString(props, "content")
				}

				if metaStr := getString(props, "metadata"); metaStr != "" {
					mem.Metadata = jsonToMetadata(metaStr)
				}

				if tagsRaw, ok := props["tags"]; ok {
					if tagsArr, ok := tagsRaw.([]interface{}); ok {
						for _, t := range tagsArr {
							if s, ok := t.(string); ok {
								mem.Tags = append(mem.Tags, s)
							}
						}
					}
				}

				if createdStr := getString(props, "created_at"); createdStr != "" {
					if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
						mem.CreatedAt = t
					}
				}
				if updatedStr := getString(props, "updated_at"); updatedStr != "" {
					if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
						mem.UpdatedAt = t
					}
				}

				relTypeClean := strings.Trim(relTypeStr, "\"")

				// Determine direction based on query pattern
				dirStr := direction
				if dirStr == "" || dirStr == "both" {
					dirStr = "both"
				}

				results = append(results, models.RelatedMemoryResult{
					Memory:       mem,
					RelationType: relTypeClean,
					Direction:    dirStr,
					Depth:        d,
				})
			}
			rows.Close()
		}
		frontier = nextFrontier
	}

	tx.Commit()
	return results, nil
}

// createRelationship creates a relationship between two nodes.
// Uses check-then-create pattern since AGE doesn't support MERGE for relationships.
func (r *Repository) createRelationship(ctx context.Context, tx *sql.Tx, fromID, toID string, relType models.RelationType) error {
	// Validate relationship type
	if err := ValidateRelationType(relType); err != nil {
		return err
	}

	// Check if relationship already exists
	checkCypher := fmt.Sprintf(
		`MATCH (a)-[r:%s]->(b)
		 WHERE a.id = '%s' AND b.id = '%s'
		 RETURN r`,
		relType,
		EscapeCypherString(fromID),
		EscapeCypherString(toID),
	)

	rows, err := r.client.execCypher(ctx, tx, checkCypher, "r agtype")
	if err != nil {
		return err
	}

	exists := rows.Next()
	rows.Close()

	if exists {
		return nil // Relationship already exists
	}

	// Create the relationship - use label() function for AGE-compatible label filtering
	createCypher := fmt.Sprintf(
		`MATCH (a), (b)
		 WHERE a.id = '%s' AND b.id = '%s' AND %s AND %s
		 CREATE (a)-[r:%s]->(b)
		 RETURN r`,
		EscapeCypherString(fromID),
		EscapeCypherString(toID),
		NodeLabelPredicate("a"),
		NodeLabelPredicate("b"),
		relType,
	)

	createRows, err := r.client.execCypher(ctx, tx, createCypher, "r agtype")
	if err != nil {
		return err
	}
	createRows.Close()

	return nil
}
