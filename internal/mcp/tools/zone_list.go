package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListZonesInput defines the input for the list_zones tool.
type ListZonesInput struct {
	Search string `json:"search,omitempty" jsonschema:"Search term to filter zones by name or description"`
	Limit  int    `json:"limit,omitempty" jsonschema:"Maximum number of zones to return (default: 50)"`
}

// ZoneSummary contains summary info about a zone.
type ZoneSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	PlanCount   int    `json:"plan_count"`
	TaskCount   int    `json:"task_count"`
	MemoryCount int    `json:"memory_count"`
	UpdatedAt   string `json:"updated_at"`
}

// ListZonesOutput defines the output for the list_zones tool.
type ListZonesOutput struct {
	Zones []ZoneSummary `json:"zones"`
	Count int           `json:"count"`
}

// ListZonesTool returns the tool definition for list_zones.
func ListZonesTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "list_zones",
		Description: "List zones with optional search filtering. Returns zone summaries with counts of plans, tasks, and memories, ordered by most recently updated.",
	}
}

// HandleListZones handles the list_zones tool call.
func (h *Handler) HandleListZones(ctx context.Context, req *mcp.CallToolRequest, input ListZonesInput) (*mcp.CallToolResult, ListZonesOutput, error) {
	h.Logger.Info("list_zones", "search", input.Search, "limit", input.Limit)

	zones, err := h.ZoneRepo.List(ctx, input.Search, input.Limit)
	if err != nil {
		h.Logger.Error("list_zones failed", "error", err)
		return nil, ListZonesOutput{}, fmt.Errorf("failed to list zones: %w", err)
	}

	// Convert to summaries
	// Initialize as empty slice (not nil) to ensure JSON serializes as [] not null
	summaries := make([]ZoneSummary, 0, len(zones))
	for _, z := range zones {
		summaries = append(summaries, ZoneSummary{
			ID:          z.Zone.ID,
			Name:        z.Zone.Name,
			Description: z.Zone.Description,
			PlanCount:   z.PlanCount,
			TaskCount:   z.TaskCount,
			MemoryCount: z.MemoryCount,
			UpdatedAt:   z.Zone.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	h.Logger.Info("list_zones complete", "count", len(summaries))
	return nil, ListZonesOutput{
		Zones: summaries,
		Count: len(summaries),
	}, nil
}
