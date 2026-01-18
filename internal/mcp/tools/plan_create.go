package tools

import (
	"context"
	"fmt"

	"github.com/fitz/associate/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreatePlanInput defines the input for the create_plan tool.
type CreatePlanInput struct {
	Name        string         `json:"name" jsonschema:"required,The name/title of the plan"`
	Description string         `json:"description,omitempty" jsonschema:"A detailed description of the plan"`
	Status      string         `json:"status,omitempty" jsonschema:"Plan status: draft, active, completed, archived (default: active)"`
	Metadata    map[string]any `json:"metadata,omitempty" jsonschema:"Key-value metadata to attach to the plan"`
	Tags        []string       `json:"tags,omitempty" jsonschema:"Tags for categorizing the plan"`
	RelatedTo   []string       `json:"related_to,omitempty" jsonschema:"IDs of existing nodes to connect using RELATES_TO"`
	References  []string       `json:"references,omitempty" jsonschema:"IDs of existing nodes this references using REFERENCES"`
}

// CreatePlanOutput defines the output for the create_plan tool.
type CreatePlanOutput struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Status      string            `json:"status"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	CreatedAt   string            `json:"created_at"`
}

// CreatePlanTool returns the tool definition for create_plan.
func CreatePlanTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "create_plan",
		Description: "Create a new plan to organize tasks. Plans have a name, description, status (draft/active/completed/archived), metadata, and tags. Returns the created plan with its ID.",
	}
}

// HandleCreatePlan handles the create_plan tool call.
func (h *Handler) HandleCreatePlan(ctx context.Context, req *mcp.CallToolRequest, input CreatePlanInput) (*mcp.CallToolResult, CreatePlanOutput, error) {
	h.Logger.Info("create_plan", "name", input.Name, "status", input.Status)

	// Validate status if provided
	status := models.PlanStatusActive
	if input.Status != "" {
		if !models.IsValidPlanStatus(input.Status) {
			return nil, CreatePlanOutput{}, fmt.Errorf("invalid status: %s (must be one of: draft, active, completed, archived)", input.Status)
		}
		status = models.PlanStatus(input.Status)
	}

	// Convert metadata
	metadata := convertMetadata(input.Metadata)

	plan := models.Plan{
		Name:        input.Name,
		Description: input.Description,
		Status:      status,
		Metadata:    metadata,
		Tags:        input.Tags,
	}

	// Build relationships
	rels := buildRelationships(
		input.RelatedTo, nil, input.References,
		nil, nil, nil, nil,
	)

	created, err := h.PlanRepo.Add(ctx, plan, rels)
	if err != nil {
		h.Logger.Error("create_plan failed", "name", input.Name, "error", err)
		return nil, CreatePlanOutput{}, fmt.Errorf("failed to create plan: %w", err)
	}

	h.Logger.Info("create_plan complete", "id", created.ID, "name", created.Name)
	return nil, CreatePlanOutput{
		ID:          created.ID,
		Name:        created.Name,
		Description: created.Description,
		Status:      string(created.Status),
		Metadata:    created.Metadata,
		Tags:        created.Tags,
		CreatedAt:   created.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}
