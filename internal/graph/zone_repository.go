package graph

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Thomas-Fitz/associate/internal/models"
	"github.com/google/uuid"
)

// ZoneRepository provides CRUD operations for zones.
type ZoneRepository struct {
	client *Client
}

// NewZoneRepository creates a new zone repository
func NewZoneRepository(client *Client) *ZoneRepository {
	return &ZoneRepository{client: client}
}

// Add creates a new zone
func (r *ZoneRepository) Add(ctx context.Context, zone models.Zone) (*models.Zone, error) {
	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	created, err := r.addWithTx(ctx, tx, zone)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return created, nil
}

// addWithTx creates a new zone using an existing transaction.
// This is used internally when zone creation needs to be part of a larger transaction.
func (r *ZoneRepository) addWithTx(ctx context.Context, tx *sql.Tx, zone models.Zone) (*models.Zone, error) {
	if zone.ID == "" {
		zone.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	zone.CreatedAt = now
	zone.UpdatedAt = now

	metadataJSON := metadataToJSON(zone.Metadata)
	tagsList := tagsToCypherList(zone.Tags)

	cypher := fmt.Sprintf(
		`CREATE (z:Zone {
			id: '%s',
			node_type: 'Zone',
			name: '%s',
			description: '%s',
			metadata: '%s',
			tags: %s,
			created_at: '%s',
			updated_at: '%s'
		}) RETURN z`,
		EscapeCypherString(zone.ID),
		EscapeCypherString(zone.Name),
		EscapeCypherString(zone.Description),
		EscapeCypherString(metadataJSON),
		tagsList,
		zone.CreatedAt.Format(time.RFC3339),
		zone.UpdatedAt.Format(time.RFC3339),
	)

	rows, err := r.client.execCypher(ctx, tx, cypher, "z agtype")
	if err != nil {
		return nil, fmt.Errorf("failed to create zone: %w", err)
	}
	rows.Close()

	return &zone, nil
}

// AddWithTx creates a new zone using an existing transaction.
// Use this when you need zone creation to be part of a larger transaction.
func (r *ZoneRepository) AddWithTx(ctx context.Context, tx *sql.Tx, zone models.Zone) (*models.Zone, error) {
	return r.addWithTx(ctx, tx, zone)
}

// ExistsWithTx checks if a zone exists using an existing transaction.
func (r *ZoneRepository) ExistsWithTx(ctx context.Context, tx *sql.Tx, id string) (bool, error) {
	cypher := fmt.Sprintf(`MATCH (z:Zone {id: '%s'}) RETURN count(z) > 0`, EscapeCypherString(id))
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

// GetByID retrieves a zone by ID
func (r *ZoneRepository) GetByID(ctx context.Context, id string) (*models.Zone, error) {
	cypher := fmt.Sprintf(`MATCH (z:Zone {id: '%s'}) RETURN z`, EscapeCypherString(id))

	rows, err := r.client.execCypher(ctx, nil, cypher, "z agtype")
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

	zone := propsToZone(props)
	return &zone, nil
}

// GetWithContents retrieves a zone by ID along with all its plans, tasks, and memories.
func (r *ZoneRepository) GetWithContents(ctx context.Context, id string) (*models.ZoneWithContents, error) {
	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Get the zone
	zoneCypher := fmt.Sprintf(`MATCH (z:Zone {id: '%s'}) RETURN z`, EscapeCypherString(id))
	zoneRows, err := r.client.execCypher(ctx, tx, zoneCypher, "z agtype")
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	var zone *models.Zone
	if zoneRows.Next() {
		var agtypeStr string
		if err := zoneRows.Scan(&agtypeStr); err == nil {
			props, err := parseAGTypeProperties(agtypeStr)
			if err == nil {
				z := propsToZone(props)
				zone = &z
			}
		}
	}
	zoneRows.Close()

	if zone == nil {
		return nil, nil // Not found
	}

	// Get plans belonging to this zone
	plansCypher := fmt.Sprintf(
		`MATCH (p:Plan)-[:BELONGS_TO]->(z:Zone {id: '%s'})
		 RETURN p
		 ORDER BY p.updated_at DESC`,
		EscapeCypherString(id))

	plansRows, err := r.client.execCypher(ctx, tx, plansCypher, "p agtype")
	if err != nil {
		return nil, fmt.Errorf("plans query failed: %w", err)
	}

	var plansInZone []models.PlanInZone
	for plansRows.Next() {
		var agtypeStr string
		if err := plansRows.Scan(&agtypeStr); err != nil {
			continue
		}
		props, err := parseAGTypeProperties(agtypeStr)
		if err != nil {
			continue
		}
		plan := propsToPlan(props)
		plansInZone = append(plansInZone, models.PlanInZone{
			Plan:  plan,
			Tasks: []models.TaskInPlan{},
		})
	}
	plansRows.Close()

	// Get tasks for each plan
	for i, piz := range plansInZone {
		tasksCypher := fmt.Sprintf(
			`MATCH (t:Task)-[r:PART_OF]->(p:Plan {id: '%s'})
			 RETURN t, r.position
			 ORDER BY r.position ASC`,
			EscapeCypherString(piz.Plan.ID))

		tasksRows, err := r.client.execCypher(ctx, tx, tasksCypher, "t agtype, position agtype")
		if err != nil {
			continue
		}

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

			plansInZone[i].Tasks = append(plansInZone[i].Tasks, models.TaskInPlan{
				Task:     task,
				Position: position,
			})
		}
		tasksRows.Close()

		// Get dependencies and blocks for each task
		for j, tip := range plansInZone[i].Tasks {
			// Get DEPENDS_ON relationships
			depCypher := fmt.Sprintf(
				`MATCH (t:Task {id: '%s'})-[:DEPENDS_ON]->(dep:Task)-[:PART_OF]->(p:Plan {id: '%s'})
				 RETURN dep.id`,
				EscapeCypherString(tip.Task.ID),
				EscapeCypherString(piz.Plan.ID))
			depRows, err := r.client.execCypher(ctx, tx, depCypher, "dep_id agtype")
			if err == nil {
				for depRows.Next() {
					var depID string
					if err := depRows.Scan(&depID); err == nil {
						depID = strings.Trim(depID, "\"")
						if depID != "" {
							plansInZone[i].Tasks[j].DependsOn = append(plansInZone[i].Tasks[j].DependsOn, depID)
						}
					}
				}
				depRows.Close()
			}

			// Get BLOCKS relationships
			blkCypher := fmt.Sprintf(
				`MATCH (t:Task {id: '%s'})-[:BLOCKS]->(blk:Task)-[:PART_OF]->(p:Plan {id: '%s'})
				 RETURN blk.id`,
				EscapeCypherString(tip.Task.ID),
				EscapeCypherString(piz.Plan.ID))
			blkRows, err := r.client.execCypher(ctx, tx, blkCypher, "blk_id agtype")
			if err == nil {
				for blkRows.Next() {
					var blkID string
					if err := blkRows.Scan(&blkID); err == nil {
						blkID = strings.Trim(blkID, "\"")
						if blkID != "" {
							plansInZone[i].Tasks[j].Blocks = append(plansInZone[i].Tasks[j].Blocks, blkID)
						}
					}
				}
				blkRows.Close()
			}
		}
	}

	// Get memories belonging to this zone
	memoriesCypher := fmt.Sprintf(
		`MATCH (m:Memory)-[:BELONGS_TO]->(z:Zone {id: '%s'})
		 RETURN m
		 ORDER BY m.updated_at DESC`,
		EscapeCypherString(id))

	memoriesRows, err := r.client.execCypher(ctx, tx, memoriesCypher, "m agtype")
	if err != nil {
		return nil, fmt.Errorf("memories query failed: %w", err)
	}

	var memoriesInZone []models.MemoryInZone
	for memoriesRows.Next() {
		var agtypeStr string
		if err := memoriesRows.Scan(&agtypeStr); err != nil {
			continue
		}
		props, err := parseAGTypeProperties(agtypeStr)
		if err != nil {
			continue
		}
		mem := propsToMemory(props)
		memoriesInZone = append(memoriesInZone, models.MemoryInZone{Memory: mem})
	}
	memoriesRows.Close()

	tx.Commit()
	return &models.ZoneWithContents{
		Zone:     *zone,
		Plans:    plansInZone,
		Memories: memoriesInZone,
	}, nil
}

// Update modifies an existing zone
func (r *ZoneRepository) Update(ctx context.Context, id string, name *string, description *string, metadata map[string]string, tags []string) (*models.Zone, error) {
	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Build dynamic SET clause
	setClauses := []string{fmt.Sprintf("z.updated_at = '%s'", time.Now().UTC().Format(time.RFC3339))}

	if name != nil {
		setClauses = append(setClauses, fmt.Sprintf("z.name = '%s'", EscapeCypherString(*name)))
	}
	if description != nil {
		setClauses = append(setClauses, fmt.Sprintf("z.description = '%s'", EscapeCypherString(*description)))
	}
	if metadata != nil {
		setClauses = append(setClauses, fmt.Sprintf("z.metadata = '%s'", EscapeCypherString(metadataToJSON(metadata))))
	}
	if tags != nil {
		setClauses = append(setClauses, fmt.Sprintf("z.tags = %s", tagsToCypherList(tags)))
	}

	cypher := fmt.Sprintf(`
		MATCH (z:Zone {id: '%s'})
		SET %s
		RETURN z`,
		EscapeCypherString(id),
		joinStrings(setClauses, ", "))

	rows, err := r.client.execCypher(ctx, tx, cypher, "z agtype")
	if err != nil {
		return nil, fmt.Errorf("failed to update zone: %w", err)
	}

	var zone *models.Zone
	if rows.Next() {
		var agtypeStr string
		if err := rows.Scan(&agtypeStr); err == nil {
			props, err := parseAGTypeProperties(agtypeStr)
			if err == nil {
				z := propsToZone(props)
				zone = &z
			}
		}
	}
	rows.Close()

	if zone == nil {
		return nil, fmt.Errorf("zone not found: %s", id)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return zone, nil
}

// Delete removes a zone and all its contents (plans, tasks, memories).
// Returns the number of plans and tasks that were deleted.
func (r *ZoneRepository) Delete(ctx context.Context, id string) (plansDeleted, tasksDeleted, memoriesDeleted int, err error) {
	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	defer tx.Rollback()

	// Step 1: Get all plans that belong to this zone
	plansCypher := fmt.Sprintf(
		`MATCH (p:Plan)-[:BELONGS_TO]->(z:Zone {id: '%s'})
		 RETURN p.id`,
		EscapeCypherString(id))

	plansRows, err := r.client.execCypher(ctx, tx, plansCypher, "plan_id agtype")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get plans: %w", err)
	}

	var planIDs []string
	for plansRows.Next() {
		var planID string
		if err := plansRows.Scan(&planID); err == nil {
			planID = strings.Trim(planID, "\"")
			if planID != "" {
				planIDs = append(planIDs, planID)
			}
		}
	}
	plansRows.Close()

	// Step 2: For each plan, delete tasks that only belong to this plan
	for _, planID := range planIDs {
		// Get tasks
		tasksCypher := fmt.Sprintf(
			`MATCH (t:Task)-[:PART_OF]->(p:Plan {id: '%s'})
			 RETURN t.id`,
			EscapeCypherString(planID))

		tasksRows, err := r.client.execCypher(ctx, tx, tasksCypher, "task_id agtype")
		if err != nil {
			continue
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

		// For each task, check if it belongs to other plans
		for _, taskID := range taskIDs {
			otherPlanCypher := fmt.Sprintf(
				`MATCH (t:Task {id: '%s'})-[:PART_OF]->(other:Plan)
				 WHERE other.id <> '%s'
				 RETURN count(other) > 0`,
				EscapeCypherString(taskID),
				EscapeCypherString(planID))

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
					tasksDeleted++
				}
			}
		}

		// Delete the plan
		planDeleteCypher := fmt.Sprintf(
			`MATCH (p:Plan {id: '%s'}) DETACH DELETE p RETURN true`,
			EscapeCypherString(planID))

		planRows, err := r.client.execCypher(ctx, tx, planDeleteCypher, "result agtype")
		if err == nil {
			planRows.Close()
			plansDeleted++
		}
	}

	// Step 3: Delete memories that belong only to this zone
	memoriesCypher := fmt.Sprintf(
		`MATCH (m:Memory)-[:BELONGS_TO]->(z:Zone {id: '%s'})
		 RETURN m.id`,
		EscapeCypherString(id))

	memoriesRows, err := r.client.execCypher(ctx, tx, memoriesCypher, "memory_id agtype")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get memories: %w", err)
	}

	var memoryIDs []string
	for memoriesRows.Next() {
		var memoryID string
		if err := memoriesRows.Scan(&memoryID); err == nil {
			memoryID = strings.Trim(memoryID, "\"")
			if memoryID != "" {
				memoryIDs = append(memoryIDs, memoryID)
			}
		}
	}
	memoriesRows.Close()

	// Delete each memory
	for _, memoryID := range memoryIDs {
		memDeleteCypher := fmt.Sprintf(
			`MATCH (m:Memory {id: '%s'}) DETACH DELETE m RETURN true`,
			EscapeCypherString(memoryID))

		memRows, err := r.client.execCypher(ctx, tx, memDeleteCypher, "result agtype")
		if err == nil {
			memRows.Close()
			memoriesDeleted++
		}
	}

	// Step 4: Delete the zone itself
	zoneDeleteCypher := fmt.Sprintf(
		`MATCH (z:Zone {id: '%s'}) DETACH DELETE z RETURN true`,
		EscapeCypherString(id))

	zoneRows, err := r.client.execCypher(ctx, tx, zoneDeleteCypher, "result agtype")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("delete failed: %w", err)
	}
	zoneRows.Close()

	if err := tx.Commit(); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to commit: %w", err)
	}

	return plansDeleted, tasksDeleted, memoriesDeleted, nil
}

// List retrieves zones with optional filtering
func (r *ZoneRepository) List(ctx context.Context, search string, limit int) ([]models.ZoneWithCounts, error) {
	if limit <= 0 {
		limit = 50
	}

	tx, err := r.client.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Build WHERE clause for search
	whereClause := ""
	if search != "" {
		escapedSearch := EscapeCypherString(strings.ToLower(search))
		whereClause = fmt.Sprintf("WHERE toLower(z.name) CONTAINS '%s' OR toLower(z.description) CONTAINS '%s'", escapedSearch, escapedSearch)
	}

	// Get zones
	cypher := fmt.Sprintf(`
		MATCH (z:Zone)
		%s
		RETURN z
		ORDER BY z.updated_at DESC
		LIMIT %d`,
		whereClause, limit)

	rows, err := r.client.execCypher(ctx, tx, cypher, "z agtype")
	if err != nil {
		return nil, fmt.Errorf("list failed: %w", err)
	}

	var zones []models.ZoneWithCounts
	for rows.Next() {
		var agtypeStr string
		if err := rows.Scan(&agtypeStr); err != nil {
			continue
		}

		props, err := parseAGTypeProperties(agtypeStr)
		if err != nil {
			continue
		}

		zone := propsToZone(props)
		zones = append(zones, models.ZoneWithCounts{Zone: zone})
	}
	rows.Close()

	// Get counts for each zone
	for i, zwc := range zones {
		// Plan count
		planCountCypher := fmt.Sprintf(
			`MATCH (p:Plan)-[:BELONGS_TO]->(z:Zone {id: '%s'}) RETURN count(p)`,
			EscapeCypherString(zwc.Zone.ID))
		planRows, err := r.client.execCypher(ctx, tx, planCountCypher, "count agtype")
		if err == nil {
			if planRows.Next() {
				var countStr string
				if err := planRows.Scan(&countStr); err == nil {
					countStr = strings.Trim(countStr, "\"")
					if count, err := strconv.Atoi(countStr); err == nil {
						zones[i].PlanCount = count
					}
				}
			}
			planRows.Close()
		}

		// Task count (tasks in plans that belong to this zone)
		taskCountCypher := fmt.Sprintf(
			`MATCH (t:Task)-[:PART_OF]->(p:Plan)-[:BELONGS_TO]->(z:Zone {id: '%s'}) RETURN count(t)`,
			EscapeCypherString(zwc.Zone.ID))
		taskRows, err := r.client.execCypher(ctx, tx, taskCountCypher, "count agtype")
		if err == nil {
			if taskRows.Next() {
				var countStr string
				if err := taskRows.Scan(&countStr); err == nil {
					countStr = strings.Trim(countStr, "\"")
					if count, err := strconv.Atoi(countStr); err == nil {
						zones[i].TaskCount = count
					}
				}
			}
			taskRows.Close()
		}

		// Memory count
		memoryCountCypher := fmt.Sprintf(
			`MATCH (m:Memory)-[:BELONGS_TO]->(z:Zone {id: '%s'}) RETURN count(m)`,
			EscapeCypherString(zwc.Zone.ID))
		memoryRows, err := r.client.execCypher(ctx, tx, memoryCountCypher, "count agtype")
		if err == nil {
			if memoryRows.Next() {
				var countStr string
				if err := memoryRows.Scan(&countStr); err == nil {
					countStr = strings.Trim(countStr, "\"")
					if count, err := strconv.Atoi(countStr); err == nil {
						zones[i].MemoryCount = count
					}
				}
			}
			memoryRows.Close()
		}
	}

	_ = tx.Commit()
	return zones, nil
}

// Exists checks if a zone with the given ID exists
func (r *ZoneRepository) Exists(ctx context.Context, id string) (bool, error) {
	cypher := fmt.Sprintf(`MATCH (z:Zone {id: '%s'}) RETURN count(z) > 0`, EscapeCypherString(id))
	rows, err := r.client.execCypher(ctx, nil, cypher, "exists agtype")
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
