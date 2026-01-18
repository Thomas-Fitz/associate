package tools

import (
	"context"
	"fmt"

	"github.com/fitz/associate/internal/models"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// AddInput defines the input for the add tool.
// Metadata accepts any JSON values; non-string values are serialized to JSON strings.
type AddInput struct {
	Content    string         `json:"content" jsonschema:"The content of the memory to store"`
	Type       string         `json:"type,omitempty" jsonschema:"Type of memory: Note, Repository, or Memory (default). For tasks use create_task, for plans use create_plan."`
	Metadata   map[string]any `json:"metadata,omitempty" jsonschema:"Key-value metadata to attach to the memory. Values can be strings or will be JSON-serialized."`
	Tags       []string       `json:"tags,omitempty" jsonschema:"Tags for categorizing the memory"`
	RelatedTo  []string       `json:"related_to,omitempty" jsonschema:"IDs of existing nodes to connect to using RELATES_TO"`
	PartOf     []string       `json:"part_of,omitempty" jsonschema:"IDs of existing nodes this is part of using PART_OF"`
	References []string       `json:"references,omitempty" jsonschema:"IDs of existing nodes this references using REFERENCES"`
	DependsOn  []string       `json:"depends_on,omitempty" jsonschema:"IDs of existing nodes this depends on using DEPENDS_ON"`
	Blocks     []string       `json:"blocks,omitempty" jsonschema:"IDs of existing nodes this blocks using BLOCKS"`
	Follows    []string       `json:"follows,omitempty" jsonschema:"IDs of existing nodes this follows in sequence using FOLLOWS"`
	Implements []string       `json:"implements,omitempty" jsonschema:"IDs of existing nodes this implements using IMPLEMENTS"`
}

// AddOutput defines the output for the add tool.
type AddOutput struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	CreatedAt string            `json:"created_at"`
}

// AddTool returns the tool definition for add_memory.
func AddTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "add_memory",
		Description: "Create a memory (type: Note, Repository, or Memory) for storing knowledge and observations. For actionable work items use create_task, for organizing tasks use create_plan. Memories can link to other nodes (memories, plans, tasks) via relationships.",
	}
}

// HandleAdd handles the add_memory tool call.
func (h *Handler) HandleAdd(ctx context.Context, req *mcp.CallToolRequest, input AddInput) (*mcp.CallToolResult, AddOutput, error) {
	h.Logger.Info("add_memory", "type", input.Type, "content_len", len(input.Content))

	// Convert metadata values to strings (serialize complex values as JSON)
	metadata := convertMetadata(input.Metadata)

	mem := models.Memory{
		Content:  input.Content,
		Type:     models.MemoryType(input.Type),
		Metadata: metadata,
		Tags:     input.Tags,
	}

	// Build relationships from slices
	rels := buildRelationships(
		input.RelatedTo, input.PartOf, input.References,
		input.DependsOn, input.Blocks, input.Follows, input.Implements,
	)

	created, err := h.Repo.Add(ctx, mem, rels)
	if err != nil {
		h.Logger.Error("add_memory failed", "type", input.Type, "error", err)
		return nil, AddOutput{}, fmt.Errorf("failed to add memory: %w", err)
	}

	h.Logger.Info("add_memory complete", "id", created.ID, "type", created.Type, "relationships", len(rels))
	return nil, AddOutput{
		ID:        created.ID,
		Type:      string(created.Type),
		Content:   created.Content,
		Metadata:  created.Metadata,
		Tags:      created.Tags,
		CreatedAt: created.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}
