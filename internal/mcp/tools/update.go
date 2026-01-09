package tools

import (
"context"
"fmt"

"github.com/modelcontextprotocol/go-sdk/mcp"
)

// UpdateInput defines the input for the update tool.
// Pointer types for optional arrays allow clients to send null instead of omitting the field.
type UpdateInput struct {
ID         string         `json:"id" jsonschema:"ID of the memory to update"`
Content    *string        `json:"content,omitempty" jsonschema:"New content (if provided, replaces existing)"`
Metadata   map[string]any `json:"metadata,omitempty" jsonschema:"New metadata (if provided, replaces existing). Values can be strings or will be JSON-serialized."`
Tags       *[]string      `json:"tags,omitempty" jsonschema:"New tags (if provided, replaces existing)"`
RelatedTo  *[]string      `json:"related_to,omitempty" jsonschema:"IDs of memories to connect using RELATES_TO"`
PartOf     *[]string      `json:"part_of,omitempty" jsonschema:"IDs of memories to connect using PART_OF"`
References *[]string      `json:"references,omitempty" jsonschema:"IDs of memories to connect using REFERENCES"`
DependsOn  *[]string      `json:"depends_on,omitempty" jsonschema:"IDs of memories to connect using DEPENDS_ON"`
Blocks     *[]string      `json:"blocks,omitempty" jsonschema:"IDs of memories to connect using BLOCKS"`
Follows    *[]string      `json:"follows,omitempty" jsonschema:"IDs of memories to connect using FOLLOWS"`
Implements *[]string      `json:"implements,omitempty" jsonschema:"IDs of memories to connect using IMPLEMENTS"`
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
Description: "Update an existing memory's content, metadata, tags, or add new relationships. All fields except ID are optional - only provided fields are updated. Available relationship types to add: related_to (RELATES_TO), part_of (PART_OF), references (REFERENCES), depends_on (DEPENDS_ON), blocks (BLOCKS), follows (FOLLOWS), implements (IMPLEMENTS). Metadata values can be complex JSON objects that will be serialized. Note: This adds new relationships but does not remove existing ones. Returns the updated memory.",
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

// Dereference tags pointer safely
tags := derefSlice(input.Tags)

// Build relationships from pointer slices
rels := buildRelationships(
input.RelatedTo, input.PartOf, input.References,
input.DependsOn, input.Blocks, input.Follows, input.Implements,
)

updated, err := h.Repo.Update(ctx, input.ID, input.Content, metadata, tags, rels)
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
