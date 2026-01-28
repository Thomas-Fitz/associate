package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DeleteZoneInput defines the input for the delete_zone tool.
type DeleteZoneInput struct {
	ID string `json:"id" jsonschema:"required,The ID of the zone to delete"`
}

// DeleteZoneOutput defines the output for the delete_zone tool.
type DeleteZoneOutput struct {
	ID              string `json:"id"`
	Deleted         bool   `json:"deleted"`
	PlansDeleted    int    `json:"plans_deleted"`
	TasksDeleted    int    `json:"tasks_deleted"`
	MemoriesDeleted int    `json:"memories_deleted"`
}

// DeleteZoneTool returns the tool definition for delete_zone.
func DeleteZoneTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "delete_zone",
		Description: "Delete a zone and cascade delete all its plans, tasks, and memories. WARNING: This is a destructive operation that cannot be undone.",
	}
}

// HandleDeleteZone handles the delete_zone tool call.
func (h *Handler) HandleDeleteZone(ctx context.Context, req *mcp.CallToolRequest, input DeleteZoneInput) (*mcp.CallToolResult, DeleteZoneOutput, error) {
	h.Logger.Info("delete_zone", "id", input.ID)

	if input.ID == "" {
		return nil, DeleteZoneOutput{}, fmt.Errorf("id is required")
	}

	plansDeleted, tasksDeleted, memoriesDeleted, err := h.ZoneRepo.Delete(ctx, input.ID)
	if err != nil {
		h.Logger.Error("delete_zone failed", "id", input.ID, "error", err)
		return nil, DeleteZoneOutput{}, fmt.Errorf("failed to delete zone: %w", err)
	}

	h.Logger.Info("delete_zone complete", "id", input.ID, "plans_deleted", plansDeleted, "tasks_deleted", tasksDeleted, "memories_deleted", memoriesDeleted)
	return nil, DeleteZoneOutput{
		ID:              input.ID,
		Deleted:         true,
		PlansDeleted:    plansDeleted,
		TasksDeleted:    tasksDeleted,
		MemoriesDeleted: memoriesDeleted,
	}, nil
}
