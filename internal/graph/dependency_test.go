//go:build integration
// +build integration

package graph

import (
	"testing"
	"time"

	"github.com/fitz/associate/internal/models"
)

// TestTaskDependencyCreation tests adding DEPENDS_ON relationships between tasks
func TestTaskDependencyCreation(t *testing.T) {
	client, ctx, cancel := getTestClient(t)
	defer cancel()
	defer client.Close(ctx)

	planRepo := NewPlanRepository(client)
	taskRepo := NewTaskRepository(client)

	planID := "test-plan-deps-" + time.Now().Format("20060102-150405-000")
	task1ID := "test-task-dep1-" + time.Now().Format("20060102-150405-000")
	task2ID := "test-task-dep2-" + time.Now().Format("20060102-150405-000")
	defer cleanupTestData(ctx, client, planID, task1ID, task2ID)

	// Create a plan
	plan := models.Plan{
		ID:     planID,
		Name:   "Dependency Test Plan",
		Status: models.PlanStatusActive,
	}
	_, err := planRepo.Add(ctx, plan, nil)
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}

	// Create two tasks
	task1 := models.Task{
		ID:      task1ID,
		Content: "Task 1",
		Status:  models.TaskStatusPending,
	}
	_, err = taskRepo.Add(ctx, task1, []string{planID}, nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create task 1: %v", err)
	}

	task2 := models.Task{
		ID:      task2ID,
		Content: "Task 2 - depends on task 1",
		Status:  models.TaskStatusPending,
	}
	_, err = taskRepo.Add(ctx, task2, []string{planID}, nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create task 2: %v", err)
	}

	// Now try to add a DEPENDS_ON relationship via update
	t.Run("AddDependency", func(t *testing.T) {
		rels := []models.Relationship{
			{ToID: task1ID, Type: models.RelDependsOn},
		}
		updated, err := taskRepo.Update(ctx, task2ID, nil, nil, nil, nil, nil, rels)
		if err != nil {
			t.Fatalf("Failed to add dependency: %v", err)
		}
		t.Logf("Added dependency: task %s now depends on %s", updated.ID, task1ID)
	})
}
