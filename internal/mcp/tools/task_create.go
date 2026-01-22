package tools

import (
	"context"
	"fmt"

	"github.com/fitz/associate/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateTaskInput defines the input for the create_task tool.
type CreateTaskInput struct {
	Content      string         `json:"content" jsonschema:"required,The content/description of the task"`
	PlanIDs      []string       `json:"plan_ids" jsonschema:"required,IDs of plans this task belongs to (at least one required, creates PART_OF relationships)"`
	Status       string         `json:"status,omitempty" jsonschema:"Task status: pending, in_progress, completed, cancelled, blocked (default: pending)"`
	Metadata     map[string]any `json:"metadata,omitempty" jsonschema:"Key-value metadata to attach to the task"`
	Tags         []string       `json:"tags,omitempty" jsonschema:"Tags for categorizing the task"`
	AfterTaskID  *string        `json:"after_task_id,omitempty" jsonschema:"ID of task to position this task after (within each plan). If not specified, appends to end."`
	BeforeTaskID *string        `json:"before_task_id,omitempty" jsonschema:"ID of task to position this task before (within each plan). Takes precedence for positioning if both after and before are specified."`
	DependsOn    []string       `json:"depends_on,omitempty" jsonschema:"IDs of tasks this depends on using DEPENDS_ON"`
	Blocks       []string       `json:"blocks,omitempty" jsonschema:"IDs of tasks this blocks using BLOCKS"`
	Follows      []string       `json:"follows,omitempty" jsonschema:"IDs of tasks this follows in sequence using FOLLOWS"`
	RelatedTo    []string       `json:"related_to,omitempty" jsonschema:"IDs of nodes to connect using RELATES_TO"`
	References   []string       `json:"references,omitempty" jsonschema:"IDs of nodes this references using REFERENCES"`
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
		Description: "Create a new task that belongs to one or more plans. Tasks must be associated with at least one plan via plan_ids. Supports dependencies (depends_on, blocks, follows) and other relationships. Returns the created task with its ID.",
	}
}

// HandleCreateTask handles the create_task tool call.
func (h *Handler) HandleCreateTask(ctx context.Context, req *mcp.CallToolRequest, input CreateTaskInput) (*mcp.CallToolResult, CreateTaskOutput, error) {
	h.Logger.Info("create_task", "content_len", len(input.Content), "plan_ids", input.PlanIDs, "status", input.Status)

	if input.Content == "" {
		return nil, CreateTaskOutput{}, fmt.Errorf("content is required")
	}

	// Validate plan_ids - at least one plan is required
	if len(input.PlanIDs) == 0 {
		return nil, CreateTaskOutput{}, fmt.Errorf("plan_ids is required: task must belong to at least one plan")
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

	// Build other relationships
	rels := buildRelationships(
		input.RelatedTo, nil, input.References,
		input.DependsOn, input.Blocks, input.Follows, nil,
	)

	created, err := h.TaskRepo.Add(ctx, task, input.PlanIDs, rels, input.AfterTaskID, input.BeforeTaskID)
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
