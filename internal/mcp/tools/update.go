package tools

import (
"context"
"fmt"

"github.com/modelcontextprotocol/go-sdk/mcp"
)

// UpdateInput defines the input for the update tool.
type UpdateInput struct {
ID         string         `json:"id" jsonschema:"ID of the memory to update"`
Content    *string        `json:"content,omitempty" jsonschema:"New content (if provided, replaces existing)"`
Metadata   map[string]any `json:"metadata,omitempty" jsonschema:"New metadata (if provided, replaces existing). Values can be strings or will be JSON-serialized."`
Tags       []string       `json:"tags,omitempty" jsonschema:"New tags (if provided, replaces existing)"`
RelatedTo  []string       `json:"related_to,omitempty" jsonschema:"IDs of memories to connect using RELATES_TO"`
PartOf     []string       `json:"part_of,omitempty" jsonschema:"IDs of memories to connect using PART_OF"`
References []string       `json:"references,omitempty" jsonschema:"IDs of memories to connect using REFERENCES"`
DependsOn  []string       `json:"depends_on,omitempty" jsonschema:"IDs of memories to connect using DEPENDS_ON"`
Blocks     []string       `json:"blocks,omitempty" jsonschema:"IDs of memories to connect using BLOCKS"`
Follows    []string       `json:"follows,omitempty" jsonschema:"IDs of memories to connect using FOLLOWS"`
Implements []string       `json:"implements,omitempty" jsonschema:"IDs of memories to connect using IMPLEMENTS"`
}

// UpdateOutput defines the output for the update tool.
type UpdateOutput struct {
ID        string            `json:"id"`
Type      string            `json:"type"`
Content   string            `json:"content"`
Metadata  map[string]string `json:"metadata,omitempty"`
Tags      []string          `json:"tags,omitempty"`
UpdatedAt string            `json:"updated_at"`
}

// UpdateTool returns the tool definition for update_memory.
func UpdateTool() *mcp.Tool {
return &mcp.Tool{
Name:        "update_memory",
Description: "Update an existing memory by id (string). Modify content (string), metadata (json), or tags (array). Add new relationships by passing arrays of existing memory IDs to: \"related_to\", \"part_of\", \"references\", \"depends_on\", \"blocks\", \"follows\", or \"implements\". Returns updated memory with updated_at timestamp.",
}
}

// HandleUpdate handles the update_memory tool call.
func (h *Handler) HandleUpdate(ctx context.Context, req *mcp.CallToolRequest, input UpdateInput) (*mcp.CallToolResult, UpdateOutput, error) {
h.Logger.Info("update_memory", "id", input.ID)

// Convert metadata values to strings (serialize complex values as JSON)
var metadata map[string]string
if input.Metadata != nil {
metadata = convertMetadata(input.Metadata)
}

// Build relationships from slices
rels := buildRelationships(
input.RelatedTo, input.PartOf, input.References,
input.DependsOn, input.Blocks, input.Follows, input.Implements,
)

updated, err := h.Repo.Update(ctx, input.ID, input.Content, metadata, input.Tags, rels)
if err != nil {
h.Logger.Error("update_memory failed", "id", input.ID, "error", err)
return nil, UpdateOutput{}, fmt.Errorf("failed to update memory: %w", err)
}

h.Logger.Info("update_memory complete", "id", updated.ID, "new_relationships", len(rels))
return nil, UpdateOutput{
ID:        updated.ID,
Type:      string(updated.Type),
Content:   updated.Content,
Metadata:  updated.Metadata,
Tags:      updated.Tags,
UpdatedAt: updated.UpdatedAt.Format("2006-01-02T15:04:05Z"),
}, nil
}
