package graph

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/Thomas-Fitz/associate/internal/models"
	"github.com/google/uuid"
)

// TaskRepository provides CRUD operations for tasks.
type TaskRepository struct {
	client *Client
}

// NewTaskRepository creates a new task repository
func NewTaskRepository(client *Client) *TaskRepository {
	return &TaskRepository{client: client}
}

// Position constants for task ordering
const (
	DefaultPositionIncrement = 1000.0
)

// Add creates a new task with required plan links and optional relationships.
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
		exists, err := r.planExists(ctx, tx, planID)
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

	metadataJSON := metadataToJSON(task.Metadata)
	tagsList := tagsToCypherList(task.Tags)

	cypher := fmt.Sprintf(
		`CREATE (t:Task {
			id: '%s',
			node_type: 'Task',
			content: '%s',
			status: '%s',
			metadata: '%s',
			tags: %s,
			created_at: '%s',
			updated_at: '%s'
		}) RETURN t`,
		EscapeCypherString(task.ID),
		EscapeCypherString(task.Content),
		EscapeCypherString(string(task.Status)),
		EscapeCypherString(metadataJSON),
		tagsList,
		task.CreatedAt.Format(time.RFC3339),
		task.UpdatedAt.Format(time.RFC3339),
	)

	rows, err := r.client.execCypher(ctx, tx, cypher, "t agtype")
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}
	rows.Close()

	// Create PART_OF relationships to plans
	for _, planID := range planIDs {
		position, err := r.calculateNewTaskPosition(ctx, tx, planID, afterTaskID, beforeTaskID)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate position for plan %s: %w", planID, err)
		}

		if err := r.createTaskToPlanRelationship(ctx, tx, task.ID, planID, position); err != nil {
			return nil, fmt.Errorf("failed to link task to plan %s: %w", planID, err)
		}
	}

	// Create other relationships
	for _, rel := range relationships {
		if err := r.createRelationshipFromTask(ctx, tx, task.ID, rel.ToID, rel.Type); err != nil {
			return nil, fmt.Errorf("failed to create %s relationship to %s: %w", rel.Type, rel.ToID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return &task, nil
}

// GetByID retrieves a task by ID
func (r *TaskRepository) GetByID(ctx context.Context, id string) (*models.Task, error) {
	cypher := fmt.Sprintf(`MATCH (t:Task {id: '%s'}) RETURN t`, EscapeCypherString(id))

	rows, err := r.client.execCypher(ctx, nil, cypher, "t agtype")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	var agtypeStr string
	if err := rows.Scan(&agtypeStr); err != nil {
		return nil, err
	}

	props, err := parseAGTypeProperties(agtypeStr)
	if err != nil {
		return nil, err
	}

	task := propsToTask(props)
	return &task, nil
}

// GetWithPlans retrieves a task by ID along with its associated plans
func (r *TaskRepository) GetWithPlans(ctx context.Context, id string) (*models.Task, []models.Plan, error) {
	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	// Get task
	taskCypher := fmt.Sprintf(`MATCH (t:Task {id: '%s'}) RETURN t`, EscapeCypherString(id))
	taskRows, err := r.client.execCypher(ctx, tx, taskCypher, "t agtype")
	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %w", err)
	}

	var task *models.Task
	if taskRows.Next() {
		var agtypeStr string
		if err := taskRows.Scan(&agtypeStr); err == nil {
			props, err := parseAGTypeProperties(agtypeStr)
			if err == nil {
				t := propsToTask(props)
				task = &t
			}
		}
	}
	taskRows.Close()

	if task == nil {
		return nil, nil, nil
	}

	// Get plans
	plansCypher := fmt.Sprintf(
		`MATCH (t:Task {id: '%s'})-[:PART_OF]->(p:Plan)
		 RETURN p`,
		EscapeCypherString(id))

	plansRows, err := r.client.execCypher(ctx, tx, plansCypher, "p agtype")
	if err != nil {
		return nil, nil, fmt.Errorf("plans query failed: %w", err)
	}

	var plans []models.Plan
	for plansRows.Next() {
		var agtypeStr string
		if err := plansRows.Scan(&agtypeStr); err != nil {
			continue
		}
		props, err := parseAGTypeProperties(agtypeStr)
		if err != nil {
			continue
		}
		plans = append(plans, propsToPlan(props))
	}
	plansRows.Close()

	tx.Commit()
	return task, plans, nil
}

// Update modifies an existing task
func (r *TaskRepository) Update(ctx context.Context, id string, content *string, status *string, metadata map[string]string, tags []string, addPlanIDs []string, newRelationships []models.Relationship) (*models.Task, error) {
	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Verify all plans exist
	for _, planID := range addPlanIDs {
		exists, err := r.planExists(ctx, tx, planID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify plan %s: %w", planID, err)
		}
		if !exists {
			return nil, fmt.Errorf("plan not found: %s", planID)
		}
	}

	// Build dynamic SET clause
	setClauses := []string{fmt.Sprintf("t.updated_at = '%s'", time.Now().UTC().Format(time.RFC3339))}

	if content != nil {
		setClauses = append(setClauses, fmt.Sprintf("t.content = '%s'", EscapeCypherString(*content)))
	}
	if status != nil {
		setClauses = append(setClauses, fmt.Sprintf("t.status = '%s'", EscapeCypherString(*status)))
	}
	if metadata != nil {
		setClauses = append(setClauses, fmt.Sprintf("t.metadata = '%s'", EscapeCypherString(metadataToJSON(metadata))))
	}
	if tags != nil {
		setClauses = append(setClauses, fmt.Sprintf("t.tags = %s", tagsToCypherList(tags)))
	}

	cypher := fmt.Sprintf(`
		MATCH (t:Task {id: '%s'})
		SET %s
		RETURN t`,
		EscapeCypherString(id),
		joinStrings(setClauses, ", "))

	rows, err := r.client.execCypher(ctx, tx, cypher, "t agtype")
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	var task *models.Task
	if rows.Next() {
		var agtypeStr string
		if err := rows.Scan(&agtypeStr); err == nil {
			props, err := parseAGTypeProperties(agtypeStr)
			if err == nil {
				t := propsToTask(props)
				task = &t
			}
		}
	}
	rows.Close()

	if task == nil {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	// Add to new plans (append to end)
	for _, planID := range addPlanIDs {
		maxPos, err := r.getMaxPosition(ctx, tx, planID)
		if err != nil {
			return nil, fmt.Errorf("failed to get max position for plan %s: %w", planID, err)
		}
		position := appendPosition(maxPos)

		if err := r.createTaskToPlanRelationship(ctx, tx, id, planID, position); err != nil {
			return nil, fmt.Errorf("failed to link task to plan %s: %w", planID, err)
		}
	}

	// Create new relationships
	for _, rel := range newRelationships {
		if err := r.createRelationshipFromTask(ctx, tx, id, rel.ToID, rel.Type); err != nil {
			return nil, fmt.Errorf("failed to create %s relationship to %s: %w", rel.Type, rel.ToID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return task, nil
}

// Delete removes a task and all its relationships
func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	cypher := fmt.Sprintf(`MATCH (t:Task {id: '%s'}) DETACH DELETE t RETURN true`, EscapeCypherString(id))

	rows, err := r.client.execCypher(ctx, nil, cypher, "result agtype")
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}
	rows.Close()

	return nil
}

// UpdatePositions batch updates task positions within a plan.
func (r *TaskRepository) UpdatePositions(ctx context.Context, planID string, taskPositions map[string]float64) error {
	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for taskID, position := range taskPositions {
		cypher := fmt.Sprintf(
			`MATCH (t:Task {id: '%s'})-[r:PART_OF]->(p:Plan {id: '%s'})
			 SET r.position = %f
			 RETURN r`,
			EscapeCypherString(taskID),
			EscapeCypherString(planID),
			position)

		rows, err := r.client.execCypher(ctx, tx, cypher, "r agtype")
		if err != nil {
			return fmt.Errorf("failed to update position for task %s: %w", taskID, err)
		}
		if !rows.Next() {
			rows.Close()
			return fmt.Errorf("task %s not found in plan %s", taskID, planID)
		}
		rows.Close()
	}

	return tx.Commit()
}

// List retrieves tasks with optional filtering.
func (r *TaskRepository) List(ctx context.Context, planID string, status string, tags []string, limit int) ([]models.TaskListResult, error) {
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
			tagChecks := make([]string, len(tags))
			for i, tag := range tags {
				tagChecks[i] = fmt.Sprintf("'%s' IN t.tags", EscapeCypherString(tag))
			}
			whereClauses = append(whereClauses, "("+joinStrings(tagChecks, " OR ")+")")
		}

		whereClause := ""
		if len(whereClauses) > 0 {
			whereClause = "AND " + joinStrings(whereClauses, " AND ")
		}

		cypher = fmt.Sprintf(`
			MATCH (t:Task)-[r:PART_OF]->(p:Plan {id: '%s'})
			WHERE true %s
			RETURN t, r.position
			ORDER BY r.position ASC
			LIMIT %d`,
			EscapeCypherString(planID), whereClause, limit)
	} else {
		whereClauses := []string{}
		if status != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("t.status = '%s'", EscapeCypherString(status)))
		}
		if len(tags) > 0 {
			tagChecks := make([]string, len(tags))
			for i, tag := range tags {
				tagChecks[i] = fmt.Sprintf("'%s' IN t.tags", EscapeCypherString(tag))
			}
			whereClauses = append(whereClauses, "("+joinStrings(tagChecks, " OR ")+")")
		}

		whereClause := ""
		if len(whereClauses) > 0 {
			whereClause = "WHERE " + joinStrings(whereClauses, " AND ")
		}

		cypher = fmt.Sprintf(`
			MATCH (t:Task)
			%s
			RETURN t, null
			ORDER BY t.updated_at DESC
			LIMIT %d`,
			whereClause, limit)
	}

	rows, err := r.client.execCypher(ctx, nil, cypher, "t agtype, position agtype")
	if err != nil {
		return nil, fmt.Errorf("list failed: %w", err)
	}
	defer rows.Close()

	var tasks []models.TaskListResult
	for rows.Next() {
		var taskStr, posStr string
		if err := rows.Scan(&taskStr, &posStr); err != nil {
			continue
		}

		props, err := parseAGTypeProperties(taskStr)
		if err != nil {
			continue
		}

		task := propsToTask(props)
		taskResult := models.TaskListResult{Task: task}

		if hasPosition && posStr != "null" && posStr != "" {
			posFloat := parseAGTypeFloat(posStr)
			taskResult.Position = &posFloat
		}

		tasks = append(tasks, taskResult)
	}

	return tasks, nil
}

// Helper methods

func (r *TaskRepository) planExists(ctx context.Context, tx *sql.Tx, planID string) (bool, error) {
	cypher := fmt.Sprintf(`MATCH (p:Plan {id: '%s'}) RETURN count(p) > 0`, EscapeCypherString(planID))
	rows, err := r.client.execCypher(ctx, tx, cypher, "exists agtype")
	if err != nil {
		return false, err
	}
	defer rows.Close()

	if !rows.Next() {
		return false, nil
	}

	var existsStr string
	if err := rows.Scan(&existsStr); err != nil {
		return false, err
	}
	return strings.Trim(existsStr, "\"") == "true", nil
}

func (r *TaskRepository) createTaskToPlanRelationship(ctx context.Context, tx *sql.Tx, taskID, planID string, position float64) error {
	// Check if relationship exists
	checkCypher := fmt.Sprintf(
		`MATCH (t:Task {id: '%s'})-[r:PART_OF]->(p:Plan {id: '%s'})
		 RETURN r`,
		EscapeCypherString(taskID),
		EscapeCypherString(planID))

	checkRows, err := r.client.execCypher(ctx, tx, checkCypher, "r agtype")
	if err != nil {
		return err
	}

	exists := checkRows.Next()
	checkRows.Close()

	if exists {
		// Update position
		updateCypher := fmt.Sprintf(
			`MATCH (t:Task {id: '%s'})-[r:PART_OF]->(p:Plan {id: '%s'})
			 SET r.position = %f
			 RETURN r`,
			EscapeCypherString(taskID),
			EscapeCypherString(planID),
			position)
		updateRows, err := r.client.execCypher(ctx, tx, updateCypher, "r agtype")
		if err != nil {
			return err
		}
		updateRows.Close()
		return nil
	}

	// Create relationship with position
	createCypher := fmt.Sprintf(
		`MATCH (t:Task {id: '%s'}), (p:Plan {id: '%s'})
		 CREATE (t)-[r:PART_OF {position: %f}]->(p)
		 RETURN r`,
		EscapeCypherString(taskID),
		EscapeCypherString(planID),
		position)

	createRows, err := r.client.execCypher(ctx, tx, createCypher, "r agtype")
	if err != nil {
		return err
	}
	createRows.Close()

	return nil
}

func (r *TaskRepository) createRelationshipFromTask(ctx context.Context, tx *sql.Tx, fromID, toID string, relType models.RelationType) error {
	if err := ValidateRelationType(relType); err != nil {
		return err
	}

	// Check if exists
	checkCypher := fmt.Sprintf(
		`MATCH (a:Task {id: '%s'})-[r:%s]->(b)
		 WHERE b.id = '%s'
		 RETURN r`,
		EscapeCypherString(fromID),
		relType,
		EscapeCypherString(toID))

	rows, err := r.client.execCypher(ctx, tx, checkCypher, "r agtype")
	if err != nil {
		return err
	}
	exists := rows.Next()
	rows.Close()

	if exists {
		return nil
	}

	// Create - use label() function for AGE-compatible label filtering
	createCypher := fmt.Sprintf(
		`MATCH (a:Task {id: '%s'}), (b)
		 WHERE b.id = '%s' AND %s
		 CREATE (a)-[r:%s]->(b)
		 RETURN r`,
		EscapeCypherString(fromID),
		EscapeCypherString(toID),
		NodeLabelPredicate("b"),
		relType)

	createRows, err := r.client.execCypher(ctx, tx, createCypher, "r agtype")
	if err != nil {
		return err
	}
	createRows.Close()

	return nil
}

func (r *TaskRepository) getMaxPosition(ctx context.Context, tx *sql.Tx, planID string) (float64, error) {
	cypher := fmt.Sprintf(
		`MATCH (t:Task)-[r:PART_OF]->(p:Plan {id: '%s'})
		 RETURN max(r.position)`,
		EscapeCypherString(planID))

	rows, err := r.client.execCypher(ctx, tx, cypher, "max_pos agtype")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, nil
	}

	var maxPosStr sql.NullString
	if err := rows.Scan(&maxPosStr); err != nil {
		return 0, err
	}

	if !maxPosStr.Valid || maxPosStr.String == "null" || maxPosStr.String == "" {
		return 0, nil
	}

	return parseAGTypeFloat(maxPosStr.String), nil
}

func (r *TaskRepository) getTaskPosition(ctx context.Context, tx *sql.Tx, taskID, planID string) (float64, error) {
	cypher := fmt.Sprintf(
		`MATCH (t:Task {id: '%s'})-[r:PART_OF]->(p:Plan {id: '%s'})
		 RETURN r.position`,
		EscapeCypherString(taskID),
		EscapeCypherString(planID))

	rows, err := r.client.execCypher(ctx, tx, cypher, "position agtype")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, nil
	}

	var posStr string
	if err := rows.Scan(&posStr); err != nil {
		return 0, err
	}

	if posStr == "null" || posStr == "" {
		return 0, nil
	}

	return parseAGTypeFloat(posStr), nil
}

func (r *TaskRepository) getAdjacentPositions(ctx context.Context, tx *sql.Tx, taskID, planID string) (before, after float64, err error) {
	currentPos, err := r.getTaskPosition(ctx, tx, taskID, planID)
	if err != nil {
		return 0, 0, err
	}

	// Get before position
	beforeCypher := fmt.Sprintf(
		`MATCH (t:Task)-[r:PART_OF]->(p:Plan {id: '%s'})
		 WHERE r.position < %f
		 RETURN r.position
		 ORDER BY r.position DESC
		 LIMIT 1`,
		EscapeCypherString(planID),
		currentPos)

	beforeRows, err := r.client.execCypher(ctx, tx, beforeCypher, "position agtype")
	if err != nil {
		return 0, 0, err
	}
	if beforeRows.Next() {
		var posStr string
		if err := beforeRows.Scan(&posStr); err == nil {
			before = parseAGTypeFloat(posStr)
		}
	}
	beforeRows.Close()

	// Get after position
	afterCypher := fmt.Sprintf(
		`MATCH (t:Task)-[r:PART_OF]->(p:Plan {id: '%s'})
		 WHERE r.position > %f
		 RETURN r.position
		 ORDER BY r.position ASC
		 LIMIT 1`,
		EscapeCypherString(planID),
		currentPos)

	afterRows, err := r.client.execCypher(ctx, tx, afterCypher, "position agtype")
	if err != nil {
		return 0, 0, err
	}
	if afterRows.Next() {
		var posStr string
		if err := afterRows.Scan(&posStr); err == nil {
			after = parseAGTypeFloat(posStr)
		}
	}
	afterRows.Close()

	return before, after, nil
}

func (r *TaskRepository) calculateNewTaskPosition(ctx context.Context, tx *sql.Tx, planID string, afterTaskID, beforeTaskID *string) (float64, error) {
	var afterPos, beforePos float64

	if afterTaskID != nil && *afterTaskID != "" {
		var err error
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
		var err error
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

func appendPosition(maxPos float64) float64 {
	nanoComponent := float64(time.Now().UnixNano()%1e9) / 1e9
	jitter := rand.Float64() * 0.0001
	return maxPos + DefaultPositionIncrement + nanoComponent + jitter
}

// CalculateInsertPositions calculates position values for inserting tasks.
func CalculateInsertPositions(afterPos, beforePos float64, count int) []float64 {
	if count <= 0 {
		return nil
	}

	positions := make([]float64, count)

	switch {
	case afterPos == 0 && beforePos == 0:
		for i := 0; i < count; i++ {
			positions[i] = DefaultPositionIncrement * float64(i+1)
		}
	case beforePos == 0:
		for i := 0; i < count; i++ {
			positions[i] = afterPos + DefaultPositionIncrement*float64(i+1)
		}
	case afterPos == 0:
		gap := beforePos / float64(count+1)
		for i := 0; i < count; i++ {
			positions[i] = gap * float64(i+1)
		}
	default:
		gap := (beforePos - afterPos) / float64(count+1)
		for i := 0; i < count; i++ {
			positions[i] = afterPos + gap*float64(i+1)
		}
	}

	return positions
}
