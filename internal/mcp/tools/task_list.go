package tools

import (
	"context"
	"fmt"

	"github.com/fitz/associate/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListTasksInput defines the input for the list_tasks tool.
type ListTasksInput struct {
	PlanID string   `json:"plan_id,omitempty" jsonschema:"Filter by plan ID (only tasks belonging to this plan)"`
	Status string   `json:"status,omitempty" jsonschema:"Filter by status: pending, in_progress, completed, cancelled, blocked"`
	Tags   []string `json:"tags,omitempty" jsonschema:"Filter by tags (tasks matching any of the tags are returned)"`
	Limit  int      `json:"limit,omitempty" jsonschema:"Maximum number of tasks to return (default: 50)"`
}

// ListTasksOutput defines the output for the list_tasks tool.
type ListTasksOutput struct {
	Tasks []TaskSummary `json:"tasks"`
	Count int           `json:"count"`
}

// ListTasksTool returns the tool definition for list_tasks.
func ListTasksTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "list_tasks",
		Description: "List tasks with optional filtering by plan_id, status, and tags. Returns task summaries ordered by most recently updated.",
	}
}

// HandleListTasks handles the list_tasks tool call.
func (h *Handler) HandleListTasks(ctx context.Context, req *mcp.CallToolRequest, input ListTasksInput) (*mcp.CallToolResult, ListTasksOutput, error) {
	h.Logger.Info("list_tasks", "plan_id", input.PlanID, "status", input.Status, "tags", input.Tags, "limit", input.Limit)

	// Validate status if provided
	if input.Status != "" && !models.IsValidTaskStatus(input.Status) {
		return nil, ListTasksOutput{}, fmt.Errorf("invalid status: %s (must be one of: pending, in_progress, completed, cancelled, blocked)", input.Status)
	}

	tasks, err := h.TaskRepo.List(ctx, input.PlanID, input.Status, input.Tags, input.Limit)
	if err != nil {
		h.Logger.Error("list_tasks failed", "error", err)
		return nil, ListTasksOutput{}, fmt.Errorf("failed to list tasks: %w", err)
	}

	// Convert to summaries
	// Initialize as empty slice (not nil) to ensure JSON serializes as [] not null
	summaries := make([]TaskSummary, 0, len(tasks))
	for _, t := range tasks {
		summaries = append(summaries, TaskSummary{
			ID:      t.ID,
			Content: t.Content,
			Status:  string(t.Status),
		})
	}

	h.Logger.Info("list_tasks complete", "count", len(summaries))
	return nil, ListTasksOutput{
		Tasks: summaries,
		Count: len(summaries),
	}, nil
}
