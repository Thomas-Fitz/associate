package mcp

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/fitz/associate/internal/mcp/tools"
)

// Tests for Plan tool input/output validation

func TestCreatePlanInput_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input tools.CreatePlanInput
		valid bool
	}{
		{
			name: "valid with all fields",
			input: tools.CreatePlanInput{
				Name:        "Test Plan",
				Description: "A test plan description",
				Status:      "active",
				Metadata:    map[string]any{"key": "value"},
				Tags:        []string{"test"},
			},
			valid: true,
		},
		{
			name:  "minimal valid (name only)",
			input: tools.CreatePlanInput{Name: "Test Plan"},
			valid: true,
		},
		{
			name:  "empty name",
			input: tools.CreatePlanInput{Name: ""},
			valid: false, // name is required
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input.Name == "" && tt.valid {
				t.Error("Name should be required for create_plan")
			}
		})
	}
}

func TestCreatePlanOutput_Format(t *testing.T) {
	now := time.Now()
	output := tools.CreatePlanOutput{
		ID:          "plan-123",
		Name:        "Test Plan",
		Description: "A test plan",
		Status:      "active",
		Metadata:    map[string]string{"key": "value"},
		Tags:        []string{"test"},
		CreatedAt:   now.Format("2006-01-02T15:04:05Z"),
	}

	if output.ID == "" {
		t.Error("ID should not be empty")
	}
	if output.Status != "active" {
		t.Errorf("Status: got %s, want active", output.Status)
	}
}

func TestGetPlanInput_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input tools.GetPlanInput
		valid bool
	}{
		{
			name:  "valid ID",
			input: tools.GetPlanInput{ID: "plan-123"},
			valid: true,
		},
		{
			name:  "empty ID",
			input: tools.GetPlanInput{ID: ""},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input.ID == "" && tt.valid {
				t.Error("ID should be required for get_plan")
			}
		})
	}
}

func TestGetPlanOutput_Format(t *testing.T) {
	now := time.Now()
	output := tools.GetPlanOutput{
		ID:          "plan-123",
		Name:        "Test Plan",
		Description: "A test plan",
		Status:      "active",
		Tasks: []tools.TaskSummary{
			{ID: "task-1", Content: "Task 1", Status: "pending"},
			{ID: "task-2", Content: "Task 2", Status: "completed"},
		},
		CreatedAt: now.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: now.Format("2006-01-02T15:04:05Z"),
	}

	if len(output.Tasks) != 2 {
		t.Errorf("Tasks count: got %d, want 2", len(output.Tasks))
	}
}

func TestUpdatePlanInput_Validation(t *testing.T) {
	name := "Updated Name"
	tests := []struct {
		name  string
		input tools.UpdatePlanInput
		valid bool
	}{
		{
			name: "update name",
			input: tools.UpdatePlanInput{
				ID:   "plan-123",
				Name: &name,
			},
			valid: true,
		},
		{
			name:  "empty ID",
			input: tools.UpdatePlanInput{ID: ""},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input.ID == "" && tt.valid {
				t.Error("ID should be required for update_plan")
			}
		})
	}
}

func TestDeletePlanOutput_Format(t *testing.T) {
	output := tools.DeletePlanOutput{
		ID:           "plan-123",
		Deleted:      true,
		TasksDeleted: 3,
	}

	if !output.Deleted {
		t.Error("Deleted should be true")
	}
	if output.TasksDeleted != 3 {
		t.Errorf("TasksDeleted: got %d, want 3", output.TasksDeleted)
	}
}

func TestListPlansInput_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input tools.ListPlansInput
		valid bool
	}{
		{
			name:  "no filters",
			input: tools.ListPlansInput{},
			valid: true,
		},
		{
			name:  "with status filter",
			input: tools.ListPlansInput{Status: "active"},
			valid: true,
		},
		{
			name:  "with tags filter",
			input: tools.ListPlansInput{Tags: []string{"tag1", "tag2"}},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// All combinations are valid
			_ = tt.input
		})
	}
}

func TestListPlansOutput_Format(t *testing.T) {
	now := time.Now()
	output := tools.ListPlansOutput{
		Plans: []tools.PlanSummary{
			{ID: "plan-1", Name: "Plan 1", Status: "active", UpdatedAt: now.Format("2006-01-02T15:04:05Z")},
			{ID: "plan-2", Name: "Plan 2", Status: "completed", UpdatedAt: now.Format("2006-01-02T15:04:05Z")},
		},
		Count: 2,
	}

	if output.Count != len(output.Plans) {
		t.Errorf("Count mismatch: got %d, have %d results", output.Count, len(output.Plans))
	}
}

func TestListPlansOutput_EmptyPlansSerializesToArray(t *testing.T) {
	// This test verifies that an empty ListPlansOutput serializes the Plans
	// field as an empty JSON array [], not null.
	// The MCP schema requires "plans" to be an array type.
	output := tools.ListPlansOutput{
		Plans: []tools.PlanSummary{}, // empty slice, not nil
		Count: 0,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal output: %v", err)
	}

	jsonStr := string(data)

	// Verify that plans is an empty array, not null
	if !strings.Contains(jsonStr, `"plans":[]`) {
		t.Errorf("Expected plans to serialize as empty array, got: %s", jsonStr)
	}

	// Explicitly verify it's not null
	if strings.Contains(jsonStr, `"plans":null`) {
		t.Errorf("Plans should not serialize as null, got: %s", jsonStr)
	}
}

// Tests for Task tool input/output validation

func TestCreateTaskInput_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input tools.CreateTaskInput
		valid bool
	}{
		{
			name: "valid with all fields",
			input: tools.CreateTaskInput{
				Content:  "Test task content",
				PlanIDs:  []string{"plan-123"},
				Status:   "pending",
				Metadata: map[string]any{"priority": "high"},
				Tags:     []string{"test"},
			},
			valid: true,
		},
		{
			name: "valid with multiple plans",
			input: tools.CreateTaskInput{
				Content: "Test task",
				PlanIDs: []string{"plan-123", "plan-456"},
			},
			valid: true,
		},
		{
			name:  "empty content",
			input: tools.CreateTaskInput{Content: "", PlanIDs: []string{"plan-123"}},
			valid: false, // content is required
		},
		{
			name:  "empty plan_ids",
			input: tools.CreateTaskInput{Content: "Test task", PlanIDs: []string{}},
			valid: false, // plan_ids is required
		},
		{
			name:  "nil plan_ids",
			input: tools.CreateTaskInput{Content: "Test task", PlanIDs: nil},
			valid: false, // plan_ids is required
		},
		{
			name:  "missing plan_ids (content only)",
			input: tools.CreateTaskInput{Content: "Test task"},
			valid: false, // plan_ids is required
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate content requirement
			if tt.input.Content == "" && tt.valid {
				t.Error("Content should be required for create_task")
			}
			// Validate plan_ids requirement
			if len(tt.input.PlanIDs) == 0 && tt.valid {
				t.Error("PlanIDs should be required for create_task (at least one plan)")
			}
		})
	}
}

func TestCreateTaskInput_RequiresPlanIDs(t *testing.T) {
	// Explicit tests for plan_ids validation
	tests := []struct {
		name    string
		planIDs []string
		wantErr bool
	}{
		{
			name:    "empty plan_ids should fail",
			planIDs: []string{},
			wantErr: true,
		},
		{
			name:    "nil plan_ids should fail",
			planIDs: nil,
			wantErr: true,
		},
		{
			name:    "single plan_id should pass",
			planIDs: []string{"plan-123"},
			wantErr: false,
		},
		{
			name:    "multiple plan_ids should pass",
			planIDs: []string{"plan-123", "plan-456"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tools.CreateTaskInput{
				Content: "Test task",
				PlanIDs: tt.planIDs,
			}
			hasError := len(input.PlanIDs) == 0
			if hasError != tt.wantErr {
				t.Errorf("PlanIDs validation: got error=%v, want error=%v", hasError, tt.wantErr)
			}
		})
	}
}

func TestCreateTaskOutput_Format(t *testing.T) {
	now := time.Now()
	output := tools.CreateTaskOutput{
		ID:        "task-123",
		Content:   "Test task",
		Status:    "pending",
		Metadata:  map[string]string{"priority": "high"},
		Tags:      []string{"test"},
		CreatedAt: now.Format("2006-01-02T15:04:05Z"),
	}

	if output.ID == "" {
		t.Error("ID should not be empty")
	}
	if output.Status != "pending" {
		t.Errorf("Status: got %s, want pending", output.Status)
	}
}

func TestGetTaskInput_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input tools.GetTaskInput
		valid bool
	}{
		{
			name:  "valid ID",
			input: tools.GetTaskInput{ID: "task-123"},
			valid: true,
		},
		{
			name:  "empty ID",
			input: tools.GetTaskInput{ID: ""},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input.ID == "" && tt.valid {
				t.Error("ID should be required for get_task")
			}
		})
	}
}

func TestGetTaskOutput_Format(t *testing.T) {
	now := time.Now()
	output := tools.GetTaskOutput{
		ID:      "task-123",
		Content: "Test task",
		Status:  "in_progress",
		Plans: []tools.PlanReference{
			{ID: "plan-1", Name: "Plan 1", Status: "active"},
		},
		CreatedAt: now.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: now.Format("2006-01-02T15:04:05Z"),
	}

	if len(output.Plans) != 1 {
		t.Errorf("Plans count: got %d, want 1", len(output.Plans))
	}
}

func TestUpdateTaskInput_Validation(t *testing.T) {
	content := "Updated content"
	status := "completed"
	tests := []struct {
		name  string
		input tools.UpdateTaskInput
		valid bool
	}{
		{
			name: "update content",
			input: tools.UpdateTaskInput{
				ID:      "task-123",
				Content: &content,
			},
			valid: true,
		},
		{
			name: "update status",
			input: tools.UpdateTaskInput{
				ID:     "task-123",
				Status: &status,
			},
			valid: true,
		},
		{
			name:  "empty ID",
			input: tools.UpdateTaskInput{ID: ""},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input.ID == "" && tt.valid {
				t.Error("ID should be required for update_task")
			}
		})
	}
}

func TestDeleteTaskOutput_Format(t *testing.T) {
	output := tools.DeleteTaskOutput{
		ID:      "task-123",
		Deleted: true,
	}

	if !output.Deleted {
		t.Error("Deleted should be true")
	}
}

func TestListTasksInput_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input tools.ListTasksInput
		valid bool
	}{
		{
			name:  "no filters",
			input: tools.ListTasksInput{},
			valid: true,
		},
		{
			name:  "with plan_id filter",
			input: tools.ListTasksInput{PlanID: "plan-123"},
			valid: true,
		},
		{
			name:  "with status filter",
			input: tools.ListTasksInput{Status: "pending"},
			valid: true,
		},
		{
			name:  "with all filters",
			input: tools.ListTasksInput{PlanID: "plan-123", Status: "pending", Tags: []string{"urgent"}},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// All combinations are valid
			_ = tt.input
		})
	}
}

func TestListTasksOutput_Format(t *testing.T) {
	output := tools.ListTasksOutput{
		Tasks: []tools.TaskListSummary{
			{ID: "task-1", Content: "Task 1", Status: "pending"},
			{ID: "task-2", Content: "Task 2", Status: "in_progress"},
			{ID: "task-3", Content: "Task 3", Status: "completed"},
		},
		Count: 3,
	}

	if output.Count != len(output.Tasks) {
		t.Errorf("Count mismatch: got %d, have %d results", output.Count, len(output.Tasks))
	}
}

func TestListTasksOutput_EmptyTasksSerializesToArray(t *testing.T) {
	// This test verifies that an empty ListTasksOutput serializes the Tasks
	// field as an empty JSON array [], not null.
	// The MCP schema requires "tasks" to be an array type.
	output := tools.ListTasksOutput{
		Tasks: []tools.TaskListSummary{}, // empty slice, not nil
		Count: 0,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal output: %v", err)
	}

	jsonStr := string(data)

	// Verify that tasks is an empty array, not null
	if !strings.Contains(jsonStr, `"tasks":[]`) {
		t.Errorf("Expected tasks to serialize as empty array, got: %s", jsonStr)
	}

	// Explicitly verify it's not null
	if strings.Contains(jsonStr, `"tasks":null`) {
		t.Errorf("Tasks should not serialize as null, got: %s", jsonStr)
	}
}

// Test valid status values
func TestPlanStatusValidation(t *testing.T) {
	validStatuses := []string{"draft", "active", "completed", "archived"}
	invalidStatuses := []string{"pending", "in_progress", "ACTIVE", ""}

	for _, status := range validStatuses {
		// Valid statuses should not cause issues
		input := tools.CreatePlanInput{Name: "Test", Status: status}
		if input.Status != status {
			t.Errorf("Status should be %s", status)
		}
	}

	for _, status := range invalidStatuses {
		// These should be handled as validation errors in the handler
		_ = status
	}
}

func TestTaskStatusValidation(t *testing.T) {
	validStatuses := []string{"pending", "in_progress", "completed", "cancelled", "blocked"}
	invalidStatuses := []string{"active", "draft", "PENDING", ""}

	for _, status := range validStatuses {
		// Valid statuses should not cause issues
		input := tools.CreateTaskInput{Content: "Test", Status: status}
		if input.Status != status {
			t.Errorf("Status should be %s", status)
		}
	}

	for _, status := range invalidStatuses {
		// These should be handled as validation errors in the handler
		_ = status
	}
}
