package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetPlanInput defines the input for the get_plan tool.
type GetPlanInput struct {
	ID string `json:"id" jsonschema:"required,The ID of the plan to retrieve"`
}

// TaskSummary contains summary info about a task in a plan.
type TaskSummary struct {
	ID        string   `json:"id"`
	Content   string   `json:"content"`
	Status    string   `json:"status"`
	Position  float64  `json:"position"`
	DependsOn []string `json:"depends_on,omitempty"`
	Blocks    []string `json:"blocks,omitempty"`
}

// GetPlanOutput defines the output for the get_plan tool.
type GetPlanOutput struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Status      string            `json:"status"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Tasks       []TaskSummary     `json:"tasks,omitempty"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

// GetPlanTool returns the tool definition for get_plan.
func GetPlanTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_plan",
		Description: "Retrieve a plan by ID, including all its tasks. Returns full plan details with task summaries (id, content, status).",
	}
}

// HandleGetPlan handles the get_plan tool call.
func (h *Handler) HandleGetPlan(ctx context.Context, req *mcp.CallToolRequest, input GetPlanInput) (*mcp.CallToolResult, GetPlanOutput, error) {
	h.Logger.Info("get_plan", "id", input.ID)

	if input.ID == "" {
		return nil, GetPlanOutput{}, fmt.Errorf("id is required")
	}

	plan, tasks, err := h.PlanRepo.GetWithTasks(ctx, input.ID)
	if err != nil {
		h.Logger.Error("get_plan failed", "id", input.ID, "error", err)
		return nil, GetPlanOutput{}, fmt.Errorf("failed to get plan: %w", err)
	}

	if plan == nil {
		return nil, GetPlanOutput{}, fmt.Errorf("plan not found: %s", input.ID)
	}

	// Convert tasks to summaries (tasks are already ordered by position from repository)
	var taskSummaries []TaskSummary
	for _, t := range tasks {
		taskSummaries = append(taskSummaries, TaskSummary{
			ID:        t.Task.ID,
			Content:   t.Task.Content,
			Status:    string(t.Task.Status),
			Position:  t.Position,
			DependsOn: t.DependsOn,
			Blocks:    t.Blocks,
		})
	}

	h.Logger.Info("get_plan complete", "id", plan.ID, "tasks", len(taskSummaries))
	return nil, GetPlanOutput{
		ID:          plan.ID,
		Name:        plan.Name,
		Description: plan.Description,
		Status:      string(plan.Status),
		Metadata:    plan.Metadata,
		Tags:        plan.Tags,
		Tasks:       taskSummaries,
		CreatedAt:   plan.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   plan.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}
