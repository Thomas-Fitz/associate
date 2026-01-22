package models

import "time"

// TaskStatus defines the status of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusCancelled  TaskStatus = "cancelled"
	TaskStatusBlocked    TaskStatus = "blocked"
)

// ValidTaskStatuses contains all valid task status values
var ValidTaskStatuses = []TaskStatus{
	TaskStatusPending,
	TaskStatusInProgress,
	TaskStatusCompleted,
	TaskStatusCancelled,
	TaskStatusBlocked,
}

// IsValidTaskStatus checks if a status string is a valid TaskStatus
func IsValidTaskStatus(s string) bool {
	for _, status := range ValidTaskStatuses {
		if string(status) == s {
			return true
		}
	}
	return false
}

// Task represents a task node in the graph database.
// Tasks are actionable work items with status tracking.
type Task struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	Status    TaskStatus        `json:"status"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// TaskSearchResult contains a task with optional related information
type TaskSearchResult struct {
	Task    Task     `json:"task"`
	Related []string `json:"related,omitempty"`
}

// TaskInPlan represents a task with its position and dependencies within a specific plan
type TaskInPlan struct {
	Task      Task     `json:"task"`
	Position  float64  `json:"position"`
	DependsOn []string `json:"depends_on,omitempty"` // IDs of tasks this task depends on
	Blocks    []string `json:"blocks,omitempty"`     // IDs of tasks this task blocks
}

// TaskListResult represents a task in list results, with optional position when filtered by plan
type TaskListResult struct {
	Task     Task     `json:"task"`
	Position *float64 `json:"position,omitempty"` // Only set when listing tasks within a plan
}
