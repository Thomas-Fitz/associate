package tools

import (
"context"
"fmt"

"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SearchInput defines the input for the search tool.
type SearchInput struct {
Query string `json:"query" jsonschema:"The search query to find matching memories"`
Limit int    `json:"limit,omitempty" jsonschema:"Maximum number of results to return (default 10)"`
}

// SearchOutput defines the output for the search tool.
type SearchOutput struct {
Results []SearchResultItem `json:"results"`
Count   int                `json:"count"`
}

// SearchResultItem represents a single search result.
type SearchResultItem struct {
ID        string            `json:"id"`
Type      string            `json:"type"`
Content   string            `json:"content"`
Score     float64           `json:"score"`
Metadata  map[string]string `json:"metadata,omitempty"`
Tags      []string          `json:"tags,omitempty"`
Related   []string          `json:"related,omitempty"`
CreatedAt string            `json:"created_at"`
UpdatedAt string            `json:"updated_at"`
}

// SearchTool returns the tool definition for search_memories.
func SearchTool() *mcp.Tool {
return &mcp.Tool{
Name:        "search_memories",
Description: "Search memories with parameters query (string) and limit (int, default 10). Returns matching results with: id, type (Note, Task, Project, Repository, Memory), content (string), score (float), metadata (json), tags (array), and related (array) memory IDs.",
}
}

// HandleSearch handles the search_memories tool call.
func (h *Handler) HandleSearch(ctx context.Context, req *mcp.CallToolRequest, input SearchInput) (*mcp.CallToolResult, SearchOutput, error) {
h.Logger.Info("search_memories", "query", input.Query, "limit", input.Limit)

results, err := h.Repo.Search(ctx, input.Query, input.Limit)
if err != nil {
h.Logger.Error("search_memories failed", "query", input.Query, "error", err)
return nil, SearchOutput{}, fmt.Errorf("search failed: %w", err)
}

output := SearchOutput{
Results: make([]SearchResultItem, len(results)),
Count:   len(results),
}

for i, r := range results {
output.Results[i] = SearchResultItem{
ID:        r.Memory.ID,
Type:      string(r.Memory.Type),
Content:   r.Memory.Content,
Score:     r.Score,
Metadata:  r.Memory.Metadata,
Tags:      r.Memory.Tags,
Related:   r.Related,
CreatedAt: r.Memory.CreatedAt.Format("2006-01-02T15:04:05Z"),
UpdatedAt: r.Memory.UpdatedAt.Format("2006-01-02T15:04:05Z"),
}
}

h.Logger.Info("search_memories complete", "results", output.Count)
return nil, output, nil
}
