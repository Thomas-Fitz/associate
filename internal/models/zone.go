package models

import "time"

// Zone represents a workspace container in the graph database.
// Zones organize Plans, Tasks, and Memories into dedicated work areas.
type Zone struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// ZoneSearchResult contains a zone with optional related information
type ZoneSearchResult struct {
	Zone    Zone     `json:"zone"`
	Related []string `json:"related,omitempty"`
}

// ZoneWithCounts contains zone data along with counts of contained items
type ZoneWithCounts struct {
	Zone        Zone `json:"zone"`
	PlanCount   int  `json:"plan_count"`
	TaskCount   int  `json:"task_count"`
	MemoryCount int  `json:"memory_count"`
}

// PlanInZone represents a plan with its tasks within a zone context
type PlanInZone struct {
	Plan  Plan         `json:"plan"`
	Tasks []TaskInPlan `json:"tasks"`
}

// MemoryInZone represents a memory within a zone context with UI positioning
type MemoryInZone struct {
	Memory Memory `json:"memory"`
	// UI positioning is stored in Memory.Metadata as ui_x, ui_y, ui_width, ui_height
}

// ZoneWithContents contains a zone with all its plans, tasks, and memories
type ZoneWithContents struct {
	Zone     Zone           `json:"zone"`
	Plans    []PlanInZone   `json:"plans"`
	Memories []MemoryInZone `json:"memories"`
}
