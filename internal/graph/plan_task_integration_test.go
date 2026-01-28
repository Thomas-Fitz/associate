//go:build integration
// +build integration

package graph

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Thomas-Fitz/associate/internal/models"
)

// Integration test helper to get a connected client
func getTestClient(t *testing.T) (*Client, context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

	cfg := Config{
		Host:     getEnvOrDefault("DB_HOST", "localhost"),
		Port:     getEnvOrDefault("DB_PORT", "5432"),
		Username: getEnvOrDefault("DB_USERNAME", "associate"),
		Password: getEnvOrDefault("DB_PASSWORD", "password"),
		Database: getEnvOrDefault("DB_DATABASE", "associate"),
	}

	client, err := NewClient(ctx, cfg)
	if err != nil {
		cancel()
		t.Fatalf("Failed to connect to PostgreSQL/AGE: %v", err)
	}

	return client, ctx, cancel
}

// cleanupTestData removes test data created during tests
func cleanupTestData(ctx context.Context, client *Client, ids ...string) {
	tx, err := client.BeginTx(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback()

	for _, id := range ids {
		cypher := fmt.Sprintf(`MATCH (n {id: '%s'}) DETACH DELETE n RETURN true`, EscapeCypherString(id))
		rows, err := client.execCypher(ctx, tx, cypher, "result agtype")
		if err == nil {
			rows.Close()
		}
	}
	tx.Commit()
}

// TestPlanRepository_CRUD tests basic Plan CRUD operations
func TestPlanRepository_CRUD(t *testing.T) {
	client, ctx, cancel := getTestClient(t)
	defer cancel()
	defer client.Close(ctx)

	repo := NewPlanRepository(client)
	testID := "test-plan-" + time.Now().Format("20060102-150405-000")
	defer cleanupTestData(ctx, client, testID)

	// Test Create
	t.Run("Create", func(t *testing.T) {
		plan := models.Plan{
			ID:          testID,
			Name:        "Test Plan",
			Description: "A test plan for integration testing",
			Status:      models.PlanStatusActive,
			Tags:        []string{"test", "integration"},
			Metadata:    map[string]string{"priority": "high"},
		}

		created, err := repo.Add(ctx, plan, nil)
		if err != nil {
			t.Fatalf("Failed to create plan: %v", err)
		}

		if created.ID != testID {
			t.Errorf("Expected ID %s, got %s", testID, created.ID)
		}
		if created.Name != "Test Plan" {
			t.Errorf("Expected name 'Test Plan', got '%s'", created.Name)
		}
		if created.Status != models.PlanStatusActive {
			t.Errorf("Expected status 'active', got '%s'", created.Status)
		}
		t.Logf("Created plan: %s", created.ID)
	})

	// Test GetByID
	t.Run("GetByID", func(t *testing.T) {
		plan, err := repo.GetByID(ctx, testID)
		if err != nil {
			t.Fatalf("Failed to get plan: %v", err)
		}
		if plan == nil {
			t.Fatal("Plan not found")
		}

		if plan.Name != "Test Plan" {
			t.Errorf("Expected name 'Test Plan', got '%s'", plan.Name)
		}
		if len(plan.Tags) != 2 {
			t.Errorf("Expected 2 tags, got %d", len(plan.Tags))
		}
		t.Logf("Retrieved plan: %s - %s", plan.ID, plan.Name)
	})

	// Test Update
	t.Run("Update", func(t *testing.T) {
		newName := "Updated Plan"
		newStatus := string(models.PlanStatusCompleted)

		updated, err := repo.Update(ctx, testID, &newName, nil, &newStatus, nil, nil, nil)
		if err != nil {
			t.Fatalf("Failed to update plan: %v", err)
		}

		if updated.Name != newName {
			t.Errorf("Expected name '%s', got '%s'", newName, updated.Name)
		}
		if string(updated.Status) != newStatus {
			t.Errorf("Expected status '%s', got '%s'", newStatus, updated.Status)
		}
		t.Logf("Updated plan: %s - %s (%s)", updated.ID, updated.Name, updated.Status)
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		plans, err := repo.List(ctx, "", nil, 50)
		if err != nil {
			t.Fatalf("Failed to list plans: %v", err)
		}

		found := false
		for _, p := range plans {
			if p.ID == testID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Test plan not found in list")
		}
		t.Logf("Listed %d plans", len(plans))
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		_, err := repo.Delete(ctx, testID)
		if err != nil {
			t.Fatalf("Failed to delete plan: %v", err)
		}

		// Verify deleted
		plan, err := repo.GetByID(ctx, testID)
		if err != nil {
			t.Fatalf("Error checking deleted plan: %v", err)
		}
		if plan != nil {
			t.Error("Plan should be deleted")
		}
		t.Logf("Deleted plan: %s", testID)
	})
}

// TestPlanWithAutoZone tests creating a plan with auto-zone creation
func TestPlanWithAutoZone(t *testing.T) {
	client, ctx, cancel := getTestClient(t)
	defer cancel()
	defer client.Close(ctx)

	planRepo := NewPlanRepository(client)
	zoneRepo := NewZoneRepository(client)

	planName := "Auto Zone Test Plan"
	testID := "test-plan-autozone-" + time.Now().Format("20060102-150405-000")
	var zoneID string
	defer func() {
		cleanupTestData(ctx, client, testID)
		if zoneID != "" {
			cleanupTestData(ctx, client, zoneID)
		}
	}()

	// Test creating plan without zone_id - should auto-create zone
	t.Run("CreatePlanWithAutoZone", func(t *testing.T) {
		plan := models.Plan{
			ID:          testID,
			Name:        planName,
			Description: "A test plan that auto-creates a zone",
			Status:      models.PlanStatusActive,
		}

		created, createdZoneID, err := planRepo.AddWithZone(ctx, plan, "", nil)
		if err != nil {
			t.Fatalf("Failed to create plan with auto-zone: %v", err)
		}

		zoneID = createdZoneID

		if created.ID != testID {
			t.Errorf("Expected plan ID %s, got %s", testID, created.ID)
		}
		if zoneID == "" {
			t.Error("Expected zone ID to be created, got empty string")
		}

		t.Logf("Created plan: %s with auto-created zone: %s", created.ID, zoneID)
	})

	// Verify zone was created with the plan's name
	t.Run("VerifyZoneCreated", func(t *testing.T) {
		zone, err := zoneRepo.GetByID(ctx, zoneID)
		if err != nil {
			t.Fatalf("Failed to get zone: %v", err)
		}
		if zone == nil {
			t.Fatal("Zone not found")
		}

		if zone.Name != planName {
			t.Errorf("Expected zone name '%s', got '%s'", planName, zone.Name)
		}
		t.Logf("Zone created with name: %s", zone.Name)
	})

	// Verify plan is linked to zone
	t.Run("VerifyPlanLinkedToZone", func(t *testing.T) {
		linkedZoneID, err := planRepo.GetZoneID(ctx, testID)
		if err != nil {
			t.Fatalf("Failed to get plan's zone ID: %v", err)
		}

		if linkedZoneID != zoneID {
			t.Errorf("Expected plan linked to zone %s, got %s", zoneID, linkedZoneID)
		}
		t.Logf("Plan is correctly linked to zone: %s", linkedZoneID)
	})
}

// TestPlanWithExistingZone tests creating a plan with an existing zone
func TestPlanWithExistingZone(t *testing.T) {
	client, ctx, cancel := getTestClient(t)
	defer cancel()
	defer client.Close(ctx)

	planRepo := NewPlanRepository(client)
	zoneRepo := NewZoneRepository(client)

	zoneID := "test-zone-existing-" + time.Now().Format("20060102-150405-000")
	planID := "test-plan-existing-zone-" + time.Now().Format("20060102-150405-000")
	defer cleanupTestData(ctx, client, zoneID, planID)

	// Create zone first
	t.Run("CreateZone", func(t *testing.T) {
		zone := models.Zone{
			ID:          zoneID,
			Name:        "Existing Zone",
			Description: "A pre-existing zone",
		}
		created, err := zoneRepo.Add(ctx, zone)
		if err != nil {
			t.Fatalf("Failed to create zone: %v", err)
		}
		t.Logf("Created zone: %s", created.ID)
	})

	// Create plan with existing zone
	t.Run("CreatePlanWithExistingZone", func(t *testing.T) {
		plan := models.Plan{
			ID:          planID,
			Name:        "Plan in Existing Zone",
			Description: "A test plan in a pre-existing zone",
			Status:      models.PlanStatusActive,
		}

		created, linkedZoneID, err := planRepo.AddWithZone(ctx, plan, zoneID, nil)
		if err != nil {
			t.Fatalf("Failed to create plan with existing zone: %v", err)
		}

		if linkedZoneID != zoneID {
			t.Errorf("Expected plan linked to zone %s, got %s", zoneID, linkedZoneID)
		}
		t.Logf("Created plan: %s linked to existing zone: %s", created.ID, linkedZoneID)
	})
}

// TestTaskRepository_CRUD tests basic Task CRUD operations
func TestTaskRepository_CRUD(t *testing.T) {
	client, ctx, cancel := getTestClient(t)
	defer cancel()
	defer client.Close(ctx)

	planRepo := NewPlanRepository(client)
	taskRepo := NewTaskRepository(client)

	planID := "test-plan-tasks-" + time.Now().Format("20060102-150405-000")
	taskID := "test-task-" + time.Now().Format("20060102-150405-000")
	defer cleanupTestData(ctx, client, planID, taskID)

	// Create a plan first
	plan := models.Plan{
		ID:     planID,
		Name:   "Plan for Tasks",
		Status: models.PlanStatusActive,
	}
	_, err := planRepo.Add(ctx, plan, nil)
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}

	// Test Create Task
	t.Run("Create", func(t *testing.T) {
		task := models.Task{
			ID:       taskID,
			Content:  "Test task content",
			Status:   models.TaskStatusPending,
			Tags:     []string{"test"},
			Metadata: map[string]string{"key": "value"},
		}

		created, err := taskRepo.Add(ctx, task, []string{planID}, nil, nil, nil)
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		if created.ID != taskID {
			t.Errorf("Expected ID %s, got %s", taskID, created.ID)
		}
		if created.Content != "Test task content" {
			t.Errorf("Expected content 'Test task content', got '%s'", created.Content)
		}
		t.Logf("Created task: %s", created.ID)
	})

	// Test GetByID
	t.Run("GetByID", func(t *testing.T) {
		task, err := taskRepo.GetByID(ctx, taskID)
		if err != nil {
			t.Fatalf("Failed to get task: %v", err)
		}
		if task == nil {
			t.Fatal("Task not found")
		}

		if task.Content != "Test task content" {
			t.Errorf("Expected content 'Test task content', got '%s'", task.Content)
		}
		t.Logf("Retrieved task: %s - %s", task.ID, task.Content)
	})

	// Test GetWithPlans
	t.Run("GetWithPlans", func(t *testing.T) {
		task, plans, err := taskRepo.GetWithPlans(ctx, taskID)
		if err != nil {
			t.Fatalf("Failed to get task with plans: %v", err)
		}
		if task == nil {
			t.Fatal("Task not found")
		}
		if len(plans) != 1 {
			t.Errorf("Expected 1 plan, got %d", len(plans))
		}
		if len(plans) > 0 && plans[0].ID != planID {
			t.Errorf("Expected plan ID %s, got %s", planID, plans[0].ID)
		}
		t.Logf("Task %s belongs to %d plans", task.ID, len(plans))
	})

	// Test Update
	t.Run("Update", func(t *testing.T) {
		newContent := "Updated task content"
		newStatus := string(models.TaskStatusInProgress)

		updated, err := taskRepo.Update(ctx, taskID, &newContent, &newStatus, nil, nil, nil, nil)
		if err != nil {
			t.Fatalf("Failed to update task: %v", err)
		}

		if updated.Content != newContent {
			t.Errorf("Expected content '%s', got '%s'", newContent, updated.Content)
		}
		if string(updated.Status) != newStatus {
			t.Errorf("Expected status '%s', got '%s'", newStatus, updated.Status)
		}
		t.Logf("Updated task: %s - %s", updated.ID, updated.Status)
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		tasks, err := taskRepo.List(ctx, planID, "", nil, 50)
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}

		found := false
		for _, tr := range tasks {
			if tr.Task.ID == taskID {
				found = true
				if tr.Position == nil {
					t.Error("Expected position to be set for plan-filtered list")
				}
				break
			}
		}
		if !found {
			t.Error("Test task not found in list")
		}
		t.Logf("Listed %d tasks in plan", len(tasks))
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		err := taskRepo.Delete(ctx, taskID)
		if err != nil {
			t.Fatalf("Failed to delete task: %v", err)
		}

		// Verify deleted
		task, err := taskRepo.GetByID(ctx, taskID)
		if err != nil {
			t.Fatalf("Error checking deleted task: %v", err)
		}
		if task != nil {
			t.Error("Task should be deleted")
		}
		t.Logf("Deleted task: %s", taskID)
	})
}

// TestMemoryRepository_CRUD tests basic Memory CRUD operations
func TestMemoryRepository_CRUD(t *testing.T) {
	client, ctx, cancel := getTestClient(t)
	defer cancel()
	defer client.Close(ctx)

	repo := NewRepository(client)
	testID := "test-memory-" + time.Now().Format("20060102-150405-000")
	defer cleanupTestData(ctx, client, testID)

	// Test Create
	t.Run("Create", func(t *testing.T) {
		mem := models.Memory{
			ID:       testID,
			Type:     models.TypeNote,
			Content:  "Test memory content",
			Tags:     []string{"test", "memory"},
			Metadata: map[string]string{"source": "test"},
		}

		created, err := repo.Add(ctx, mem, nil)
		if err != nil {
			t.Fatalf("Failed to create memory: %v", err)
		}

		if created.ID != testID {
			t.Errorf("Expected ID %s, got %s", testID, created.ID)
		}
		t.Logf("Created memory: %s", created.ID)
	})

	// Test GetByID
	t.Run("GetByID", func(t *testing.T) {
		mem, err := repo.GetByID(ctx, testID)
		if err != nil {
			t.Fatalf("Failed to get memory: %v", err)
		}
		if mem == nil {
			t.Fatal("Memory not found")
		}

		if mem.Content != "Test memory content" {
			t.Errorf("Expected content 'Test memory content', got '%s'", mem.Content)
		}
		t.Logf("Retrieved memory: %s - %s", mem.ID, mem.Type)
	})

	// Test Search
	t.Run("Search", func(t *testing.T) {
		results, err := repo.Search(ctx, "Test memory", 10)
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}

		found := false
		for _, r := range results {
			if r.Memory.ID == testID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Test memory not found in search results")
		}
		t.Logf("Search returned %d results", len(results))
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, testID)
		if err != nil {
			t.Fatalf("Failed to delete memory: %v", err)
		}

		// Verify deleted
		mem, err := repo.GetByID(ctx, testID)
		if err != nil {
			t.Fatalf("Error checking deleted memory: %v", err)
		}
		if mem != nil {
			t.Error("Memory should be deleted")
		}
		t.Logf("Deleted memory: %s", testID)
	})
}

// TestTaskPositioning tests task ordering within plans
func TestTaskPositioning(t *testing.T) {
	client, ctx, cancel := getTestClient(t)
	defer cancel()
	defer client.Close(ctx)

	planRepo := NewPlanRepository(client)
	taskRepo := NewTaskRepository(client)

	planID := "test-plan-pos-" + time.Now().Format("20060102-150405-000")
	task1ID := "test-task-pos1-" + time.Now().Format("20060102-150405-000")
	task2ID := "test-task-pos2-" + time.Now().Format("20060102-150405-000")
	task3ID := "test-task-pos3-" + time.Now().Format("20060102-150405-000")
	defer cleanupTestData(ctx, client, planID, task1ID, task2ID, task3ID)

	// Create plan
	plan := models.Plan{ID: planID, Name: "Position Test Plan", Status: models.PlanStatusActive}
	_, err := planRepo.Add(ctx, plan, nil)
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}

	// Create tasks in order
	t.Run("CreateTasksInOrder", func(t *testing.T) {
		task1 := models.Task{ID: task1ID, Content: "Task 1", Status: models.TaskStatusPending}
		_, err := taskRepo.Add(ctx, task1, []string{planID}, nil, nil, nil)
		if err != nil {
			t.Fatalf("Failed to create task 1: %v", err)
		}

		task2 := models.Task{ID: task2ID, Content: "Task 2", Status: models.TaskStatusPending}
		_, err = taskRepo.Add(ctx, task2, []string{planID}, nil, nil, nil)
		if err != nil {
			t.Fatalf("Failed to create task 2: %v", err)
		}

		task3 := models.Task{ID: task3ID, Content: "Task 3", Status: models.TaskStatusPending}
		_, err = taskRepo.Add(ctx, task3, []string{planID}, nil, nil, nil)
		if err != nil {
			t.Fatalf("Failed to create task 3: %v", err)
		}
	})

	// Verify order
	t.Run("VerifyOrder", func(t *testing.T) {
		tasks, err := taskRepo.List(ctx, planID, "", nil, 50)
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}

		if len(tasks) != 3 {
			t.Fatalf("Expected 3 tasks, got %d", len(tasks))
		}

		// Tasks should be in order 1, 2, 3
		if tasks[0].Task.ID != task1ID {
			t.Errorf("Expected first task to be %s, got %s", task1ID, tasks[0].Task.ID)
		}
		if tasks[1].Task.ID != task2ID {
			t.Errorf("Expected second task to be %s, got %s", task2ID, tasks[1].Task.ID)
		}
		if tasks[2].Task.ID != task3ID {
			t.Errorf("Expected third task to be %s, got %s", task3ID, tasks[2].Task.ID)
		}
		t.Logf("Tasks in correct order")
	})

	// Test reordering
	t.Run("Reorder", func(t *testing.T) {
		// Move task 3 to position 1 (swap with task 1)
		err := taskRepo.UpdatePositions(ctx, planID, map[string]float64{
			task3ID: 100, // Move to front
			task1ID: 200, // Move to middle
			task2ID: 300, // Move to end
		})
		if err != nil {
			t.Fatalf("Failed to update positions: %v", err)
		}

		// Verify new order: 3, 1, 2
		tasks, err := taskRepo.List(ctx, planID, "", nil, 50)
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}

		if tasks[0].Task.ID != task3ID {
			t.Errorf("Expected first task to be %s, got %s", task3ID, tasks[0].Task.ID)
		}
		if tasks[1].Task.ID != task1ID {
			t.Errorf("Expected second task to be %s, got %s", task1ID, tasks[1].Task.ID)
		}
		if tasks[2].Task.ID != task2ID {
			t.Errorf("Expected third task to be %s, got %s", task2ID, tasks[2].Task.ID)
		}
		t.Logf("Tasks reordered successfully")
	})
}

// TestCascadeDelete tests that deleting a plan cascades to orphan tasks
func TestCascadeDelete(t *testing.T) {
	client, ctx, cancel := getTestClient(t)
	defer cancel()
	defer client.Close(ctx)

	planRepo := NewPlanRepository(client)
	taskRepo := NewTaskRepository(client)

	plan1ID := "test-plan-cascade1-" + time.Now().Format("20060102-150405-000")
	plan2ID := "test-plan-cascade2-" + time.Now().Format("20060102-150405-000")
	taskOrphanID := "test-task-orphan-" + time.Now().Format("20060102-150405-000")
	taskSharedID := "test-task-shared-" + time.Now().Format("20060102-150405-000")
	defer cleanupTestData(ctx, client, plan1ID, plan2ID, taskOrphanID, taskSharedID)

	// Create two plans
	_, err := planRepo.Add(ctx, models.Plan{ID: plan1ID, Name: "Plan 1", Status: models.PlanStatusActive}, nil)
	if err != nil {
		t.Fatalf("Failed to create plan 1: %v", err)
	}
	_, err = planRepo.Add(ctx, models.Plan{ID: plan2ID, Name: "Plan 2", Status: models.PlanStatusActive}, nil)
	if err != nil {
		t.Fatalf("Failed to create plan 2: %v", err)
	}

	// Create orphan task (only in plan 1)
	_, err = taskRepo.Add(ctx, models.Task{ID: taskOrphanID, Content: "Orphan Task", Status: models.TaskStatusPending}, []string{plan1ID}, nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create orphan task: %v", err)
	}

	// Create shared task (in both plans)
	_, err = taskRepo.Add(ctx, models.Task{ID: taskSharedID, Content: "Shared Task", Status: models.TaskStatusPending}, []string{plan1ID, plan2ID}, nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create shared task: %v", err)
	}

	// Delete plan 1
	deletedCount, err := planRepo.Delete(ctx, plan1ID)
	if err != nil {
		t.Fatalf("Failed to delete plan: %v", err)
	}
	t.Logf("Deleted plan 1, cascade deleted %d tasks", deletedCount)

	// Orphan task should be deleted
	task, _ := taskRepo.GetByID(ctx, taskOrphanID)
	if task != nil {
		t.Error("Orphan task should have been deleted")
	}

	// Shared task should still exist
	task, _ = taskRepo.GetByID(ctx, taskSharedID)
	if task == nil {
		t.Error("Shared task should still exist")
	}

	// Shared task should now only be in plan 2
	task, plans, err := taskRepo.GetWithPlans(ctx, taskSharedID)
	if err != nil {
		t.Fatalf("Failed to get task with plans: %v", err)
	}
	if len(plans) != 1 {
		t.Errorf("Expected shared task to be in 1 plan, got %d", len(plans))
	}
	if len(plans) > 0 && plans[0].ID != plan2ID {
		t.Errorf("Expected shared task to be in plan 2, got %s", plans[0].ID)
	}
}
