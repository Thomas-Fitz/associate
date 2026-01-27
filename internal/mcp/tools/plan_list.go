package tools

import (
	"context"
	"fmt"

	"github.com/Thomas-Fitz/associate/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListPlansInput defines the input for the list_plans tool.
type ListPlansInput struct {
	Status string   `json:"status,omitempty" jsonschema:"Filter by status: draft, active, completed, archived"`
	Tags   []string `json:"tags,omitempty" jsonschema:"Filter by tags (plans matching any of the tags are returned)"`
	Limit  int      `json:"limit,omitempty" jsonschema:"Maximum number of plans to return (default: 50)"`
}

// PlanSummary contains summary info about a plan.
type PlanSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	UpdatedAt   string `json:"updated_at"`
}

// ListPlansOutput defines the output for the list_plans tool.
type ListPlansOutput struct {
	Plans []PlanSummary `json:"plans"`
	Count int           `json:"count"`
}

// ListPlansTool returns the tool definition for list_plans.
func ListPlansTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "list_plans",
		Description: "List plans with optional filtering by status and tags. Returns plan summaries ordered by most recently updated.",
	}
}

// HandleListPlans handles the list_plans tool call.
func (h *Handler) HandleListPlans(ctx context.Context, req *mcp.CallToolRequest, input ListPlansInput) (*mcp.CallToolResult, ListPlansOutput, error) {
	h.Logger.Info("list_plans", "status", input.Status, "tags", input.Tags, "limit", input.Limit)

	// Validate status if provided
	if input.Status != "" && !models.IsValidPlanStatus(input.Status) {
		return nil, ListPlansOutput{}, fmt.Errorf("invalid status: %s (must be one of: draft, active, completed, archived)", input.Status)
	}

	plans, err := h.PlanRepo.List(ctx, input.Status, input.Tags, input.Limit)
	if err != nil {
		h.Logger.Error("list_plans failed", "error", err)
		return nil, ListPlansOutput{}, fmt.Errorf("failed to list plans: %w", err)
	}

	// Convert to summaries
	// Initialize as empty slice (not nil) to ensure JSON serializes as [] not null
	summaries := make([]PlanSummary, 0, len(plans))
	for _, p := range plans {
		summaries = append(summaries, PlanSummary{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Status:      string(p.Status),
			UpdatedAt:   p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	h.Logger.Info("list_plans complete", "count", len(summaries))
	return nil, ListPlansOutput{
		Plans: summaries,
		Count: len(summaries),
	}, nil
}
