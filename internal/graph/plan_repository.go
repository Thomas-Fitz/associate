package graph

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/fitz/associate/internal/models"
	"github.com/google/uuid"
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

	if plan.Status == "" {
		plan.Status = models.PlanStatusActive
	}

	metadataJSON := metadataToJSON(plan.Metadata)
	tagsList := tagsToCypherList(plan.Tags)

	cypher := fmt.Sprintf(
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
		}) RETURN p`,
		EscapeCypherString(plan.ID),
		EscapeCypherString(plan.Name),
		EscapeCypherString(plan.Description),
		EscapeCypherString(string(plan.Status)),
		EscapeCypherString(metadataJSON),
		tagsList,
		plan.CreatedAt.Format(time.RFC3339),
		plan.UpdatedAt.Format(time.RFC3339),
	)

	rows, err := r.client.execCypher(ctx, tx, cypher, "p agtype")
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}
	rows.Close()

	// Create relationships
	for _, rel := range relationships {
		if err := r.createRelationshipFromPlan(ctx, tx, plan.ID, rel.ToID, rel.Type); err != nil {
			fmt.Printf("warning: failed to create relationship: %v\n", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return &plan, nil
}

// GetByID retrieves a plan by ID
func (r *PlanRepository) GetByID(ctx context.Context, id string) (*models.Plan, error) {
	cypher := fmt.Sprintf(`MATCH (p:Plan {id: '%s'}) RETURN p`, EscapeCypherString(id))

	rows, err := r.client.execCypher(ctx, nil, cypher, "p agtype")
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

	plan := propsToPlan(props)
	return &plan, nil
}

// GetWithTasks retrieves a plan by ID along with all its tasks ordered by position.
func (r *PlanRepository) GetWithTasks(ctx context.Context, id string) (*models.Plan, []models.TaskInPlan, error) {
	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	// First get the plan
	planCypher := fmt.Sprintf(`MATCH (p:Plan {id: '%s'}) RETURN p`, EscapeCypherString(id))
	planRows, err := r.client.execCypher(ctx, tx, planCypher, "p agtype")
	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %w", err)
	}

	var plan *models.Plan
	if planRows.Next() {
		var agtypeStr string
		if err := planRows.Scan(&agtypeStr); err == nil {
			props, err := parseAGTypeProperties(agtypeStr)
			if err == nil {
				p := propsToPlan(props)
				plan = &p
			}
		}
	}
	planRows.Close()

	if plan == nil {
		return nil, nil, nil // Not found
	}

	// Get tasks with position
	tasksCypher := fmt.Sprintf(
		`MATCH (t:Task)-[r:PART_OF]->(p:Plan {id: '%s'})
		 RETURN t, r.position
		 ORDER BY r.position ASC`,
		EscapeCypherString(id))

	tasksRows, err := r.client.execCypher(ctx, tx, tasksCypher, "t agtype, position agtype")
	if err != nil {
		return nil, nil, fmt.Errorf("tasks query failed: %w", err)
	}

	var tasks []models.TaskInPlan
	taskIDs := []string{}
	for tasksRows.Next() {
		var taskStr, posStr string
		if err := tasksRows.Scan(&taskStr, &posStr); err != nil {
			continue
		}

		props, err := parseAGTypeProperties(taskStr)
		if err != nil {
			continue
		}

		task := propsToTask(props)
		position := parseAGTypeFloat(posStr)

		taskIDs = append(taskIDs, task.ID)
		tasks = append(tasks, models.TaskInPlan{
			Task:     task,
			Position: position,
		})
	}
	tasksRows.Close()

	// Get dependencies and blocks for each task
	for i, taskInPlan := range tasks {
		// Get DEPENDS_ON relationships
		depCypher := fmt.Sprintf(
			`MATCH (t:Task {id: '%s'})-[:DEPENDS_ON]->(dep:Task)-[:PART_OF]->(p:Plan {id: '%s'})
			 RETURN dep.id`,
			EscapeCypherString(taskInPlan.Task.ID),
			EscapeCypherString(id))
		depRows, err := r.client.execCypher(ctx, tx, depCypher, "dep_id agtype")
		if err == nil {
			for depRows.Next() {
				var depID string
				if err := depRows.Scan(&depID); err == nil {
					depID = strings.Trim(depID, "\"")
					if depID != "" {
						tasks[i].DependsOn = append(tasks[i].DependsOn, depID)
					}
				}
			}
			depRows.Close()
		}

		// Get BLOCKS relationships
		blkCypher := fmt.Sprintf(
			`MATCH (t:Task {id: '%s'})-[:BLOCKS]->(blk:Task)-[:PART_OF]->(p:Plan {id: '%s'})
			 RETURN blk.id`,
			EscapeCypherString(taskInPlan.Task.ID),
			EscapeCypherString(id))
		blkRows, err := r.client.execCypher(ctx, tx, blkCypher, "blk_id agtype")
		if err == nil {
			for blkRows.Next() {
				var blkID string
				if err := blkRows.Scan(&blkID); err == nil {
					blkID = strings.Trim(blkID, "\"")
					if blkID != "" {
						tasks[i].Blocks = append(tasks[i].Blocks, blkID)
					}
				}
			}
			blkRows.Close()
		}
	}

	tx.Commit()
	return plan, tasks, nil
}

// Update modifies an existing plan
func (r *PlanRepository) Update(ctx context.Context, id string, name *string, description *string, status *string, metadata map[string]string, tags []string, newRelationships []models.Relationship) (*models.Plan, error) {
	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Build dynamic SET clause
	setClauses := []string{fmt.Sprintf("p.updated_at = '%s'", time.Now().UTC().Format(time.RFC3339))}

	if name != nil {
		setClauses = append(setClauses, fmt.Sprintf("p.name = '%s'", EscapeCypherString(*name)))
	}
	if description != nil {
		setClauses = append(setClauses, fmt.Sprintf("p.description = '%s'", EscapeCypherString(*description)))
	}
	if status != nil {
		setClauses = append(setClauses, fmt.Sprintf("p.status = '%s'", EscapeCypherString(*status)))
	}
	if metadata != nil {
		setClauses = append(setClauses, fmt.Sprintf("p.metadata = '%s'", EscapeCypherString(metadataToJSON(metadata))))
	}
	if tags != nil {
		setClauses = append(setClauses, fmt.Sprintf("p.tags = %s", tagsToCypherList(tags)))
	}

	cypher := fmt.Sprintf(`
		MATCH (p:Plan {id: '%s'})
		SET %s
		RETURN p`,
		EscapeCypherString(id),
		joinStrings(setClauses, ", "))

	rows, err := r.client.execCypher(ctx, tx, cypher, "p agtype")
	if err != nil {
		return nil, fmt.Errorf("failed to update plan: %w", err)
	}

	var plan *models.Plan
	if rows.Next() {
		var agtypeStr string
		if err := rows.Scan(&agtypeStr); err == nil {
			props, err := parseAGTypeProperties(agtypeStr)
			if err == nil {
				p := propsToPlan(props)
				plan = &p
			}
		}
	}
	rows.Close()

	if plan == nil {
		return nil, fmt.Errorf("plan not found: %s", id)
	}

	// Create new relationships
	for _, rel := range newRelationships {
		if err := r.createRelationshipFromPlan(ctx, tx, id, rel.ToID, rel.Type); err != nil {
			fmt.Printf("warning: failed to create relationship: %v\n", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return plan, nil
}

// Delete removes a plan and cascades to tasks not linked to other plans.
// Uses multi-step Go loop since AGE doesn't support FOREACH/NOT EXISTS patterns.
func (r *PlanRepository) Delete(ctx context.Context, id string) (int, error) {
	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Step 1: Get all tasks that belong to this plan
	tasksCypher := fmt.Sprintf(
		`MATCH (t:Task)-[:PART_OF]->(p:Plan {id: '%s'})
		 RETURN t.id`,
		EscapeCypherString(id))

	tasksRows, err := r.client.execCypher(ctx, tx, tasksCypher, "task_id agtype")
	if err != nil {
		return 0, fmt.Errorf("failed to get tasks: %w", err)
	}

	var taskIDs []string
	for tasksRows.Next() {
		var taskID string
		if err := tasksRows.Scan(&taskID); err == nil {
			taskID = strings.Trim(taskID, "\"")
			if taskID != "" {
				taskIDs = append(taskIDs, taskID)
			}
		}
	}
	tasksRows.Close()

	// Step 2: For each task, check if it belongs to other plans
	deletedCount := 0
	for _, taskID := range taskIDs {
		// Check for other plans
		otherPlanCypher := fmt.Sprintf(
			`MATCH (t:Task {id: '%s'})-[:PART_OF]->(other:Plan)
			 WHERE other.id <> '%s'
			 RETURN count(other) > 0`,
			EscapeCypherString(taskID),
			EscapeCypherString(id))

		otherRows, err := r.client.execCypher(ctx, tx, otherPlanCypher, "has_other agtype")
		if err != nil {
			continue
		}

		hasOther := false
		if otherRows.Next() {
			var hasOtherStr string
			if err := otherRows.Scan(&hasOtherStr); err == nil {
				hasOther = strings.Trim(hasOtherStr, "\"") == "true"
			}
		}
		otherRows.Close()

		// If task has no other plans, delete it
		if !hasOther {
			deleteCypher := fmt.Sprintf(
				`MATCH (t:Task {id: '%s'}) DETACH DELETE t RETURN true`,
				EscapeCypherString(taskID))
			delRows, err := r.client.execCypher(ctx, tx, deleteCypher, "result agtype")
			if err == nil {
				delRows.Close()
				deletedCount++
			}
		}
	}

	// Step 3: Delete the plan itself
	planDeleteCypher := fmt.Sprintf(
		`MATCH (p:Plan {id: '%s'}) DETACH DELETE p RETURN true`,
		EscapeCypherString(id))

	planRows, err := r.client.execCypher(ctx, tx, planDeleteCypher, "result agtype")
	if err != nil {
		return 0, fmt.Errorf("delete failed: %w", err)
	}
	planRows.Close()

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit: %w", err)
	}

	return deletedCount, nil
}

// List retrieves plans with optional filtering
func (r *PlanRepository) List(ctx context.Context, status string, tags []string, limit int) ([]models.Plan, error) {
	if limit <= 0 {
		limit = 50
	}

	// Build WHERE clause
	whereClauses := []string{}
	if status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("p.status = '%s'", EscapeCypherString(status)))
	}
	if len(tags) > 0 {
		// Check if any provided tag is in the plan's tags
		tagChecks := make([]string, len(tags))
		for i, tag := range tags {
			tagChecks[i] = fmt.Sprintf("'%s' IN p.tags", EscapeCypherString(tag))
		}
		whereClauses = append(whereClauses, "("+joinStrings(tagChecks, " OR ")+")")
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
		LIMIT %d`,
		whereClause, limit)

	rows, err := r.client.execCypher(ctx, nil, cypher, "p agtype")
	if err != nil {
		return nil, fmt.Errorf("list failed: %w", err)
	}
	defer rows.Close()

	var plans []models.Plan
	for rows.Next() {
		var agtypeStr string
		if err := rows.Scan(&agtypeStr); err != nil {
			continue
		}

		props, err := parseAGTypeProperties(agtypeStr)
		if err != nil {
			continue
		}

		plans = append(plans, propsToPlan(props))
	}

	return plans, nil
}

// createRelationshipFromPlan creates a relationship from a Plan to another node
func (r *PlanRepository) createRelationshipFromPlan(ctx context.Context, tx *sql.Tx, fromID, toID string, relType models.RelationType) error {
	if err := ValidateRelationType(relType); err != nil {
		return err
	}

	// Check if relationship already exists
	checkCypher := fmt.Sprintf(
		`MATCH (a:Plan {id: '%s'})-[r:%s]->(b)
		 WHERE b.id = '%s'
		 RETURN r`,
		EscapeCypherString(fromID),
		relType,
		EscapeCypherString(toID),
	)

	rows, err := r.client.execCypher(ctx, tx, checkCypher, "r agtype")
	if err != nil {
		return err
	}

	exists := rows.Next()
	rows.Close()

	if exists {
		return nil
	}

	// Create the relationship
	createCypher := fmt.Sprintf(
		`MATCH (a:Plan {id: '%s'}), (b)
		 WHERE b.id = '%s' AND (b:Memory OR b:Plan OR b:Task)
		 CREATE (a)-[r:%s]->(b)
		 RETURN r`,
		EscapeCypherString(fromID),
		EscapeCypherString(toID),
		relType,
	)

	createRows, err := r.client.execCypher(ctx, tx, createCypher, "r agtype")
	if err != nil {
		return err
	}
	createRows.Close()

	return nil
}

// parseAGTypeFloat parses a float from an agtype string
func parseAGTypeFloat(s string) float64 {
	s = strings.Trim(s, "\"")
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
