package models

import "time"

// PlanStatus defines the status of a plan
type PlanStatus string

const (
	PlanStatusDraft     PlanStatus = "draft"
	PlanStatusActive    PlanStatus = "active"
	PlanStatusCompleted PlanStatus = "completed"
	PlanStatusArchived  PlanStatus = "archived"
)

// ValidPlanStatuses contains all valid plan status values
var ValidPlanStatuses = []PlanStatus{
	PlanStatusDraft,
	PlanStatusActive,
	PlanStatusCompleted,
	PlanStatusArchived,
}

// IsValidPlanStatus checks if a status string is a valid PlanStatus
func IsValidPlanStatus(s string) bool {
	for _, status := range ValidPlanStatuses {
		if string(status) == s {
			return true
		}
	}
	return false
}

// Plan represents a plan node in the graph database.
// Plans are containers for organizing tasks with status tracking.
type Plan struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Status      PlanStatus        `json:"status"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// PlanSearchResult contains a plan with optional related information
type PlanSearchResult struct {
	Plan    Plan     `json:"plan"`
	Related []string `json:"related,omitempty"`
}
