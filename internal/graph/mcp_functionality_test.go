//go:build integration
// +build integration

package graph

import (
	"testing"
	"time"

	"github.com/Thomas-Fitz/associate/internal/models"
)

// TestMCPFunctionality tests the core MCP functionality that was broken
// This test validates that all node types (Zone, Plan, Task) can be created
// and that the auto-zone feature works correctly
func TestMCPFunctionality(t *testing.T) {
	client, ctx, cancel := getTestClient(t)
	defer cancel()
	defer client.Close(ctx)

	zoneRepo := NewZoneRepository(client)
	planRepo := NewPlanRepositoryWithZoneRepo(client, zoneRepo)
	taskRepo := NewTaskRepository(client)

	timestamp := time.Now().Format("20060102-150405-000")

	// Track IDs for cleanup
	var createdZoneID, autoZoneID, createdPlanID, createdPlan2ID, createdTaskID string

	defer func() {
		// Cleanup all created resources
		if createdTaskID != "" {
			taskRepo.Delete(ctx, createdTaskID)
		}
		if createdPlanID != "" {
			planRepo.Delete(ctx, createdPlanID)
		}
		if createdPlan2ID != "" {
			planRepo.Delete(ctx, createdPlan2ID)
		}
		if createdZoneID != "" {
			zoneRepo.Delete(ctx, createdZoneID)
		}
		if autoZoneID != "" {
			zoneRepo.Delete(ctx, autoZoneID)
		}
	}()

	// Test 1: Create Zone directly
	t.Run("CreateZone", func(t *testing.T) {
		zone := models.Zone{
			Name:        "MCP Test Zone " + timestamp,
			Description: "A test zone for MCP functionality",
		}
		createdZone, err := zoneRepo.Add(ctx, zone)
		if err != nil {
			t.Fatalf("Failed to create zone: %v", err)
		}
		createdZoneID = createdZone.ID
		t.Logf("Created zone: %s (%s)", createdZone.Name, createdZone.ID)
	})

	// Test 2: Create Plan with auto-zone (the main bug that was fixed)
	t.Run("CreatePlanWithAutoZone", func(t *testing.T) {
		plan := models.Plan{
			Name:        "MCP Test Plan Auto Zone " + timestamp,
			Description: "A test plan with auto zone creation",
			Status:      models.PlanStatusActive,
		}
		createdPlan, zoneID, err := planRepo.AddWithZone(ctx, plan, "", nil)
		if err != nil {
			t.Fatalf("Failed to create plan with auto zone: %v", err)
		}
		createdPlanID = createdPlan.ID
		autoZoneID = zoneID

		if zoneID == "" {
			t.Error("Expected auto-created zone ID, got empty string")
		}

		// Verify the zone was created with the plan's name
		autoZone, err := zoneRepo.GetByID(ctx, zoneID)
		if err != nil {
			t.Fatalf("Failed to get auto-created zone: %v", err)
		}
		if autoZone.Name != plan.Name {
			t.Errorf("Expected zone name '%s', got '%s'", plan.Name, autoZone.Name)
		}

		t.Logf("Created plan: %s with auto-zone: %s", createdPlan.ID, zoneID)
	})

	// Test 3: Create Plan with existing zone
	t.Run("CreatePlanWithExistingZone", func(t *testing.T) {
		plan := models.Plan{
			Name:        "MCP Test Plan Existing Zone " + timestamp,
			Description: "A test plan with existing zone",
			Status:      models.PlanStatusActive,
		}
		createdPlan, zoneID, err := planRepo.AddWithZone(ctx, plan, createdZoneID, nil)
		if err != nil {
			t.Fatalf("Failed to create plan with existing zone: %v", err)
		}
		createdPlan2ID = createdPlan.ID

		if zoneID != createdZoneID {
			t.Errorf("Expected zone ID %s, got %s", createdZoneID, zoneID)
		}

		t.Logf("Created plan: %s linked to zone: %s", createdPlan.ID, zoneID)
	})

	// Test 4: Create Task
	t.Run("CreateTask", func(t *testing.T) {
		task := models.Task{
			Content: "MCP Test Task " + timestamp,
			Status:  models.TaskStatusPending,
		}
		createdTask, err := taskRepo.Add(ctx, task, []string{createdPlanID}, nil, nil, nil)
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
		createdTaskID = createdTask.ID
		t.Logf("Created task: %s", createdTask.ID)
	})

	// Test 5: Verify plan-task relationship
	t.Run("VerifyPlanTaskRelationship", func(t *testing.T) {
		task, plans, err := taskRepo.GetWithPlans(ctx, createdTaskID)
		if err != nil {
			t.Fatalf("Failed to get task with plans: %v", err)
		}
		if len(plans) != 1 {
			t.Errorf("Expected 1 plan, got %d", len(plans))
		}
		if len(plans) > 0 && plans[0].ID != createdPlanID {
			t.Errorf("Expected plan ID %s, got %s", createdPlanID, plans[0].ID)
		}
		t.Logf("Task %s belongs to plan %s", task.ID, plans[0].ID)
	})

	// Test 6: Verify zone-plan relationship
	t.Run("VerifyZonePlanRelationship", func(t *testing.T) {
		zoneID, err := planRepo.GetZoneID(ctx, createdPlanID)
		if err != nil {
			t.Fatalf("Failed to get plan zone ID: %v", err)
		}
		if zoneID != autoZoneID {
			t.Errorf("Expected zone ID %s, got %s", autoZoneID, zoneID)
		}
		t.Logf("Plan %s belongs to zone %s", createdPlanID, zoneID)
	})

	// Test 7: Verify zone contents
	t.Run("VerifyZoneContents", func(t *testing.T) {
		zoneWithContents, err := zoneRepo.GetWithContents(ctx, autoZoneID)
		if err != nil {
			t.Fatalf("Failed to get zone contents: %v", err)
		}
		if zoneWithContents == nil {
			t.Fatal("Zone contents is nil")
		}
		if len(zoneWithContents.Plans) != 1 {
			t.Errorf("Expected 1 plan in zone, got %d", len(zoneWithContents.Plans))
		}
		if len(zoneWithContents.Plans) > 0 && len(zoneWithContents.Plans[0].Tasks) != 1 {
			t.Errorf("Expected 1 task in plan, got %d", len(zoneWithContents.Plans[0].Tasks))
		}
		t.Logf("Zone %s contains %d plans", zoneWithContents.Zone.ID, len(zoneWithContents.Plans))
	})

	// Test 8: List zones
	t.Run("ListZones", func(t *testing.T) {
		zones, err := zoneRepo.List(ctx, "", 100)
		if err != nil {
			t.Fatalf("Failed to list zones: %v", err)
		}

		foundCreated := false
		foundAuto := false
		for _, z := range zones {
			if z.Zone.ID == createdZoneID {
				foundCreated = true
			}
			if z.Zone.ID == autoZoneID {
				foundAuto = true
			}
		}

		if !foundCreated {
			t.Error("Created zone not found in list")
		}
		if !foundAuto {
			t.Error("Auto-created zone not found in list")
		}
		t.Logf("Found %d zones", len(zones))
	})
}
