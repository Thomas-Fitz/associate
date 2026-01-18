package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DeleteTaskInput defines the input for the delete_task tool.
type DeleteTaskInput struct {
	ID string `json:"id" jsonschema:"required,The ID of the task to delete"`
}

// DeleteTaskOutput defines the output for the delete_task tool.
type DeleteTaskOutput struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// DeleteTaskTool returns the tool definition for delete_task.
func DeleteTaskTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "delete_task",
		Description: "Delete a task and all its relationships. This action cannot be undone.",
	}
}

// HandleDeleteTask handles the delete_task tool call.
func (h *Handler) HandleDeleteTask(ctx context.Context, req *mcp.CallToolRequest, input DeleteTaskInput) (*mcp.CallToolResult, DeleteTaskOutput, error) {
	h.Logger.Info("delete_task", "id", input.ID)

	if input.ID == "" {
		return nil, DeleteTaskOutput{}, fmt.Errorf("id is required")
	}

	err := h.TaskRepo.Delete(ctx, input.ID)
	if err != nil {
		h.Logger.Error("delete_task failed", "id", input.ID, "error", err)
		return nil, DeleteTaskOutput{}, fmt.Errorf("failed to delete task: %w", err)
	}

	h.Logger.Info("delete_task complete", "id", input.ID)
	return nil, DeleteTaskOutput{
		ID:      input.ID,
		Deleted: true,
	}, nil
}
