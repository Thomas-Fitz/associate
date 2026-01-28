package tools

import (
	"context"
	"fmt"

	"github.com/Thomas-Fitz/associate/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateZoneInput defines the input for the create_zone tool.
type CreateZoneInput struct {
	Name        string         `json:"name" jsonschema:"required,The name/title of the zone"`
	Description string         `json:"description,omitempty" jsonschema:"A detailed description of the zone"`
	Metadata    map[string]any `json:"metadata,omitempty" jsonschema:"Key-value metadata to attach to the zone"`
	Tags        []string       `json:"tags,omitempty" jsonschema:"Tags for categorizing the zone"`
}

// CreateZoneOutput defines the output for the create_zone tool.
type CreateZoneOutput struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	CreatedAt   string            `json:"created_at"`
}

// CreateZoneTool returns the tool definition for create_zone.
func CreateZoneTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "create_zone",
		Description: "Create a new zone to organize plans and memories. Zones are top-level workspaces that contain related plans, tasks, and memories. Returns the created zone with its ID.",
	}
}

// HandleCreateZone handles the create_zone tool call.
func (h *Handler) HandleCreateZone(ctx context.Context, req *mcp.CallToolRequest, input CreateZoneInput) (*mcp.CallToolResult, CreateZoneOutput, error) {
	h.Logger.Info("create_zone", "name", input.Name)

	if input.Name == "" {
		return nil, CreateZoneOutput{}, fmt.Errorf("name is required")
	}

	// Convert metadata
	metadata := convertMetadata(input.Metadata)

	zone := models.Zone{
		Name:        input.Name,
		Description: input.Description,
		Metadata:    metadata,
		Tags:        input.Tags,
	}

	created, err := h.ZoneRepo.Add(ctx, zone)
	if err != nil {
		h.Logger.Error("create_zone failed", "name", input.Name, "error", err)
		return nil, CreateZoneOutput{}, fmt.Errorf("failed to create zone: %w", err)
	}

	h.Logger.Info("create_zone complete", "id", created.ID, "name", created.Name)
	return nil, CreateZoneOutput{
		ID:          created.ID,
		Name:        created.Name,
		Description: created.Description,
		Metadata:    created.Metadata,
		Tags:        created.Tags,
		CreatedAt:   created.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}
