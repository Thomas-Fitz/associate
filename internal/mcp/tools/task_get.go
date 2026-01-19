package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetTaskInput defines the input for the get_task tool.
type GetTaskInput struct {
	ID string `json:"id" jsonschema:"required,The ID of the task to retrieve"`
}

// PlanReference contains summary info about a plan the task belongs to.
type PlanReference struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// GetTaskOutput defines the output for the get_task tool.
type GetTaskOutput struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	Status    string            `json:"status"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	Plans     []PlanReference   `json:"plans,omitempty"`
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
}

// GetTaskTool returns the tool definition for get_task.
func GetTaskTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_task",
		Description: "Retrieve a task by ID, including the plans it belongs to. Returns full task details with plan references.",
	}
}

// HandleGetTask handles the get_task tool call.
func (h *Handler) HandleGetTask(ctx context.Context, req *mcp.CallToolRequest, input GetTaskInput) (*mcp.CallToolResult, GetTaskOutput, error) {
	h.Logger.Info("get_task", "id", input.ID)

	if input.ID == "" {
		return nil, GetTaskOutput{}, fmt.Errorf("id is required")
	}

	task, plans, err := h.TaskRepo.GetWithPlans(ctx, input.ID)
	if err != nil {
		h.Logger.Error("get_task failed", "id", input.ID, "error", err)
		return nil, GetTaskOutput{}, fmt.Errorf("failed to get task: %w", err)
	}

	if task == nil {
		return nil, GetTaskOutput{}, fmt.Errorf("task not found: %s", input.ID)
	}

	// Convert plans to references
	var planRefs []PlanReference
	for _, p := range plans {
		planRefs = append(planRefs, PlanReference{
			ID:     p.ID,
			Name:   p.Name,
			Status: string(p.Status),
		})
	}

	h.Logger.Info("get_task complete", "id", task.ID, "plans", len(planRefs))
	return nil, GetTaskOutput{
		ID:        task.ID,
		Content:   task.Content,
		Status:    string(task.Status),
		Metadata:  task.Metadata,
		Tags:      task.Tags,
		Plans:     planRefs,
		CreatedAt: task.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: task.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}
