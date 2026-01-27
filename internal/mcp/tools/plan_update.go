package tools

import (
	"context"
	"fmt"

	"github.com/Thomas-Fitz/associate/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// UpdatePlanInput defines the input for the update_plan tool.
type UpdatePlanInput struct {
	ID          string         `json:"id" jsonschema:"required,The ID of the plan to update"`
	Name        *string        `json:"name,omitempty" jsonschema:"New name for the plan"`
	Description *string        `json:"description,omitempty" jsonschema:"New description for the plan"`
	Status      *string        `json:"status,omitempty" jsonschema:"New status: draft, active, completed, archived"`
	Metadata    map[string]any `json:"metadata,omitempty" jsonschema:"New metadata (replaces existing)"`
	Tags        []string       `json:"tags,omitempty" jsonschema:"New tags (replaces existing)"`
	RelatedTo   []string       `json:"related_to,omitempty" jsonschema:"IDs of nodes to connect using RELATES_TO"`
	References  []string       `json:"references,omitempty" jsonschema:"IDs of nodes to connect using REFERENCES"`
}

// UpdatePlanOutput defines the output for the update_plan tool.
type UpdatePlanOutput struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Status      string            `json:"status"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	UpdatedAt   string            `json:"updated_at"`
}

// UpdatePlanTool returns the tool definition for update_plan.
func UpdatePlanTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "update_plan",
		Description: "Update an existing plan. Only provided fields are updated. Can update name, description, status, metadata, tags, and add new relationships.",
	}
}

// HandleUpdatePlan handles the update_plan tool call.
func (h *Handler) HandleUpdatePlan(ctx context.Context, req *mcp.CallToolRequest, input UpdatePlanInput) (*mcp.CallToolResult, UpdatePlanOutput, error) {
	h.Logger.Info("update_plan", "id", input.ID)

	if input.ID == "" {
		return nil, UpdatePlanOutput{}, fmt.Errorf("id is required")
	}

	// Validate status if provided
	var status *string
	if input.Status != nil {
		if !models.IsValidPlanStatus(*input.Status) {
			return nil, UpdatePlanOutput{}, fmt.Errorf("invalid status: %s (must be one of: draft, active, completed, archived)", *input.Status)
		}
		status = input.Status
	}

	// Convert metadata
	var metadata map[string]string
	if input.Metadata != nil {
		metadata = convertMetadata(input.Metadata)
	}

	// Build relationships
	rels := buildRelationships(
		input.RelatedTo, nil, input.References,
		nil, nil, nil, nil,
	)

	updated, err := h.PlanRepo.Update(ctx, input.ID, input.Name, input.Description, status, metadata, input.Tags, rels)
	if err != nil {
		h.Logger.Error("update_plan failed", "id", input.ID, "error", err)
		return nil, UpdatePlanOutput{}, fmt.Errorf("failed to update plan: %w", err)
	}

	h.Logger.Info("update_plan complete", "id", updated.ID)
	return nil, UpdatePlanOutput{
		ID:          updated.ID,
		Name:        updated.Name,
		Description: updated.Description,
		Status:      string(updated.Status),
		Metadata:    updated.Metadata,
		Tags:        updated.Tags,
		UpdatedAt:   updated.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}
