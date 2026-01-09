package tools

import (
"context"
"fmt"

"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetRelatedInput defines the input for the get_related tool.
type GetRelatedInput struct {
ID           string `json:"id" jsonschema:"The ID of the memory to get related nodes for"`
RelationType string `json:"relationship_type,omitempty" jsonschema:"Filter by relationship type (RELATES_TO, PART_OF, REFERENCES, DEPENDS_ON, BLOCKS, FOLLOWS, IMPLEMENTS)"`
Direction    string `json:"direction,omitempty" jsonschema:"Filter by direction: incoming, outgoing, or both (default: both)"`
Depth        int    `json:"depth,omitempty" jsonschema:"How many relationship hops to traverse (default: 1, max: 5)"`
}

// GetRelatedOutput defines the output for the get_related tool.
type GetRelatedOutput struct {
ID      string              `json:"id"`
Related []RelatedMemoryFull `json:"related"`
Count   int                 `json:"count"`
}

// GetRelatedTool returns the tool definition for get_related.
func GetRelatedTool() *mcp.Tool {
return &mcp.Tool{
Name:        "get_related",
Description: "Traverse the graph to find memories connected to a given node. Filter by relationship_type: RELATES_TO (general connection), PART_OF (hierarchical), REFERENCES (citation), DEPENDS_ON (dependency), BLOCKS (task gating), FOLLOWS (sequence), IMPLEMENTS (code-to-decision). Filter by direction: incoming (edges pointing to this node), outgoing (edges from this node), or both (default). Specify depth for multi-hop traversal (default: 1, max: 5). Returns full memory details for all related nodes including their relationship type, direction, and depth in the traversal.",
}
}

// HandleGetRelated handles the get_related tool call.
func (h *Handler) HandleGetRelated(ctx context.Context, req *mcp.CallToolRequest, input GetRelatedInput) (*mcp.CallToolResult, GetRelatedOutput, error) {
h.Logger.Info("get_related", "id", input.ID, "rel_type", input.RelationType, "direction", input.Direction, "depth", input.Depth)

// Set defaults
depth := input.Depth
if depth <= 0 {
depth = 1
}
if depth > 5 {
depth = 5
}

direction := input.Direction
if direction == "" {
direction = "both"
}

related, err := h.Repo.GetRelated(ctx, input.ID, input.RelationType, direction, depth)
if err != nil {
h.Logger.Error("get_related failed", "id", input.ID, "error", err)
return nil, GetRelatedOutput{}, fmt.Errorf("failed to get related memories: %w", err)
}

output := GetRelatedOutput{
ID:      input.ID,
Related: make([]RelatedMemoryFull, len(related)),
Count:   len(related),
}

for i, r := range related {
output.Related[i] = RelatedMemoryFull{
ID:           r.Memory.ID,
Type:         string(r.Memory.Type),
Content:      r.Memory.Content,
Metadata:     r.Memory.Metadata,
Tags:         r.Memory.Tags,
RelationType: r.RelationType,
Direction:    r.Direction,
Depth:        r.Depth,
CreatedAt:    r.Memory.CreatedAt.Format("2006-01-02T15:04:05Z"),
UpdatedAt:    r.Memory.UpdatedAt.Format("2006-01-02T15:04:05Z"),
}
}

h.Logger.Info("get_related complete", "id", input.ID, "results", output.Count)
return nil, output, nil
}
