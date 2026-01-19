//go:build integration
// +build integration

package neo4j

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fitz/associate/internal/models"
)

// Integration test helper to get a connected client
func getTestClient(t *testing.T) (*Client, context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

	uri := os.Getenv("NEO4J_URI")
	if uri == "" {
		uri = "bolt://localhost:7687"
	}

	cfg := Config{
		URI:      uri,
		Username: getEnvOrDefault("NEO4J_USERNAME", "neo4j"),
		Password: getEnvOrDefault("NEO4J_PASSWORD", "password"),
		Database: getEnvOrDefault("NEO4J_DATABASE", "neo4j"),
	}

	client, err := NewClient(ctx, cfg)
	if err != nil {
		cancel()
		t.Fatalf("Failed to connect to Neo4j: %v", err)
	}

	return client, ctx, cancel
}

// cleanupTestData removes test data created during tests
func cleanupTestData(ctx context.Context, client *Client, ids ...string) {
	session := client.Session(ctx)
	defer session.Close(ctx)

	for _, id := range ids {
		cypher := `
		MATCH (n)
		WHERE n.id = $id
		DETACH DELETE n
		`
		session.Run(ctx, cypher, map[string]any{"id": id})
	}
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
		t.Logf("✓ Created plan: %s", created.ID)
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
		if plan.Description != "A test plan for integration testing" {
			t.Errorf("Expected description mismatch")
		}
		t.Logf("✓ Retrieved plan: %s", plan.ID)
	})

	// Test Update
	t.Run("Update", func(t *testing.T) {
		newName := "Updated Test Plan"
		newStatus := "completed"

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
		t.Logf("✓ Updated plan: %s", updated.ID)
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		plans, err := repo.List(ctx, "completed", nil, 10)
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
			t.Errorf("Expected to find test plan in list")
		}
		t.Logf("✓ Listed %d plans", len(plans))
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		_, err := repo.Delete(ctx, testID)
		if err != nil {
			t.Fatalf("Failed to delete plan: %v", err)
		}

		// Verify deletion
		plan, _ := repo.GetByID(ctx, testID)
		if plan != nil {
			t.Error("Plan should have been deleted")
		}
		t.Log("✓ Deleted plan")
	})
}

// TestTaskRepository_CRUD tests basic Task CRUD operations
func TestTaskRepository_CRUD(t *testing.T) {
	client, ctx, cancel := getTestClient(t)
	defer cancel()
	defer client.Close(ctx)

	planRepo := NewPlanRepository(client)
	repo := NewTaskRepository(client)
	planID := "test-plan-crud-" + time.Now().Format("20060102-150405-000")
	testID := "test-task-" + time.Now().Format("20060102-150405-000")
	defer cleanupTestData(ctx, client, planID, testID)

	// Create a plan first (tasks require at least one plan)
	_, err := planRepo.Add(ctx, models.Plan{ID: planID, Name: "Test Plan for Tasks", Status: models.PlanStatusActive}, nil)
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}
	t.Log("✓ Created plan for task CRUD test")

	// Test Create
	t.Run("Create", func(t *testing.T) {
		task := models.Task{
			ID:       testID,
			Content:  "Test task for integration testing",
			Status:   models.TaskStatusPending,
			Tags:     []string{"test", "integration"},
			Metadata: map[string]string{"priority": "1"},
		}

		created, err := repo.Add(ctx, task, []string{planID}, nil)
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		if created.ID != testID {
			t.Errorf("Expected ID %s, got %s", testID, created.ID)
		}
		if created.Status != models.TaskStatusPending {
			t.Errorf("Expected status 'pending', got '%s'", created.Status)
		}
		t.Logf("✓ Created task: %s", created.ID)
	})

	// Test GetByID
	t.Run("GetByID", func(t *testing.T) {
		task, err := repo.GetByID(ctx, testID)
		if err != nil {
			t.Fatalf("Failed to get task: %v", err)
		}
		if task == nil {
			t.Fatal("Task not found")
		}

		if task.Content != "Test task for integration testing" {
			t.Errorf("Expected content mismatch")
		}
		t.Logf("✓ Retrieved task: %s", task.ID)
	})

	// Test Update
	t.Run("Update", func(t *testing.T) {
		newContent := "Updated test task"
		newStatus := "in_progress"

		updated, err := repo.Update(ctx, testID, &newContent, &newStatus, nil, nil, nil, nil)
		if err != nil {
			t.Fatalf("Failed to update task: %v", err)
		}

		if updated.Content != newContent {
			t.Errorf("Expected content '%s', got '%s'", newContent, updated.Content)
		}
		if string(updated.Status) != newStatus {
			t.Errorf("Expected status '%s', got '%s'", newStatus, updated.Status)
		}
		t.Logf("✓ Updated task: %s", updated.ID)
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		tasks, err := repo.List(ctx, "", "in_progress", nil, 10)
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}

		found := false
		for _, tsk := range tasks {
			if tsk.ID == testID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find test task in list")
		}
		t.Logf("✓ Listed %d tasks", len(tasks))
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, testID)
		if err != nil {
			t.Fatalf("Failed to delete task: %v", err)
		}

		// Verify deletion
		task, _ := repo.GetByID(ctx, testID)
		if task != nil {
			t.Error("Task should have been deleted")
		}
		t.Log("✓ Deleted task")
	})
}

// TestPlanTaskRelationship tests Plan-Task relationships
func TestPlanTaskRelationship(t *testing.T) {
	client, ctx, cancel := getTestClient(t)
	defer cancel()
	defer client.Close(ctx)

	planRepo := NewPlanRepository(client)
	taskRepo := NewTaskRepository(client)

	planID := "test-plan-rel-" + time.Now().Format("20060102-150405-000")
	taskID := "test-task-rel-" + time.Now().Format("20060102-150405-000")
	defer cleanupTestData(ctx, client, planID, taskID)

	// Create Plan
	plan := models.Plan{
		ID:          planID,
		Name:        "Relationship Test Plan",
		Description: "Testing Plan-Task relationships",
		Status:      models.PlanStatusActive,
	}
	_, err := planRepo.Add(ctx, plan, nil)
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}
	t.Log("✓ Created plan for relationship test")

	// Create Task linked to Plan
	task := models.Task{
		ID:      taskID,
		Content: "Task linked to plan",
		Status:  models.TaskStatusPending,
	}
	_, err = taskRepo.Add(ctx, task, []string{planID}, nil)
	if err != nil {
		t.Fatalf("Failed to create task with plan link: %v", err)
	}
	t.Log("✓ Created task linked to plan")

	// Verify GetWithTasks returns the task
	t.Run("GetPlanWithTasks", func(t *testing.T) {
		p, tasks, err := planRepo.GetWithTasks(ctx, planID)
		if err != nil {
			t.Fatalf("Failed to get plan with tasks: %v", err)
		}
		if p == nil {
			t.Fatal("Plan not found")
		}

		if len(tasks) != 1 {
			t.Errorf("Expected 1 task, got %d", len(tasks))
		} else if tasks[0].ID != taskID {
			t.Errorf("Expected task ID %s, got %s", taskID, tasks[0].ID)
		}
		t.Logf("✓ GetWithTasks returned %d tasks", len(tasks))
	})

	// Verify GetWithPlans returns the plan
	t.Run("GetTaskWithPlans", func(t *testing.T) {
		tsk, plans, err := taskRepo.GetWithPlans(ctx, taskID)
		if err != nil {
			t.Fatalf("Failed to get task with plans: %v", err)
		}
		if tsk == nil {
			t.Fatal("Task not found")
		}

		if len(plans) != 1 {
			t.Errorf("Expected 1 plan, got %d", len(plans))
		} else if plans[0].ID != planID {
			t.Errorf("Expected plan ID %s, got %s", planID, plans[0].ID)
		}
		t.Logf("✓ GetWithPlans returned %d plans", len(plans))
	})

	// Verify ListTasks by planID
	t.Run("ListTasksByPlan", func(t *testing.T) {
		tasks, err := taskRepo.List(ctx, planID, "", nil, 10)
		if err != nil {
			t.Fatalf("Failed to list tasks by plan: %v", err)
		}

		if len(tasks) != 1 {
			t.Errorf("Expected 1 task, got %d", len(tasks))
		}
		t.Logf("✓ ListTasks by plan returned %d tasks", len(tasks))
	})
}

// TestCascadeDelete tests that deleting a Plan cascade deletes orphan tasks
func TestCascadeDelete(t *testing.T) {
	client, ctx, cancel := getTestClient(t)
	defer cancel()
	defer client.Close(ctx)

	planRepo := NewPlanRepository(client)
	taskRepo := NewTaskRepository(client)

	plan1ID := "test-plan-cascade1-" + time.Now().Format("20060102-150405-000")
	plan2ID := "test-plan-cascade2-" + time.Now().Format("20060102-150405-000")
	task1ID := "test-task-orphan-" + time.Now().Format("20060102-150405-000")
	task2ID := "test-task-shared-" + time.Now().Format("20060102-150405-000")
	defer cleanupTestData(ctx, client, plan1ID, plan2ID, task1ID, task2ID)

	// Create two plans
	_, err := planRepo.Add(ctx, models.Plan{ID: plan1ID, Name: "Plan 1", Status: models.PlanStatusActive}, nil)
	if err != nil {
		t.Fatalf("Failed to create plan1: %v", err)
	}
	_, err = planRepo.Add(ctx, models.Plan{ID: plan2ID, Name: "Plan 2", Status: models.PlanStatusActive}, nil)
	if err != nil {
		t.Fatalf("Failed to create plan2: %v", err)
	}
	t.Log("✓ Created two plans for cascade delete test")

	// Create task1 linked only to plan1 (orphan when plan1 deleted)
	_, err = taskRepo.Add(ctx, models.Task{ID: task1ID, Content: "Orphan task", Status: models.TaskStatusPending}, []string{plan1ID}, nil)
	if err != nil {
		t.Fatalf("Failed to create task1: %v", err)
	}

	// Create task2 linked to both plans (should survive plan1 deletion)
	_, err = taskRepo.Add(ctx, models.Task{ID: task2ID, Content: "Shared task", Status: models.TaskStatusPending}, []string{plan1ID, plan2ID}, nil)
	if err != nil {
		t.Fatalf("Failed to create task2: %v", err)
	}
	t.Log("✓ Created orphan task and shared task")

	// Delete plan1 - should cascade delete task1 but keep task2
	deletedCount, err := planRepo.Delete(ctx, plan1ID)
	if err != nil {
		t.Fatalf("Failed to delete plan1: %v", err)
	}
	t.Logf("✓ Deleted plan1, cascade deleted %d tasks", deletedCount)

	// Verify task1 (orphan) was deleted
	task1, _ := taskRepo.GetByID(ctx, task1ID)
	if task1 != nil {
		t.Error("Task1 (orphan) should have been cascade deleted")
	} else {
		t.Log("✓ Orphan task was cascade deleted")
	}

	// Verify task2 (shared) still exists
	task2, err := taskRepo.GetByID(ctx, task2ID)
	if err != nil {
		t.Fatalf("Failed to get task2: %v", err)
	}
	if task2 == nil {
		t.Error("Task2 (shared) should NOT have been deleted")
	} else {
		t.Log("✓ Shared task still exists")
	}

	// Verify task2 is still linked to plan2
	_, plans, err := taskRepo.GetWithPlans(ctx, task2ID)
	if err != nil {
		t.Fatalf("Failed to get task2 with plans: %v", err)
	}
	if len(plans) != 1 || plans[0].ID != plan2ID {
		t.Errorf("Task2 should still be linked to plan2, got %d plans", len(plans))
	} else {
		t.Log("✓ Shared task still linked to plan2")
	}
}

// TestDeletePlanWithNoTasks tests that deleting a plan with no tasks works correctly
// This is a regression test for a bug where the delete query returned "plan not found"
// when the plan had no associated tasks due to UNWIND on an empty collection.
func TestDeletePlanWithNoTasks(t *testing.T) {
	client, ctx, cancel := getTestClient(t)
	defer cancel()
	defer client.Close(ctx)

	planRepo := NewPlanRepository(client)
	planID := "test-plan-notasks-" + time.Now().Format("20060102-150405-000")
	defer cleanupTestData(ctx, client, planID)

	// Create a plan with no tasks
	_, err := planRepo.Add(ctx, models.Plan{ID: planID, Name: "Plan No Tasks", Status: models.PlanStatusActive}, nil)
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}
	t.Log("✓ Created plan with no tasks")

	// Delete the plan (should succeed, not return "plan not found")
	deletedCount, err := planRepo.Delete(ctx, planID)
	if err != nil {
		t.Fatalf("Failed to delete plan with no tasks: %v", err)
	}
	if deletedCount != 0 {
		t.Errorf("Expected 0 tasks deleted, got %d", deletedCount)
	}
	t.Logf("✓ Deleted plan successfully, %d tasks cascade deleted", deletedCount)

	// Verify plan is gone
	plan, _ := planRepo.GetByID(ctx, planID)
	if plan != nil {
		t.Error("Plan should have been deleted")
	} else {
		t.Log("✓ Verified plan no longer exists")
	}
}

// TestTaskRequiresPlan tests that tasks must belong to at least one valid plan
func TestTaskRequiresPlan(t *testing.T) {
	client, ctx, cancel := getTestClient(t)
	defer cancel()
	defer client.Close(ctx)

	planRepo := NewPlanRepository(client)
	taskRepo := NewTaskRepository(client)

	timestamp := time.Now().Format("20060102-150405-000")
	planID := "test-plan-require-" + timestamp
	taskID := "test-task-require-" + timestamp
	defer cleanupTestData(ctx, client, planID, taskID)

	// Test: Creating task with no plan_ids should fail
	t.Run("CreateWithNoPlansFails", func(t *testing.T) {
		task := models.Task{
			ID:      taskID + "-noplan",
			Content: "Task without plan",
			Status:  models.TaskStatusPending,
		}
		_, err := taskRepo.Add(ctx, task, []string{}, nil)
		if err == nil {
			t.Error("Expected error when creating task without plans")
			cleanupTestData(ctx, client, taskID+"-noplan")
		} else {
			t.Logf("✓ Got expected error: %v", err)
		}
	})

	// Test: Creating task with nil plan_ids should fail
	t.Run("CreateWithNilPlansFails", func(t *testing.T) {
		task := models.Task{
			ID:      taskID + "-nilplan",
			Content: "Task with nil plans",
			Status:  models.TaskStatusPending,
		}
		_, err := taskRepo.Add(ctx, task, nil, nil)
		if err == nil {
			t.Error("Expected error when creating task with nil plans")
			cleanupTestData(ctx, client, taskID+"-nilplan")
		} else {
			t.Logf("✓ Got expected error: %v", err)
		}
	})

	// Test: Creating task with non-existent plan should fail
	t.Run("CreateWithNonExistentPlanFails", func(t *testing.T) {
		task := models.Task{
			ID:      taskID + "-badplan",
			Content: "Task with non-existent plan",
			Status:  models.TaskStatusPending,
		}
		_, err := taskRepo.Add(ctx, task, []string{"non-existent-plan-id"}, nil)
		if err == nil {
			t.Error("Expected error when creating task with non-existent plan")
			cleanupTestData(ctx, client, taskID+"-badplan")
		} else {
			t.Logf("✓ Got expected error: %v", err)
		}
	})

	// Test: Creating task with valid plan should succeed
	t.Run("CreateWithValidPlanSucceeds", func(t *testing.T) {
		// First create a valid plan
		_, err := planRepo.Add(ctx, models.Plan{ID: planID, Name: "Valid Plan", Status: models.PlanStatusActive}, nil)
		if err != nil {
			t.Fatalf("Failed to create plan: %v", err)
		}

		task := models.Task{
			ID:      taskID,
			Content: "Task with valid plan",
			Status:  models.TaskStatusPending,
		}
		created, err := taskRepo.Add(ctx, task, []string{planID}, nil)
		if err != nil {
			t.Fatalf("Expected success when creating task with valid plan, got: %v", err)
		}
		t.Logf("✓ Created task successfully: %s", created.ID)
	})

	// Test: Updating task to add non-existent plan should fail
	t.Run("UpdateWithNonExistentPlanFails", func(t *testing.T) {
		_, err := taskRepo.Update(ctx, taskID, nil, nil, nil, nil, []string{"non-existent-plan-id"}, nil)
		if err == nil {
			t.Error("Expected error when updating task with non-existent plan")
		} else {
			t.Logf("✓ Got expected error: %v", err)
		}
	})
}

// TestCrossTypeRelationships tests relationships between Memory, Plan, and Task nodes
func TestCrossTypeRelationships(t *testing.T) {
	client, ctx, cancel := getTestClient(t)
	defer cancel()
	defer client.Close(ctx)

	memRepo := NewRepository(client)
	planRepo := NewPlanRepository(client)
	taskRepo := NewTaskRepository(client)

	memID := "test-mem-cross-" + time.Now().Format("20060102-150405-000")
	planID := "test-plan-cross-" + time.Now().Format("20060102-150405-000")
	taskID := "test-task-cross-" + time.Now().Format("20060102-150405-000")
	defer cleanupTestData(ctx, client, memID, planID, taskID)

	// Create a memory for architectural decision
	mem := models.Memory{
		ID:      memID,
		Type:    models.TypeNote,
		Content: "Authentication architecture decision",
		Tags:    []string{"architecture", "auth"},
	}
	_, err := memRepo.Add(ctx, mem, nil)
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}
	t.Log("✓ Created memory for cross-type test")

	// Create plan that references the memory
	plan := models.Plan{
		ID:          planID,
		Name:        "Auth Implementation",
		Description: "Implement authentication based on architecture decision",
		Status:      models.PlanStatusActive,
	}
	planRels := []models.Relationship{
		{ToID: memID, Type: models.RelReferences},
	}
	_, err = planRepo.Add(ctx, plan, planRels)
	if err != nil {
		t.Fatalf("Failed to create plan with memory relationship: %v", err)
	}
	t.Log("✓ Created plan with REFERENCES relationship to memory")

	// Create task that implements the memory decision
	task := models.Task{
		ID:      taskID,
		Content: "Implement JWT token validation",
		Status:  models.TaskStatusPending,
	}
	taskRels := []models.Relationship{
		{ToID: memID, Type: models.RelImplements},
	}
	_, err = taskRepo.Add(ctx, task, []string{planID}, taskRels)
	if err != nil {
		t.Fatalf("Failed to create task with memory relationship: %v", err)
	}
	t.Log("✓ Created task with IMPLEMENTS relationship to memory and PART_OF plan")

	// Verify cross-type traversal using GetRelated
	t.Run("GetRelatedFromMemory", func(t *testing.T) {
		related, err := memRepo.GetRelated(ctx, memID, "", "incoming", 1)
		if err != nil {
			t.Fatalf("Failed to get related from memory: %v", err)
		}

		foundPlan := false
		foundTask := false
		for _, r := range related {
			if r.ID == planID {
				foundPlan = true
				if r.RelationType != string(models.RelReferences) {
					t.Errorf("Expected REFERENCES relationship, got %s", r.RelationType)
				}
			}
			if r.ID == taskID {
				foundTask = true
				if r.RelationType != string(models.RelImplements) {
					t.Errorf("Expected IMPLEMENTS relationship, got %s", r.RelationType)
				}
			}
		}

		if !foundPlan {
			t.Error("Expected to find plan in related nodes")
		}
		if !foundTask {
			t.Error("Expected to find task in related nodes")
		}
		t.Logf("✓ GetRelated found %d cross-type relationships", len(related))
	})
}

// TestEndToEndWorkflow simulates a complete agent workflow
func TestEndToEndWorkflow(t *testing.T) {
	client, ctx, cancel := getTestClient(t)
	defer cancel()
	defer client.Close(ctx)

	planRepo := NewPlanRepository(client)
	taskRepo := NewTaskRepository(client)

	timestamp := time.Now().Format("20060102-150405-000")
	planID := "e2e-plan-" + timestamp
	task1ID := "e2e-task1-" + timestamp
	task2ID := "e2e-task2-" + timestamp
	task3ID := "e2e-task3-" + timestamp
	defer cleanupTestData(ctx, client, planID, task1ID, task2ID, task3ID)

	// Step 1: Create a plan for a feature
	t.Log("=== Step 1: Create Plan ===")
	plan := models.Plan{
		ID:          planID,
		Name:        "Add User Authentication",
		Description: "Implement user authentication with JWT tokens",
		Status:      models.PlanStatusDraft,
		Tags:        []string{"feature", "auth"},
	}
	createdPlan, err := planRepo.Add(ctx, plan, nil)
	if err != nil {
		t.Fatalf("Step 1 failed: %v", err)
	}
	t.Logf("✓ Created plan: %s (status: %s)", createdPlan.Name, createdPlan.Status)

	// Step 2: Add tasks to the plan
	t.Log("=== Step 2: Create Tasks ===")
	tasks := []models.Task{
		{ID: task1ID, Content: "Design database schema", Status: models.TaskStatusPending, Metadata: map[string]string{"priority": "1"}},
		{ID: task2ID, Content: "Implement user model", Status: models.TaskStatusPending, Metadata: map[string]string{"priority": "2"}},
		{ID: task3ID, Content: "Add JWT endpoints", Status: models.TaskStatusPending, Metadata: map[string]string{"priority": "3"}},
	}

	prevTaskID := ""
	for _, task := range tasks {
		var rels []models.Relationship
		if prevTaskID != "" {
			rels = append(rels, models.Relationship{ToID: prevTaskID, Type: models.RelFollows})
		}
		_, err := taskRepo.Add(ctx, task, []string{planID}, rels)
		if err != nil {
			t.Fatalf("Failed to create task %s: %v", task.ID, err)
		}
		t.Logf("✓ Created task: %s", task.Content)
		prevTaskID = task.ID
	}

	// Step 3: Activate the plan
	t.Log("=== Step 3: Activate Plan ===")
	activeStatus := "active"
	_, err = planRepo.Update(ctx, planID, nil, nil, &activeStatus, nil, nil, nil)
	if err != nil {
		t.Fatalf("Step 3 failed: %v", err)
	}
	t.Log("✓ Plan activated")

	// Step 4: Start working on first task
	t.Log("=== Step 4: Work on Task ===")
	inProgressStatus := "in_progress"
	_, err = taskRepo.Update(ctx, task1ID, nil, &inProgressStatus, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Step 4 failed: %v", err)
	}
	t.Log("✓ Task 1 now in progress")

	// Step 5: Complete first task
	t.Log("=== Step 5: Complete Task ===")
	completedStatus := "completed"
	_, err = taskRepo.Update(ctx, task1ID, nil, &completedStatus, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("Step 5 failed: %v", err)
	}
	t.Log("✓ Task 1 completed")

	// Step 6: Check plan progress
	t.Log("=== Step 6: Check Progress ===")
	_, planTasks, err := planRepo.GetWithTasks(ctx, planID)
	if err != nil {
		t.Fatalf("Step 6 failed: %v", err)
	}

	completed := 0
	for _, tsk := range planTasks {
		if tsk.Status == models.TaskStatusCompleted {
			completed++
		}
	}
	t.Logf("✓ Progress: %d/%d tasks completed", completed, len(planTasks))

	// Step 7: Complete all tasks and plan
	t.Log("=== Step 7: Complete All ===")
	for _, id := range []string{task2ID, task3ID} {
		_, err = taskRepo.Update(ctx, id, nil, &completedStatus, nil, nil, nil, nil)
		if err != nil {
			t.Fatalf("Failed to complete task %s: %v", id, err)
		}
	}
	planCompletedStatus := "completed"
	_, err = planRepo.Update(ctx, planID, nil, nil, &planCompletedStatus, nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to complete plan: %v", err)
	}
	t.Log("✓ All tasks and plan completed")

	// Verify final state
	finalPlan, finalTasks, _ := planRepo.GetWithTasks(ctx, planID)
	if finalPlan.Status != models.PlanStatusCompleted {
		t.Errorf("Expected plan status 'completed', got '%s'", finalPlan.Status)
	}
	allCompleted := true
	for _, tsk := range finalTasks {
		if tsk.Status != models.TaskStatusCompleted {
			allCompleted = false
			break
		}
	}
	if !allCompleted {
		t.Error("Not all tasks are completed")
	}
	t.Log("✓ End-to-end workflow completed successfully")
}
