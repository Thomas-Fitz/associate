package tools

import (
	"context"
	"fmt"

	"github.com/fitz/associate/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// UpdateTaskInput defines the input for the update_task tool.
type UpdateTaskInput struct {
	ID         string         `json:"id" jsonschema:"required,The ID of the task to update"`
	Content    *string        `json:"content,omitempty" jsonschema:"New content for the task"`
	Status     *string        `json:"status,omitempty" jsonschema:"New status: pending, in_progress, completed, cancelled, blocked"`
	Metadata   map[string]any `json:"metadata,omitempty" jsonschema:"New metadata (replaces existing)"`
	Tags       []string       `json:"tags,omitempty" jsonschema:"New tags (replaces existing)"`
	PlanIDs    []string       `json:"plan_ids,omitempty" jsonschema:"IDs of plans to add this task to (creates PART_OF relationships)"`
	DependsOn  []string       `json:"depends_on,omitempty" jsonschema:"IDs of tasks to add DEPENDS_ON relationships to"`
	Blocks     []string       `json:"blocks,omitempty" jsonschema:"IDs of tasks to add BLOCKS relationships to"`
	Follows    []string       `json:"follows,omitempty" jsonschema:"IDs of tasks to add FOLLOWS relationships to"`
	RelatedTo  []string       `json:"related_to,omitempty" jsonschema:"IDs of nodes to connect using RELATES_TO"`
	References []string       `json:"references,omitempty" jsonschema:"IDs of nodes to connect using REFERENCES"`
}

// UpdateTaskOutput defines the output for the update_task tool.
type UpdateTaskOutput struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	Status    string            `json:"status"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	UpdatedAt string            `json:"updated_at"`
}

// UpdateTaskTool returns the tool definition for update_task.
func UpdateTaskTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "update_task",
		Description: "Update an existing task. Only provided fields are updated. Can update content, status, metadata, tags, add to plans, and add new relationships.",
	}
}

// HandleUpdateTask handles the update_task tool call.
func (h *Handler) HandleUpdateTask(ctx context.Context, req *mcp.CallToolRequest, input UpdateTaskInput) (*mcp.CallToolResult, UpdateTaskOutput, error) {
	h.Logger.Info("update_task", "id", input.ID)

	if input.ID == "" {
		return nil, UpdateTaskOutput{}, fmt.Errorf("id is required")
	}

	// Validate status if provided
	var status *string
	if input.Status != nil {
		if !models.IsValidTaskStatus(*input.Status) {
			return nil, UpdateTaskOutput{}, fmt.Errorf("invalid status: %s (must be one of: pending, in_progress, completed, cancelled, blocked)", *input.Status)
		}
		status = input.Status
	}

	// Convert metadata
	var metadata map[string]string
	if input.Metadata != nil {
		metadata = convertMetadata(input.Metadata)
	}

	// Build other relationships
	rels := buildRelationships(
		input.RelatedTo, nil, input.References,
		input.DependsOn, input.Blocks, input.Follows, nil,
	)

	updated, err := h.TaskRepo.Update(ctx, input.ID, input.Content, status, metadata, input.Tags, input.PlanIDs, rels)
	if err != nil {
		h.Logger.Error("update_task failed", "id", input.ID, "error", err)
		return nil, UpdateTaskOutput{}, fmt.Errorf("failed to update task: %w", err)
	}

	h.Logger.Info("update_task complete", "id", updated.ID, "status", updated.Status)
	return nil, UpdateTaskOutput{
		ID:        updated.ID,
		Content:   updated.Content,
		Status:    string(updated.Status),
		Metadata:  updated.Metadata,
		Tags:      updated.Tags,
		UpdatedAt: updated.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}
