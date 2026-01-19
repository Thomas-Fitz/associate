package models

import (
	"testing"
	"time"
)

func TestTaskStatusConstants(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected string
	}{
		{TaskStatusPending, "pending"},
		{TaskStatusInProgress, "in_progress"},
		{TaskStatusCompleted, "completed"},
		{TaskStatusCancelled, "cancelled"},
		{TaskStatusBlocked, "blocked"},
	}

	for _, tc := range tests {
		if string(tc.status) != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, tc.status)
		}
	}
}

func TestIsValidTaskStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"pending", true},
		{"in_progress", true},
		{"completed", true},
		{"cancelled", true},
		{"blocked", true},
		{"invalid", false},
		{"", false},
		{"PENDING", false},     // case sensitive
		{"in-progress", false}, // wrong format
	}

	for _, tc := range tests {
		result := IsValidTaskStatus(tc.input)
		if result != tc.expected {
			t.Errorf("IsValidTaskStatus(%q) = %v, expected %v", tc.input, result, tc.expected)
		}
	}
}

func TestTaskStruct(t *testing.T) {
	now := time.Now().UTC()
	task := Task{
		ID:        "test-task-id",
		Content:   "Test task content",
		Status:    TaskStatusPending,
		Metadata:  map[string]string{"priority": "high"},
		Tags:      []string{"test", "task"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if task.ID != "test-task-id" {
		t.Errorf("expected ID 'test-task-id', got %s", task.ID)
	}
	if task.Content != "Test task content" {
		t.Errorf("expected Content 'Test task content', got %s", task.Content)
	}
	if task.Status != TaskStatusPending {
		t.Errorf("expected Status 'pending', got %s", task.Status)
	}
	if task.Metadata["priority"] != "high" {
		t.Errorf("expected Metadata[priority] 'high', got %s", task.Metadata["priority"])
	}
	if len(task.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(task.Tags))
	}
}

func TestValidTaskStatuses(t *testing.T) {
	expected := 5
	if len(ValidTaskStatuses) != expected {
		t.Errorf("expected %d valid task statuses, got %d", expected, len(ValidTaskStatuses))
	}
}
