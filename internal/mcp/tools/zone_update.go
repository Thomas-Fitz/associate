package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// UpdateZoneInput defines the input for the update_zone tool.
type UpdateZoneInput struct {
	ID          string         `json:"id" jsonschema:"required,The ID of the zone to update"`
	Name        *string        `json:"name,omitempty" jsonschema:"New name for the zone"`
	Description *string        `json:"description,omitempty" jsonschema:"New description for the zone"`
	Metadata    map[string]any `json:"metadata,omitempty" jsonschema:"New metadata (replaces existing)"`
	Tags        []string       `json:"tags,omitempty" jsonschema:"New tags (replaces existing)"`
}

// UpdateZoneOutput defines the output for the update_zone tool.
type UpdateZoneOutput struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	UpdatedAt   string            `json:"updated_at"`
}

// UpdateZoneTool returns the tool definition for update_zone.
func UpdateZoneTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "update_zone",
		Description: "Update an existing zone. Only provided fields are updated. Can update name, description, metadata, and tags.",
	}
}

// HandleUpdateZone handles the update_zone tool call.
func (h *Handler) HandleUpdateZone(ctx context.Context, req *mcp.CallToolRequest, input UpdateZoneInput) (*mcp.CallToolResult, UpdateZoneOutput, error) {
	h.Logger.Info("update_zone", "id", input.ID)

	if input.ID == "" {
		return nil, UpdateZoneOutput{}, fmt.Errorf("id is required")
	}

	// Convert metadata
	var metadata map[string]string
	if input.Metadata != nil {
		metadata = convertMetadata(input.Metadata)
	}

	updated, err := h.ZoneRepo.Update(ctx, input.ID, input.Name, input.Description, metadata, input.Tags)
	if err != nil {
		h.Logger.Error("update_zone failed", "id", input.ID, "error", err)
		return nil, UpdateZoneOutput{}, fmt.Errorf("failed to update zone: %w", err)
	}

	h.Logger.Info("update_zone complete", "id", updated.ID)
	return nil, UpdateZoneOutput{
		ID:          updated.ID,
		Name:        updated.Name,
		Description: updated.Description,
		Metadata:    updated.Metadata,
		Tags:        updated.Tags,
		UpdatedAt:   updated.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}
