package tools

import (
	"context"
	"fmt"

	"github.com/fitz/associate/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateTaskInput defines the input for the create_task tool.
type CreateTaskInput struct {
	Content    string         `json:"content" jsonschema:"required,The content/description of the task"`
	PlanID     string         `json:"plan_id,omitempty" jsonschema:"ID of a plan to add this task to (creates PART_OF relationship)"`
	Status     string         `json:"status,omitempty" jsonschema:"Task status: pending, in_progress, completed, cancelled, blocked (default: pending)"`
	Metadata   map[string]any `json:"metadata,omitempty" jsonschema:"Key-value metadata to attach to the task"`
	Tags       []string       `json:"tags,omitempty" jsonschema:"Tags for categorizing the task"`
	DependsOn  []string       `json:"depends_on,omitempty" jsonschema:"IDs of tasks this depends on using DEPENDS_ON"`
	Blocks     []string       `json:"blocks,omitempty" jsonschema:"IDs of tasks this blocks using BLOCKS"`
	Follows    []string       `json:"follows,omitempty" jsonschema:"IDs of tasks this follows in sequence using FOLLOWS"`
	RelatedTo  []string       `json:"related_to,omitempty" jsonschema:"IDs of nodes to connect using RELATES_TO"`
	References []string       `json:"references,omitempty" jsonschema:"IDs of nodes this references using REFERENCES"`
}

// CreateTaskOutput defines the output for the create_task tool.
type CreateTaskOutput struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	Status    string            `json:"status"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	CreatedAt string            `json:"created_at"`
}

// CreateTaskTool returns the tool definition for create_task.
func CreateTaskTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "create_task",
		Description: "Create a new task with content, status, and optional plan association. Tasks can have dependencies (depends_on, blocks, follows) and other relationships. Returns the created task with its ID.",
	}
}

// HandleCreateTask handles the create_task tool call.
func (h *Handler) HandleCreateTask(ctx context.Context, req *mcp.CallToolRequest, input CreateTaskInput) (*mcp.CallToolResult, CreateTaskOutput, error) {
	h.Logger.Info("create_task", "content_len", len(input.Content), "plan_id", input.PlanID, "status", input.Status)

	if input.Content == "" {
		return nil, CreateTaskOutput{}, fmt.Errorf("content is required")
	}

	// Validate status if provided
	status := models.TaskStatusPending
	if input.Status != "" {
		if !models.IsValidTaskStatus(input.Status) {
			return nil, CreateTaskOutput{}, fmt.Errorf("invalid status: %s (must be one of: pending, in_progress, completed, cancelled, blocked)", input.Status)
		}
		status = models.TaskStatus(input.Status)
	}

	// Convert metadata
	metadata := convertMetadata(input.Metadata)

	task := models.Task{
		Content:  input.Content,
		Status:   status,
		Metadata: metadata,
		Tags:     input.Tags,
	}

	// Build plan IDs list
	var planIDs []string
	if input.PlanID != "" {
		planIDs = append(planIDs, input.PlanID)
	}

	// Build other relationships
	rels := buildRelationships(
		input.RelatedTo, nil, input.References,
		input.DependsOn, input.Blocks, input.Follows, nil,
	)

	created, err := h.TaskRepo.Add(ctx, task, planIDs, rels)
	if err != nil {
		h.Logger.Error("create_task failed", "error", err)
		return nil, CreateTaskOutput{}, fmt.Errorf("failed to create task: %w", err)
	}

	h.Logger.Info("create_task complete", "id", created.ID, "status", created.Status)
	return nil, CreateTaskOutput{
		ID:        created.ID,
		Content:   created.Content,
		Status:    string(created.Status),
		Metadata:  created.Metadata,
		Tags:      created.Tags,
		CreatedAt: created.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}
