package neo4j

import (
	"context"
	"fmt"
	"time"

	"github.com/fitz/associate/internal/models"
	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// TaskRepository provides CRUD operations for tasks.
type TaskRepository struct {
	client *Client
}

// NewTaskRepository creates a new task repository
func NewTaskRepository(client *Client) *TaskRepository {
	return &TaskRepository{client: client}
}

// Add creates a new task with optional plan links and relationships
func (r *TaskRepository) Add(ctx context.Context, task models.Task, planIDs []string, relationships []models.Relationship) (*models.Task, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	task.CreatedAt = now
	task.UpdatedAt = now

	if task.Status == "" {
		task.Status = models.TaskStatusPending
	}

	// Create the task node
	metadataJSON := metadataToJSON(task.Metadata)
	tags := task.Tags
	if tags == nil {
		tags = []string{}
	}

	params := map[string]any{
		"id":         task.ID,
		"content":    task.Content,
		"status":     string(task.Status),
		"metadata":   metadataJSON,
		"tags":       tags,
		"created_at": task.CreatedAt.Format(time.RFC3339),
		"updated_at": task.UpdatedAt.Format(time.RFC3339),
	}

	cypher := `
CREATE (t:Task {
	id: $id,
	content: $content,
	status: $status,
	metadata: $metadata,
	tags: $tags,
	created_at: datetime($created_at),
	updated_at: datetime($updated_at)
})
RETURN t
`

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	if !result.Next(ctx) {
		if err := result.Err(); err != nil {
			return nil, fmt.Errorf("result iteration error: %w", err)
		}
		return nil, fmt.Errorf("no result returned from create")
	}

	// Create PART_OF relationships to plans
	for _, planID := range planIDs {
		if err := r.createTaskToPlanRelationship(ctx, session, task.ID, planID); err != nil {
			fmt.Printf("warning: failed to create plan relationship: %v\n", err)
		}
	}

	// Create other relationships
	for _, rel := range relationships {
		if err := r.createRelationshipFromTask(ctx, session, task.ID, rel.ToID, rel.Type); err != nil {
			fmt.Printf("warning: failed to create relationship: %v\n", err)
		}
	}

	return &task, nil
}

// GetByID retrieves a task by ID
func (r *TaskRepository) GetByID(ctx context.Context, id string) (*models.Task, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	cypher := `MATCH (t:Task {id: $id}) RETURN t`
	result, err := session.Run(ctx, cypher, map[string]any{"id": id})
	if err != nil {
		return nil, err
	}

	if !result.Next(ctx) {
		return nil, nil // Not found
	}

	node, _ := result.Record().Get("t")
	task := nodeToTask(node.(neo4j.Node))
	return &task, nil
}

// GetWithPlans retrieves a task by ID along with its associated plans
func (r *TaskRepository) GetWithPlans(ctx context.Context, id string) (*models.Task, []models.Plan, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	cypher := `
MATCH (t:Task {id: $id})
OPTIONAL MATCH (t)-[:PART_OF]->(p:Plan)
RETURN t, collect(p) as plans
`

	result, err := session.Run(ctx, cypher, map[string]any{"id": id})
	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %w", err)
	}

	if !result.Next(ctx) {
		return nil, nil, nil // Not found
	}

	record := result.Record()
	node, _ := record.Get("t")
	plansRaw, _ := record.Get("plans")

	task := nodeToTask(node.(neo4j.Node))

	var plans []models.Plan
	if planNodes, ok := plansRaw.([]any); ok {
		for _, pn := range planNodes {
			if pn != nil {
				if planNode, ok := pn.(neo4j.Node); ok {
					plans = append(plans, nodeToPlan(planNode))
				}
			}
		}
	}

	return &task, plans, nil
}

// Update modifies an existing task
func (r *TaskRepository) Update(ctx context.Context, id string, content *string, status *string, metadata map[string]string, tags []string, addPlanIDs []string, newRelationships []models.Relationship) (*models.Task, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	// Build dynamic SET clause
	setClauses := []string{"t.updated_at = datetime($updated_at)"}
	params := map[string]any{
		"id":         id,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}

	if content != nil {
		setClauses = append(setClauses, "t.content = $content")
		params["content"] = *content
	}
	if status != nil {
		setClauses = append(setClauses, "t.status = $status")
		params["status"] = *status
	}
	if metadata != nil {
		setClauses = append(setClauses, "t.metadata = $metadata")
		params["metadata"] = metadataToJSON(metadata)
	}
	if tags != nil {
		setClauses = append(setClauses, "t.tags = $tags")
		params["tags"] = tags
	}

	cypher := fmt.Sprintf(`
MATCH (t:Task {id: $id})
SET %s
RETURN t
`, joinStrings(setClauses, ", "))

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	if !result.Next(ctx) {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	node, _ := result.Record().Get("t")
	task := nodeToTask(node.(neo4j.Node))

	// Add to new plans
	for _, planID := range addPlanIDs {
		if err := r.createTaskToPlanRelationship(ctx, session, id, planID); err != nil {
			fmt.Printf("warning: failed to create plan relationship: %v\n", err)
		}
	}

	// Create new relationships
	for _, rel := range newRelationships {
		if err := r.createRelationshipFromTask(ctx, session, id, rel.ToID, rel.Type); err != nil {
			fmt.Printf("warning: failed to create relationship: %v\n", err)
		}
	}

	return &task, nil
}

// Delete removes a task and all its relationships
func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	cypher := `
MATCH (t:Task {id: $id})
DETACH DELETE t
RETURN count(t) as deleted
`

	result, err := session.Run(ctx, cypher, map[string]any{"id": id})
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	if !result.Next(ctx) {
		return fmt.Errorf("task not found: %s", id)
	}

	return nil
}

// List retrieves tasks with optional filtering
func (r *TaskRepository) List(ctx context.Context, planID string, status string, tags []string, limit int) ([]models.Task, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	if limit <= 0 {
		limit = 50
	}

	// Build query based on filters
	var cypher string
	params := map[string]any{"limit": limit}

	if planID != "" {
		// Filter by plan
		whereClauses := []string{}
		if status != "" {
			whereClauses = append(whereClauses, "t.status = $status")
			params["status"] = status
		}
		if len(tags) > 0 {
			whereClauses = append(whereClauses, "any(tag IN $tags WHERE tag IN t.tags)")
			params["tags"] = tags
		}

		whereClause := ""
		if len(whereClauses) > 0 {
			whereClause = "AND " + joinStrings(whereClauses, " AND ")
		}

		params["plan_id"] = planID
		cypher = fmt.Sprintf(`
MATCH (t:Task)-[:PART_OF]->(p:Plan {id: $plan_id})
WHERE true %s
RETURN t
ORDER BY t.updated_at DESC
LIMIT $limit
`, whereClause)
	} else {
		// No plan filter
		whereClauses := []string{}
		if status != "" {
			whereClauses = append(whereClauses, "t.status = $status")
			params["status"] = status
		}
		if len(tags) > 0 {
			whereClauses = append(whereClauses, "any(tag IN $tags WHERE tag IN t.tags)")
			params["tags"] = tags
		}

		whereClause := ""
		if len(whereClauses) > 0 {
			whereClause = "WHERE " + joinStrings(whereClauses, " AND ")
		}

		cypher = fmt.Sprintf(`
MATCH (t:Task)
%s
RETURN t
ORDER BY t.updated_at DESC
LIMIT $limit
`, whereClause)
	}

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("list failed: %w", err)
	}

	var tasks []models.Task
	for result.Next(ctx) {
		node, _ := result.Record().Get("t")
		tasks = append(tasks, nodeToTask(node.(neo4j.Node)))
	}

	return tasks, result.Err()
}

// createTaskToPlanRelationship creates a PART_OF relationship from task to plan
func (r *TaskRepository) createTaskToPlanRelationship(ctx context.Context, session neo4j.SessionWithContext, taskID, planID string) error {
	cypher := `
MATCH (t:Task {id: $task_id})
MATCH (p:Plan {id: $plan_id})
MERGE (t)-[r:PART_OF]->(p)
RETURN r
`
	_, err := session.Run(ctx, cypher, map[string]any{
		"task_id": taskID,
		"plan_id": planID,
	})
	return err
}

// createRelationshipFromTask creates a relationship from a Task to another node (any type)
func (r *TaskRepository) createRelationshipFromTask(ctx context.Context, session neo4j.SessionWithContext, fromID, toID string, relType models.RelationType) error {
	// Try to match any node type (Memory, Plan, or Task)
	cypher := fmt.Sprintf(`
MATCH (a:Task {id: $from_id})
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

// nodeToTask converts a Neo4j node to a Task struct
func nodeToTask(node neo4j.Node) models.Task {
	props := node.Props

	task := models.Task{
		ID:      getString(props, "id"),
		Content: getString(props, "content"),
		Status:  models.TaskStatus(getString(props, "status")),
	}

	// Metadata is stored as JSON string
	if metadataStr, ok := props["metadata"].(string); ok && metadataStr != "" {
		task.Metadata = jsonToMetadata(metadataStr)
	}

	if tags, ok := props["tags"].([]any); ok {
		for _, t := range tags {
			if s, ok := t.(string); ok {
				task.Tags = append(task.Tags, s)
			}
		}
	}

	if createdAt, ok := props["created_at"].(time.Time); ok {
		task.CreatedAt = createdAt
	}
	if updatedAt, ok := props["updated_at"].(time.Time); ok {
		task.UpdatedAt = updatedAt
	}

	return task
}
