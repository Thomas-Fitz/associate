package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetZoneInput defines the input for the get_zone tool.
type GetZoneInput struct {
	ID           string `json:"id" jsonschema:"required,The ID of the zone to retrieve"`
	IncludePlans bool   `json:"include_plans,omitempty" jsonschema:"If true, include all plans with their tasks"`
}

// ZonePlanSummary contains summary info about a plan in a zone, including tasks.
type ZonePlanSummary struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Status      string        `json:"status"`
	Tasks       []TaskSummary `json:"tasks,omitempty"`
}

// MemorySummary contains summary info about a memory in a zone.
type MemorySummary struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

// GetZoneOutput defines the output for the get_zone tool.
type GetZoneOutput struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Plans       []ZonePlanSummary `json:"plans,omitempty"`
	Memories    []MemorySummary   `json:"memories,omitempty"`
	PlanCount   int               `json:"plan_count"`
	TaskCount   int               `json:"task_count"`
	MemoryCount int               `json:"memory_count"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

// GetZoneTool returns the tool definition for get_zone.
func GetZoneTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_zone",
		Description: "Retrieve a zone by ID. Optionally includes all plans with their tasks. Returns full zone details with counts.",
	}
}

// HandleGetZone handles the get_zone tool call.
func (h *Handler) HandleGetZone(ctx context.Context, req *mcp.CallToolRequest, input GetZoneInput) (*mcp.CallToolResult, GetZoneOutput, error) {
	h.Logger.Info("get_zone", "id", input.ID, "include_plans", input.IncludePlans)

	if input.ID == "" {
		return nil, GetZoneOutput{}, fmt.Errorf("id is required")
	}

	if input.IncludePlans {
		// Get zone with all contents
		zwc, err := h.ZoneRepo.GetWithContents(ctx, input.ID)
		if err != nil {
			h.Logger.Error("get_zone failed", "id", input.ID, "error", err)
			return nil, GetZoneOutput{}, fmt.Errorf("failed to get zone: %w", err)
		}

		if zwc == nil {
			return nil, GetZoneOutput{}, fmt.Errorf("zone not found: %s", input.ID)
		}

		// Convert to output
		var plans []ZonePlanSummary
		taskCount := 0
		for _, piz := range zwc.Plans {
			var tasks []TaskSummary
			for _, tip := range piz.Tasks {
				tasks = append(tasks, TaskSummary{
					ID:        tip.Task.ID,
					Content:   tip.Task.Content,
					Status:    string(tip.Task.Status),
					Position:  tip.Position,
					DependsOn: tip.DependsOn,
					Blocks:    tip.Blocks,
				})
			}
			taskCount += len(piz.Tasks)
			plans = append(plans, ZonePlanSummary{
				ID:          piz.Plan.ID,
				Name:        piz.Plan.Name,
				Description: piz.Plan.Description,
				Status:      string(piz.Plan.Status),
				Tasks:       tasks,
			})
		}

		var memories []MemorySummary
		for _, miz := range zwc.Memories {
			memories = append(memories, MemorySummary{
				ID:      miz.Memory.ID,
				Type:    string(miz.Memory.Type),
				Content: miz.Memory.Content,
			})
		}

		h.Logger.Info("get_zone complete", "id", zwc.Zone.ID, "plans", len(plans), "tasks", taskCount, "memories", len(memories))
		return nil, GetZoneOutput{
			ID:          zwc.Zone.ID,
			Name:        zwc.Zone.Name,
			Description: zwc.Zone.Description,
			Metadata:    zwc.Zone.Metadata,
			Tags:        zwc.Zone.Tags,
			Plans:       plans,
			Memories:    memories,
			PlanCount:   len(plans),
			TaskCount:   taskCount,
			MemoryCount: len(memories),
			CreatedAt:   zwc.Zone.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   zwc.Zone.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}, nil
	}

	// Get just the zone with counts
	zone, err := h.ZoneRepo.GetByID(ctx, input.ID)
	if err != nil {
		h.Logger.Error("get_zone failed", "id", input.ID, "error", err)
		return nil, GetZoneOutput{}, fmt.Errorf("failed to get zone: %w", err)
	}

	if zone == nil {
		return nil, GetZoneOutput{}, fmt.Errorf("zone not found: %s", input.ID)
	}

	// Get counts using List with the zone ID filter
	zones, err := h.ZoneRepo.List(ctx, "", 1)
	planCount, taskCount, memoryCount := 0, 0, 0
	if err == nil {
		for _, z := range zones {
			if z.Zone.ID == input.ID {
				planCount = z.PlanCount
				taskCount = z.TaskCount
				memoryCount = z.MemoryCount
				break
			}
		}
	}

	h.Logger.Info("get_zone complete", "id", zone.ID)
	return nil, GetZoneOutput{
		ID:          zone.ID,
		Name:        zone.Name,
		Description: zone.Description,
		Metadata:    zone.Metadata,
		Tags:        zone.Tags,
		PlanCount:   planCount,
		TaskCount:   taskCount,
		MemoryCount: memoryCount,
		CreatedAt:   zone.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   zone.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}
