package models

import (
	"testing"
	"time"
)

func TestZone_Struct(t *testing.T) {
	now := time.Now()
	zone := Zone{
		ID:          "test-zone-id",
		Name:        "Test Zone",
		Description: "A test zone description",
		Metadata:    map[string]string{"key": "value"},
		Tags:        []string{"tag1", "tag2"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if zone.ID != "test-zone-id" {
		t.Errorf("ID: got %s, want test-zone-id", zone.ID)
	}
	if zone.Name != "Test Zone" {
		t.Errorf("Name: got %s, want 'Test Zone'", zone.Name)
	}
	if zone.Description != "A test zone description" {
		t.Errorf("Description: got %s, want 'A test zone description'", zone.Description)
	}
	if len(zone.Tags) != 2 {
		t.Errorf("Tags length: got %d, want 2", len(zone.Tags))
	}
	if zone.Metadata["key"] != "value" {
		t.Errorf("Metadata[key]: got %s, want 'value'", zone.Metadata["key"])
	}
}

func TestZoneSearchResult_Struct(t *testing.T) {
	zsr := ZoneSearchResult{
		Zone:    Zone{ID: "test-zone"},
		Related: []string{"related-1", "related-2"},
	}

	if zsr.Zone.ID != "test-zone" {
		t.Errorf("Zone.ID: got %s, want test-zone", zsr.Zone.ID)
	}
	if len(zsr.Related) != 2 {
		t.Errorf("Related length: got %d, want 2", len(zsr.Related))
	}
}

func TestZoneWithCounts_Struct(t *testing.T) {
	zwc := ZoneWithCounts{
		Zone:        Zone{ID: "test-zone", Name: "Test Zone"},
		PlanCount:   5,
		TaskCount:   20,
		MemoryCount: 3,
	}

	if zwc.Zone.ID != "test-zone" {
		t.Errorf("Zone.ID: got %s, want test-zone", zwc.Zone.ID)
	}
	if zwc.PlanCount != 5 {
		t.Errorf("PlanCount: got %d, want 5", zwc.PlanCount)
	}
	if zwc.TaskCount != 20 {
		t.Errorf("TaskCount: got %d, want 20", zwc.TaskCount)
	}
	if zwc.MemoryCount != 3 {
		t.Errorf("MemoryCount: got %d, want 3", zwc.MemoryCount)
	}
}

func TestPlanInZone_Struct(t *testing.T) {
	piz := PlanInZone{
		Plan: Plan{ID: "plan-1", Name: "Test Plan"},
		Tasks: []TaskInPlan{
			{Task: Task{ID: "task-1", Content: "Task 1"}, Position: 1000},
			{Task: Task{ID: "task-2", Content: "Task 2"}, Position: 2000},
		},
	}

	if piz.Plan.ID != "plan-1" {
		t.Errorf("Plan.ID: got %s, want plan-1", piz.Plan.ID)
	}
	if len(piz.Tasks) != 2 {
		t.Errorf("Tasks length: got %d, want 2", len(piz.Tasks))
	}
	if piz.Tasks[0].Position != 1000 {
		t.Errorf("Tasks[0].Position: got %f, want 1000", piz.Tasks[0].Position)
	}
}

func TestMemoryInZone_Struct(t *testing.T) {
	now := time.Now()
	miz := MemoryInZone{
		Memory: Memory{
			ID:        "mem-1",
			Type:      TypeNote,
			Content:   "Memory content",
			Metadata:  map[string]string{"ui_x": "100", "ui_y": "200"},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	if miz.Memory.ID != "mem-1" {
		t.Errorf("Memory.ID: got %s, want mem-1", miz.Memory.ID)
	}
	if miz.Memory.Metadata["ui_x"] != "100" {
		t.Errorf("Memory.Metadata[ui_x]: got %s, want '100'", miz.Memory.Metadata["ui_x"])
	}
}

func TestZoneWithContents_Struct(t *testing.T) {
	now := time.Now()
	zwc := ZoneWithContents{
		Zone: Zone{ID: "zone-1", Name: "Test Zone", CreatedAt: now, UpdatedAt: now},
		Plans: []PlanInZone{
			{Plan: Plan{ID: "plan-1", Name: "Plan 1"}, Tasks: []TaskInPlan{}},
			{Plan: Plan{ID: "plan-2", Name: "Plan 2"}, Tasks: []TaskInPlan{}},
		},
		Memories: []MemoryInZone{
			{Memory: Memory{ID: "mem-1", Type: TypeNote, Content: "Note 1"}},
		},
	}

	if zwc.Zone.ID != "zone-1" {
		t.Errorf("Zone.ID: got %s, want zone-1", zwc.Zone.ID)
	}
	if len(zwc.Plans) != 2 {
		t.Errorf("Plans length: got %d, want 2", len(zwc.Plans))
	}
	if len(zwc.Memories) != 1 {
		t.Errorf("Memories length: got %d, want 1", len(zwc.Memories))
	}
}

func TestRelBelongsTo_Constant(t *testing.T) {
	// Verify the BELONGS_TO relationship type is defined correctly
	if string(RelBelongsTo) != "BELONGS_TO" {
		t.Errorf("RelBelongsTo: got %s, want BELONGS_TO", RelBelongsTo)
	}
}
