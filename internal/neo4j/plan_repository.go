package neo4j

import (
	"context"
	"fmt"
	"time"

	"github.com/fitz/associate/internal/models"
	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// PlanRepository provides CRUD operations for plans.
type PlanRepository struct {
	client *Client
}

// NewPlanRepository creates a new plan repository
func NewPlanRepository(client *Client) *PlanRepository {
	return &PlanRepository{client: client}
}

// Add creates a new plan and optional relationships
func (r *PlanRepository) Add(ctx context.Context, plan models.Plan, relationships []models.Relationship) (*models.Plan, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	if plan.ID == "" {
		plan.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	plan.CreatedAt = now
	plan.UpdatedAt = now

	if plan.Status == "" {
		plan.Status = models.PlanStatusActive
	}

	// Create the plan node
	metadataJSON := metadataToJSON(plan.Metadata)
	tags := plan.Tags
	if tags == nil {
		tags = []string{}
	}

	params := map[string]any{
		"id":          plan.ID,
		"name":        plan.Name,
		"description": plan.Description,
		"status":      string(plan.Status),
		"metadata":    metadataJSON,
		"tags":        tags,
		"created_at":  plan.CreatedAt.Format(time.RFC3339),
		"updated_at":  plan.UpdatedAt.Format(time.RFC3339),
	}

	cypher := `
CREATE (p:Plan {
	id: $id,
	name: $name,
	description: $description,
	status: $status,
	metadata: $metadata,
	tags: $tags,
	created_at: datetime($created_at),
	updated_at: datetime($updated_at)
})
RETURN p
`

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	if !result.Next(ctx) {
		if err := result.Err(); err != nil {
			return nil, fmt.Errorf("result iteration error: %w", err)
		}
		return nil, fmt.Errorf("no result returned from create")
	}

	// Create relationships
	for _, rel := range relationships {
		if err := r.createRelationshipFromPlan(ctx, session, plan.ID, rel.ToID, rel.Type); err != nil {
			// Log warning but don't fail
			fmt.Printf("warning: failed to create relationship: %v\n", err)
		}
	}

	return &plan, nil
}

// GetByID retrieves a plan by ID
func (r *PlanRepository) GetByID(ctx context.Context, id string) (*models.Plan, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	cypher := `MATCH (p:Plan {id: $id}) RETURN p`
	result, err := session.Run(ctx, cypher, map[string]any{"id": id})
	if err != nil {
		return nil, err
	}

	if !result.Next(ctx) {
		return nil, nil // Not found
	}

	node, _ := result.Record().Get("p")
	plan := nodeToPlan(node.(neo4j.Node))
	return &plan, nil
}

// GetWithTasks retrieves a plan by ID along with all its tasks ordered by position.
// Returns TaskInPlan which includes position and dependency information.
func (r *PlanRepository) GetWithTasks(ctx context.Context, id string) (*models.Plan, []models.TaskInPlan, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	// First get the plan
	planCypher := `MATCH (p:Plan {id: $id}) RETURN p`
	planResult, err := session.Run(ctx, planCypher, map[string]any{"id": id})
	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %w", err)
	}

	if !planResult.Next(ctx) {
		return nil, nil, nil // Not found
	}

	planNode, _ := planResult.Record().Get("p")
	plan := nodeToPlan(planNode.(neo4j.Node))

	// Now get tasks with position and dependencies
	tasksCypher := `
MATCH (p:Plan {id: $id})
OPTIONAL MATCH (t:Task)-[r:PART_OF]->(p)
OPTIONAL MATCH (t)-[:DEPENDS_ON]->(dep:Task)-[:PART_OF]->(p)
OPTIONAL MATCH (t)-[:BLOCKS]->(blk:Task)-[:PART_OF]->(p)
WITH t, r, collect(DISTINCT dep.id) as depends_on, collect(DISTINCT blk.id) as blocks
WHERE t IS NOT NULL
RETURN t, COALESCE(r.position, 0.0) as position, depends_on, blocks
ORDER BY position ASC
`
	tasksResult, err := session.Run(ctx, tasksCypher, map[string]any{"id": id})
	if err != nil {
		return nil, nil, fmt.Errorf("tasks query failed: %w", err)
	}

	var tasks []models.TaskInPlan
	for tasksResult.Next(ctx) {
		record := tasksResult.Record()
		taskNode, _ := record.Get("t")
		position, _ := record.Get("position")
		dependsOnRaw, _ := record.Get("depends_on")
		blocksRaw, _ := record.Get("blocks")

		task := nodeToTask(taskNode.(neo4j.Node))
		taskInPlan := models.TaskInPlan{
			Task:     task,
			Position: toFloat64(position),
		}

		// Convert depends_on array
		if deps, ok := dependsOnRaw.([]any); ok {
			for _, d := range deps {
				if s, ok := d.(string); ok && s != "" {
					taskInPlan.DependsOn = append(taskInPlan.DependsOn, s)
				}
			}
		}

		// Convert blocks array
		if blks, ok := blocksRaw.([]any); ok {
			for _, b := range blks {
				if s, ok := b.(string); ok && s != "" {
					taskInPlan.Blocks = append(taskInPlan.Blocks, s)
				}
			}
		}

		tasks = append(tasks, taskInPlan)
	}

	return &plan, tasks, nil
}

// Update modifies an existing plan
func (r *PlanRepository) Update(ctx context.Context, id string, name *string, description *string, status *string, metadata map[string]string, tags []string, newRelationships []models.Relationship) (*models.Plan, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	// Build dynamic SET clause
	setClauses := []string{"p.updated_at = datetime($updated_at)"}
	params := map[string]any{
		"id":         id,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}

	if name != nil {
		setClauses = append(setClauses, "p.name = $name")
		params["name"] = *name
	}
	if description != nil {
		setClauses = append(setClauses, "p.description = $description")
		params["description"] = *description
	}
	if status != nil {
		setClauses = append(setClauses, "p.status = $status")
		params["status"] = *status
	}
	if metadata != nil {
		setClauses = append(setClauses, "p.metadata = $metadata")
		params["metadata"] = metadataToJSON(metadata)
	}
	if tags != nil {
		setClauses = append(setClauses, "p.tags = $tags")
		params["tags"] = tags
	}

	cypher := fmt.Sprintf(`
MATCH (p:Plan {id: $id})
SET %s
RETURN p
`, joinStrings(setClauses, ", "))

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update plan: %w", err)
	}

	if !result.Next(ctx) {
		return nil, fmt.Errorf("plan not found: %s", id)
	}

	node, _ := result.Record().Get("p")
	plan := nodeToPlan(node.(neo4j.Node))

	// Create new relationships
	for _, rel := range newRelationships {
		if err := r.createRelationshipFromPlan(ctx, session, id, rel.ToID, rel.Type); err != nil {
			fmt.Printf("warning: failed to create relationship: %v\n", err)
		}
	}

	return &plan, nil
}

// Delete removes a plan and cascades to tasks not linked to other plans
func (r *PlanRepository) Delete(ctx context.Context, id string) (int, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	// First, find and delete tasks that only belong to this plan
	// Tasks with PART_OF relationships to other plans are kept
	// Note: We collect all tasks first, then filter in a list comprehension to handle
	// the case where the plan has no tasks (avoids WHERE filtering out null rows)
	deleteCypher := `
MATCH (p:Plan {id: $id})
OPTIONAL MATCH (t:Task)-[:PART_OF]->(p)
WITH p, collect(t) as allTasks
WITH p, [task IN allTasks WHERE task IS NOT NULL AND NOT EXISTS {
    MATCH (task)-[:PART_OF]->(other:Plan)
    WHERE other.id <> p.id
}] as tasksToDelete
FOREACH (task IN tasksToDelete | DETACH DELETE task)
DETACH DELETE p
RETURN size(tasksToDelete) as deletedCount
`

	result, err := session.Run(ctx, deleteCypher, map[string]any{"id": id})
	if err != nil {
		return 0, fmt.Errorf("delete failed: %w", err)
	}

	if !result.Next(ctx) {
		return 0, fmt.Errorf("plan not found: %s", id)
	}

	deletedCount, _ := result.Record().Get("deletedCount")
	count := 0
	if c, ok := deletedCount.(int64); ok {
		count = int(c)
	}

	return count, nil
}

// List retrieves plans with optional filtering
func (r *PlanRepository) List(ctx context.Context, status string, tags []string, limit int) ([]models.Plan, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	if limit <= 0 {
		limit = 50
	}

	// Build dynamic WHERE clause
	whereClauses := []string{}
	params := map[string]any{"limit": limit}

	if status != "" {
		whereClauses = append(whereClauses, "p.status = $status")
		params["status"] = status
	}
	if len(tags) > 0 {
		whereClauses = append(whereClauses, "any(tag IN $tags WHERE tag IN p.tags)")
		params["tags"] = tags
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + joinStrings(whereClauses, " AND ")
	}

	cypher := fmt.Sprintf(`
MATCH (p:Plan)
%s
RETURN p
ORDER BY p.updated_at DESC
LIMIT $limit
`, whereClause)

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("list failed: %w", err)
	}

	var plans []models.Plan
	for result.Next(ctx) {
		node, _ := result.Record().Get("p")
		plans = append(plans, nodeToPlan(node.(neo4j.Node)))
	}

	return plans, result.Err()
}

// createRelationshipFromPlan creates a relationship from a Plan to another node (any type)
func (r *PlanRepository) createRelationshipFromPlan(ctx context.Context, session neo4j.SessionWithContext, fromID, toID string, relType models.RelationType) error {
	// Try to match any node type (Memory, Plan, or Task)
	cypher := fmt.Sprintf(`
MATCH (a:Plan {id: $from_id})
MATCH (b) WHERE b.id = $to_id AND (b:Memory OR b:Plan OR b:Task)
MERGE (a)-[r:%s]->(b)
RETURN r
`, relType)

	_, err := session.Run(ctx, cypher, map[string]any{
		"from_id": fromID,
		"to_id":   toID,
	})
	return err
}

// nodeToPlan converts a Neo4j node to a Plan struct
func nodeToPlan(node neo4j.Node) models.Plan {
	props := node.Props

	plan := models.Plan{
		ID:          getString(props, "id"),
		Name:        getString(props, "name"),
		Description: getString(props, "description"),
		Status:      models.PlanStatus(getString(props, "status")),
	}

	// Metadata is stored as JSON string
	if metadataStr, ok := props["metadata"].(string); ok && metadataStr != "" {
		plan.Metadata = jsonToMetadata(metadataStr)
	}

	if tags, ok := props["tags"].([]any); ok {
		for _, t := range tags {
			if s, ok := t.(string); ok {
				plan.Tags = append(plan.Tags, s)
			}
		}
	}

	if createdAt, ok := props["created_at"].(time.Time); ok {
		plan.CreatedAt = createdAt
	}
	if updatedAt, ok := props["updated_at"].(time.Time); ok {
		plan.UpdatedAt = updatedAt
	}

	return plan
}
