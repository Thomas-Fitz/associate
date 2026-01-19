package models

import (
	"testing"
	"time"
)

func TestPlanStatusConstants(t *testing.T) {
	tests := []struct {
		status   PlanStatus
		expected string
	}{
		{PlanStatusDraft, "draft"},
		{PlanStatusActive, "active"},
		{PlanStatusCompleted, "completed"},
		{PlanStatusArchived, "archived"},
	}

	for _, tc := range tests {
		if string(tc.status) != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, tc.status)
		}
	}
}

func TestIsValidPlanStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"draft", true},
		{"active", true},
		{"completed", true},
		{"archived", true},
		{"invalid", false},
		{"", false},
		{"ACTIVE", false}, // case sensitive
	}

	for _, tc := range tests {
		result := IsValidPlanStatus(tc.input)
		if result != tc.expected {
			t.Errorf("IsValidPlanStatus(%q) = %v, expected %v", tc.input, result, tc.expected)
		}
	}
}

func TestPlanStruct(t *testing.T) {
	now := time.Now().UTC()
	plan := Plan{
		ID:          "test-plan-id",
		Name:        "Test Plan",
		Description: "A test plan description",
		Status:      PlanStatusActive,
		Metadata:    map[string]string{"key": "value"},
		Tags:        []string{"test", "plan"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if plan.ID != "test-plan-id" {
		t.Errorf("expected ID 'test-plan-id', got %s", plan.ID)
	}
	if plan.Name != "Test Plan" {
		t.Errorf("expected Name 'Test Plan', got %s", plan.Name)
	}
	if plan.Description != "A test plan description" {
		t.Errorf("expected Description 'A test plan description', got %s", plan.Description)
	}
	if plan.Status != PlanStatusActive {
		t.Errorf("expected Status 'active', got %s", plan.Status)
	}
	if plan.Metadata["key"] != "value" {
		t.Errorf("expected Metadata[key] 'value', got %s", plan.Metadata["key"])
	}
	if len(plan.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(plan.Tags))
	}
}

func TestValidPlanStatuses(t *testing.T) {
	expected := 4
	if len(ValidPlanStatuses) != expected {
		t.Errorf("expected %d valid plan statuses, got %d", expected, len(ValidPlanStatuses))
	}
}
