package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DeletePlanInput defines the input for the delete_plan tool.
type DeletePlanInput struct {
	ID string `json:"id" jsonschema:"required,The ID of the plan to delete"`
}

// DeletePlanOutput defines the output for the delete_plan tool.
type DeletePlanOutput struct {
	ID           string `json:"id"`
	Deleted      bool   `json:"deleted"`
	TasksDeleted int    `json:"tasks_deleted"`
}

// DeletePlanTool returns the tool definition for delete_plan.
func DeletePlanTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "delete_plan",
		Description: "Delete a plan and cascade delete tasks that only belong to this plan. Tasks that are PART_OF other plans are preserved (only the relationship to this plan is removed).",
	}
}

// HandleDeletePlan handles the delete_plan tool call.
func (h *Handler) HandleDeletePlan(ctx context.Context, req *mcp.CallToolRequest, input DeletePlanInput) (*mcp.CallToolResult, DeletePlanOutput, error) {
	h.Logger.Info("delete_plan", "id", input.ID)

	if input.ID == "" {
		return nil, DeletePlanOutput{}, fmt.Errorf("id is required")
	}

	tasksDeleted, err := h.PlanRepo.Delete(ctx, input.ID)
	if err != nil {
		h.Logger.Error("delete_plan failed", "id", input.ID, "error", err)
		return nil, DeletePlanOutput{}, fmt.Errorf("failed to delete plan: %w", err)
	}

	h.Logger.Info("delete_plan complete", "id", input.ID, "tasks_deleted", tasksDeleted)
	return nil, DeletePlanOutput{
		ID:           input.ID,
		Deleted:      true,
		TasksDeleted: tasksDeleted,
	}, nil
}
