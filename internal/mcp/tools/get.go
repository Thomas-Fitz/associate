package tools

import (
"context"
"fmt"

"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetInput defines the input for the get tool.
type GetInput struct {
ID string `json:"id" jsonschema:"The ID of the memory to retrieve"`
}

// GetOutput defines the output for the get tool.
type GetOutput struct {
ID        string            `json:"id"`
Type      string            `json:"type"`
Content   string            `json:"content"`
Metadata  map[string]string `json:"metadata,omitempty"`
Tags      []string          `json:"tags,omitempty"`
Related   []RelatedMemory   `json:"related,omitempty"`
CreatedAt string            `json:"created_at"`
UpdatedAt string            `json:"updated_at"`
}

// GetTool returns the tool definition for get_memory.
func GetTool() *mcp.Tool {
return &mcp.Tool{
Name:        "get_memory",
Description: "Get a single memory by ID with full details including all relationships. Returns: id, type (Note|Task|Project|Repository|Memory), content, metadata, tags, related memories with their relationship types and direction (incoming|outgoing), created_at, and updated_at timestamps.",
}
}

// HandleGet handles the get_memory tool call.
func (h *Handler) HandleGet(ctx context.Context, req *mcp.CallToolRequest, input GetInput) (*mcp.CallToolResult, GetOutput, error) {
h.Logger.Info("get_memory", "id", input.ID)

mem, related, err := h.Repo.GetByIDWithRelated(ctx, input.ID)
if err != nil {
h.Logger.Error("get_memory failed", "id", input.ID, "error", err)
return nil, GetOutput{}, fmt.Errorf("failed to get memory: %w", err)
}
if mem == nil {
h.Logger.Warn("get_memory not found", "id", input.ID)
return nil, GetOutput{}, fmt.Errorf("memory not found: %s", input.ID)
}

output := GetOutput{
ID:        mem.ID,
Type:      string(mem.Type),
Content:   mem.Content,
Metadata:  mem.Metadata,
Tags:      mem.Tags,
CreatedAt: mem.CreatedAt.Format("2006-01-02T15:04:05Z"),
UpdatedAt: mem.UpdatedAt.Format("2006-01-02T15:04:05Z"),
}

for _, r := range related {
output.Related = append(output.Related, RelatedMemory{
ID:           r.ID,
Type:         string(r.Type),
RelationType: r.RelationType,
Direction:    r.Direction,
})
}

h.Logger.Info("get_memory complete", "id", mem.ID, "type", mem.Type, "related", len(related))
return nil, output, nil
}
