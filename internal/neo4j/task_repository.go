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

// Add creates a new task with required plan links and optional relationships.
// Tasks must belong to at least one plan. If any plan doesn't exist or relationship
// creation fails, the operation is rolled back and an error is returned.
// Optional afterTaskID and beforeTaskID control positioning within plans:
//   - If neither is specified, the task is appended to the end of each plan
//   - If afterTaskID is specified, the task is positioned after that task
//   - If beforeTaskID is specified, the task is positioned before that task
func (r *TaskRepository) Add(ctx context.Context, task models.Task, planIDs []string, relationships []models.Relationship, afterTaskID, beforeTaskID *string) (*models.Task, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	// Validate: at least one plan is required
	if len(planIDs) == 0 {
		return nil, fmt.Errorf("task must belong to at least one plan")
	}

	// Verify all plans exist BEFORE creating the task
	for _, planID := range planIDs {
		exists, err := r.planExists(ctx, session, planID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify plan %s: %w", planID, err)
		}
		if !exists {
			return nil, fmt.Errorf("plan not found: %s", planID)
		}
	}

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

	// Create PART_OF relationships to plans - fail and rollback if any fail
	for _, planID := range planIDs {
		// Calculate position based on afterTaskID/beforeTaskID or append to end
		position, err := r.calculateNewTaskPosition(ctx, session, planID, afterTaskID, beforeTaskID)
		if err != nil {
			_ = r.Delete(ctx, task.ID) // Best-effort rollback
			return nil, fmt.Errorf("failed to calculate position for plan %s: %w", planID, err)
		}

		if err := r.createTaskToPlanRelationship(ctx, session, task.ID, planID, position); err != nil {
			fmt.Printf("warning: failed to create plan relationship to %s: %v\n", planID, err)
			// Rollback: delete the task we just created
			_ = r.Delete(ctx, task.ID) // Best-effort rollback
			return nil, fmt.Errorf("failed to link task to plan %s: %w", planID, err)
		}
	}

	// Create other relationships - fail and rollback if any fail
	for _, rel := range relationships {
		if err := r.createRelationshipFromTask(ctx, session, task.ID, rel.ToID, rel.Type); err != nil {
			fmt.Printf("warning: failed to create %s relationship to %s: %v\n", rel.Type, rel.ToID, err)
			// Rollback: delete the task we just created
			_ = r.Delete(ctx, task.ID) // Best-effort rollback
			return nil, fmt.Errorf("failed to create %s relationship to %s: %w", rel.Type, rel.ToID, err)
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

// Update modifies an existing task. If adding to new plans, verifies they exist first.
// If any plan doesn't exist or relationship creation fails, the entire update fails.
func (r *TaskRepository) Update(ctx context.Context, id string, content *string, status *string, metadata map[string]string, tags []string, addPlanIDs []string, newRelationships []models.Relationship) (*models.Task, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	// Verify all plans exist BEFORE making any changes
	for _, planID := range addPlanIDs {
		exists, err := r.planExists(ctx, session, planID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify plan %s: %w", planID, err)
		}
		if !exists {
			return nil, fmt.Errorf("plan not found: %s", planID)
		}
	}

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

	// Add to new plans - fail on error (always append to end when updating)
	for _, planID := range addPlanIDs {
		maxPos, posErr := r.getMaxPosition(ctx, session, planID)
		if posErr != nil {
			return nil, fmt.Errorf("failed to get max position for plan %s: %w", planID, posErr)
		}
		position := maxPos + DefaultPositionIncrement

		if err := r.createTaskToPlanRelationship(ctx, session, id, planID, position); err != nil {
			fmt.Printf("warning: failed to create plan relationship to %s: %v\n", planID, err)
			return nil, fmt.Errorf("failed to link task to plan %s: %w", planID, err)
		}
	}

	// Create new relationships - fail on error
	for _, rel := range newRelationships {
		if err := r.createRelationshipFromTask(ctx, session, id, rel.ToID, rel.Type); err != nil {
			fmt.Printf("warning: failed to create %s relationship to %s: %v\n", rel.Type, rel.ToID, err)
			return nil, fmt.Errorf("failed to create %s relationship to %s: %w", rel.Type, rel.ToID, err)
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

// UpdatePositions batch updates task positions within a plan.
// taskPositions maps task IDs to their new positions.
func (r *TaskRepository) UpdatePositions(ctx context.Context, planID string, taskPositions map[string]float64) error {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	// Update each task's position in the plan
	for taskID, position := range taskPositions {
		cypher := `
MATCH (t:Task {id: $task_id})-[r:PART_OF]->(p:Plan {id: $plan_id})
SET r.position = $position
RETURN r
`
		result, err := session.Run(ctx, cypher, map[string]any{
			"task_id":  taskID,
			"plan_id":  planID,
			"position": position,
		})
		if err != nil {
			return fmt.Errorf("failed to update position for task %s: %w", taskID, err)
		}
		if !result.Next(ctx) {
			return fmt.Errorf("task %s not found in plan %s", taskID, planID)
		}
	}

	return nil
}

// List retrieves tasks with optional filtering.
// When planID is provided, tasks are ordered by position and Position is populated.
// When planID is not provided, tasks are ordered by updated_at DESC and Position is nil.
func (r *TaskRepository) List(ctx context.Context, planID string, status string, tags []string, limit int) ([]models.TaskListResult, error) {
	session := r.client.Session(ctx)
	defer session.Close(ctx)

	if limit <= 0 {
		limit = 50
	}

	// Build query based on filters
	var cypher string
	params := map[string]any{"limit": limit}
	hasPosition := false

	if planID != "" {
		// Filter by plan - order by position within the plan
		hasPosition = true
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
MATCH (t:Task)-[r:PART_OF]->(p:Plan {id: $plan_id})
WHERE true %s
RETURN t, r.position as position
ORDER BY r.position ASC
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
RETURN t, null as position
ORDER BY t.updated_at DESC
LIMIT $limit
`, whereClause)
	}

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("list failed: %w", err)
	}

	var tasks []models.TaskListResult
	for result.Next(ctx) {
		node, _ := result.Record().Get("t")
		task := nodeToTask(node.(neo4j.Node))

		taskResult := models.TaskListResult{Task: task}
		if hasPosition {
			pos, _ := result.Record().Get("position")
			if pos != nil {
				posFloat := toFloat64(pos)
				taskResult.Position = &posFloat
			}
		}
		tasks = append(tasks, taskResult)
	}

	return tasks, result.Err()
}

// createTaskToPlanRelationship creates a PART_OF relationship from task to plan with a position
func (r *TaskRepository) createTaskToPlanRelationship(ctx context.Context, session neo4j.SessionWithContext, taskID, planID string, position float64) error {
	cypher := `
MATCH (t:Task {id: $task_id})
MATCH (p:Plan {id: $plan_id})
MERGE (t)-[r:PART_OF]->(p)
SET r.position = $position
RETURN r
`
	_, err := session.Run(ctx, cypher, map[string]any{
		"task_id":  taskID,
		"plan_id":  planID,
		"position": position,
	})
	return err
}

// planExists checks if a plan with the given ID exists in the database
func (r *TaskRepository) planExists(ctx context.Context, session neo4j.SessionWithContext, planID string) (bool, error) {
	cypher := `MATCH (p:Plan {id: $id}) RETURN count(p) > 0 as exists`
	result, err := session.Run(ctx, cypher, map[string]any{"id": planID})
	if err != nil {
		return false, err
	}
	if !result.Next(ctx) {
		return false, nil
	}
	exists, _ := result.Record().Get("exists")
	if b, ok := exists.(bool); ok {
		return b, nil
	}
	return false, nil
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

// Position constants for task ordering
const (
	// DefaultPositionIncrement is the default spacing between positions
	DefaultPositionIncrement = 1000.0
)

// getMaxPosition returns the maximum position value for tasks in a plan.
// Returns 0.0 if no tasks exist in the plan.
func (r *TaskRepository) getMaxPosition(ctx context.Context, session neo4j.SessionWithContext, planID string) (float64, error) {
	cypher := `
MATCH (t:Task)-[r:PART_OF]->(p:Plan {id: $plan_id})
RETURN COALESCE(max(r.position), 0.0) as max_pos
`
	result, err := session.Run(ctx, cypher, map[string]any{"plan_id": planID})
	if err != nil {
		return 0, fmt.Errorf("failed to get max position: %w", err)
	}

	if !result.Next(ctx) {
		return 0, nil
	}

	maxPos, _ := result.Record().Get("max_pos")
	return toFloat64(maxPos), nil
}

// getTaskPosition returns the position of a task within a specific plan.
// Returns 0.0 if the task is not in the plan or has no position.
func (r *TaskRepository) getTaskPosition(ctx context.Context, session neo4j.SessionWithContext, taskID, planID string) (float64, error) {
	cypher := `
MATCH (t:Task {id: $task_id})-[r:PART_OF]->(p:Plan {id: $plan_id})
RETURN COALESCE(r.position, 0.0) as position
`
	result, err := session.Run(ctx, cypher, map[string]any{
		"task_id": taskID,
		"plan_id": planID,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get task position: %w", err)
	}

	if !result.Next(ctx) {
		return 0, nil // Task not in plan
	}

	pos, _ := result.Record().Get("position")
	return toFloat64(pos), nil
}

// getAdjacentPositions returns the positions of tasks immediately before and after
// the task with the given ID in a plan. Returns (0, 0) if task not found.
// before will be 0 if this is the first task, after will be 0 if this is the last task.
func (r *TaskRepository) getAdjacentPositions(ctx context.Context, session neo4j.SessionWithContext, taskID, planID string) (before, after float64, err error) {
	// First get the task's current position
	currentPos, err := r.getTaskPosition(ctx, session, taskID, planID)
	if err != nil {
		return 0, 0, err
	}

	// Get the position of the task immediately before this one
	beforeCypher := `
MATCH (t:Task)-[r:PART_OF]->(p:Plan {id: $plan_id})
WHERE r.position < $current_pos
RETURN r.position as position
ORDER BY r.position DESC
LIMIT 1
`
	result, err := session.Run(ctx, beforeCypher, map[string]any{
		"plan_id":     planID,
		"current_pos": currentPos,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get before position: %w", err)
	}
	if result.Next(ctx) {
		pos, _ := result.Record().Get("position")
		before = toFloat64(pos)
	}

	// Get the position of the task immediately after this one
	afterCypher := `
MATCH (t:Task)-[r:PART_OF]->(p:Plan {id: $plan_id})
WHERE r.position > $current_pos
RETURN r.position as position
ORDER BY r.position ASC
LIMIT 1
`
	result, err = session.Run(ctx, afterCypher, map[string]any{
		"plan_id":     planID,
		"current_pos": currentPos,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get after position: %w", err)
	}
	if result.Next(ctx) {
		pos, _ := result.Record().Get("position")
		after = toFloat64(pos)
	}

	return before, after, nil
}

// calculateNewTaskPosition calculates the position for a new task in a plan based on
// optional afterTaskID and beforeTaskID parameters.
// - If neither is specified, appends to the end of the plan
// - If afterTaskID is specified, positions after that task
// - If beforeTaskID is specified, positions before that task
// - If both are specified, positions between them
func (r *TaskRepository) calculateNewTaskPosition(ctx context.Context, session neo4j.SessionWithContext, planID string, afterTaskID, beforeTaskID *string) (float64, error) {
	var afterPos, beforePos float64
	var err error

	// Get position of the "after" task if specified
	if afterTaskID != nil && *afterTaskID != "" {
		afterPos, err = r.getTaskPosition(ctx, session, *afterTaskID, planID)
		if err != nil {
			return 0, fmt.Errorf("failed to get position of after task: %w", err)
		}
		// Note: afterPos could be 0 if task not found in plan - this is handled below

		// If only afterTaskID is specified, we also need to find what comes after it
		if beforeTaskID == nil || *beforeTaskID == "" {
			_, afterPos2, err := r.getAdjacentPositions(ctx, session, *afterTaskID, planID)
			if err != nil {
				return 0, fmt.Errorf("failed to get adjacent positions: %w", err)
			}
			beforePos = afterPos2 // Could be 0 if inserting at end
		}
	}

	// Get position of the "before" task if specified
	if beforeTaskID != nil && *beforeTaskID != "" {
		beforePos, err = r.getTaskPosition(ctx, session, *beforeTaskID, planID)
		if err != nil {
			return 0, fmt.Errorf("failed to get position of before task: %w", err)
		}
		// If only beforeTaskID is specified and no afterTaskID, find what comes before it
		if afterTaskID == nil || *afterTaskID == "" {
			beforePos2, _, err := r.getAdjacentPositions(ctx, session, *beforeTaskID, planID)
			if err != nil {
				return 0, fmt.Errorf("failed to get adjacent positions: %w", err)
			}
			afterPos = beforePos2 // Could be 0 if inserting at start
		}
	}

	// If neither specified, append to end
	if (afterTaskID == nil || *afterTaskID == "") && (beforeTaskID == nil || *beforeTaskID == "") {
		maxPos, err := r.getMaxPosition(ctx, session, planID)
		if err != nil {
			return 0, fmt.Errorf("failed to get max position: %w", err)
		}
		return maxPos + DefaultPositionIncrement, nil
	}

	// Calculate position for single task insertion
	positions := CalculateInsertPositions(afterPos, beforePos, 1)
	if len(positions) == 0 {
		return DefaultPositionIncrement, nil
	}
	return positions[0], nil
}

// CalculateInsertPositions calculates position values for inserting tasks between
// afterPos and beforePos. If afterPos is 0, positions start from beforePos - increment*count.
// If beforePos is 0, positions start from afterPos + increment.
// Returns a slice of positions for the tasks to be inserted.
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
		// Calculate positions that fit before beforePos
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

// toFloat64 converts various numeric types to float64
func toFloat64(v any) float64 {
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
