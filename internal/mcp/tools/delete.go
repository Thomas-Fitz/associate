package tools

import (
"context"
"fmt"

"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DeleteInput defines the input for the delete tool.
type DeleteInput struct {
ID string `json:"id" jsonschema:"The ID of the memory to delete"`
}

// DeleteOutput defines the output for the delete tool.
type DeleteOutput struct {
ID      string `json:"id"`
Deleted bool   `json:"deleted"`
}

// DeleteTool returns the tool definition for delete_memory.
func DeleteTool() *mcp.Tool {
return &mcp.Tool{
Name:        "delete_memory",
Description: "Permanently delete a memory and all its relationships from the graph database. This operation cannot be undone. Related memories are not deleted, only the relationships connecting them. Returns the deleted ID and confirmation boolean.",
}
}

// HandleDelete handles the delete_memory tool call.
func (h *Handler) HandleDelete(ctx context.Context, req *mcp.CallToolRequest, input DeleteInput) (*mcp.CallToolResult, DeleteOutput, error) {
h.Logger.Info("delete_memory", "id", input.ID)

err := h.Repo.Delete(ctx, input.ID)
if err != nil {
h.Logger.Error("delete_memory failed", "id", input.ID, "error", err)
return nil, DeleteOutput{ID: input.ID, Deleted: false}, fmt.Errorf("failed to delete memory: %w", err)
}

h.Logger.Info("delete_memory complete", "id", input.ID)
return nil, DeleteOutput{
ID:      input.ID,
Deleted: true,
}, nil
}
