package cmd

import (
"context"
"fmt"
"path/filepath"
"strings"

"github.com/fitz/associate/internal/config"
"github.com/fitz/associate/internal/graph"
"github.com/spf13/cobra"
)

var saveMemoryCmd = &cobra.Command{
Use:   "save-memory [content]",
Short: "Save a memory or context note to the graph database",
Long: `Save a memory or context note to the graph database for the current repository.

This allows you to manually store important context, decisions, or notes that
AI agents can later retrieve.

Example:
  associate save-memory "Authentication uses JWT tokens with 15min expiry" \
    --type architectural_decision \
    --tags auth,security`,
Args: cobra.ExactArgs(1),
Run: func(cmd *cobra.Command, args []string) {
content := args[0]
contextType, _ := cmd.Flags().GetString("type")
tagsStr, _ := cmd.Flags().GetString("tags")
relatedTo, _ := cmd.Flags().GetString("related-to")

// Parse tags
var tags []string
if tagsStr != "" {
tags = strings.Split(tagsStr, ",")
for i, tag := range tags {
tags[i] = strings.TrimSpace(tag)
}
}

// Get absolute path
path := "."
absPath, err := filepath.Abs(path)
if err != nil {
exitWithError(fmt.Errorf("invalid path: %w", err))
}

// Load configuration
cfg, err := config.Load(absPath)
if err != nil {
exitWithError(fmt.Errorf("failed to load config: %w", err))
}

// Create graph client
client, err := graph.NewClient(cfg.Neo4jURI, cfg.Neo4jUsername, cfg.Neo4jPassword, cfg.Neo4jDatabase)
if err != nil {
exitWithError(fmt.Errorf("failed to create graph client: %w", err))
}
defer client.Close(context.Background())

ctx := context.Background()

// Verify repository exists
_, err = client.GetRepo(ctx, absPath)
if err != nil {
exitWithError(fmt.Errorf("repository not initialized. Run 'associate init' first"))
}

// Save memory
memory := &graph.MemoryNode{
Content:     content,
ContextType: contextType,
Tags:        tags,
RelatedTo:   relatedTo,
}

if err := client.SaveMemory(ctx, absPath, memory); err != nil {
exitWithError(fmt.Errorf("failed to save memory: %w", err))
}

fmt.Printf("✓ Memory saved successfully\n")
fmt.Printf("  Type: %s\n", contextType)
if len(tags) > 0 {
fmt.Printf("  Tags: %v\n", tags)
}
},
}

var searchMemoryCmd = &cobra.Command{
Use:   "search-memory [query]",
Short: "Search for memories in the graph database",
Long: `Search for memories in the graph database for the current repository.

You can search by content, filter by type and tags, and limit results.

Example:
  associate search-memory "authentication"
  associate search-memory --type architectural_decision
  associate search-memory --tags auth,security --limit 5`,
Args: cobra.MaximumNArgs(1),
Run: func(cmd *cobra.Command, args []string) {
query := ""
if len(args) > 0 {
query = args[0]
}

contextType, _ := cmd.Flags().GetString("type")
tagsStr, _ := cmd.Flags().GetString("tags")
limit, _ := cmd.Flags().GetInt("limit")

// Parse tags
var tags []string
if tagsStr != "" {
tags = strings.Split(tagsStr, ",")
for i, tag := range tags {
tags[i] = strings.TrimSpace(tag)
}
}

// Get absolute path
path := "."
absPath, err := filepath.Abs(path)
if err != nil {
exitWithError(fmt.Errorf("invalid path: %w", err))
}

// Load configuration
cfg, err := config.Load(absPath)
if err != nil {
exitWithError(fmt.Errorf("failed to load config: %w", err))
}

// Create graph client
client, err := graph.NewClient(cfg.Neo4jURI, cfg.Neo4jUsername, cfg.Neo4jPassword, cfg.Neo4jDatabase)
if err != nil {
exitWithError(fmt.Errorf("failed to create graph client: %w", err))
}
defer client.Close(context.Background())

ctx := context.Background()

// Verify repository exists
_, err = client.GetRepo(ctx, absPath)
if err != nil {
exitWithError(fmt.Errorf("repository not initialized. Run 'associate init' first"))
}

// Search memories
memories, err := client.SearchMemory(ctx, absPath, query, contextType, tags, limit)
if err != nil {
exitWithError(fmt.Errorf("failed to search memories: %w", err))
}

if len(memories) == 0 {
fmt.Println("No memories found matching the criteria.")
return
}

fmt.Printf("Found %d memory(ies):\n\n", len(memories))
for i, m := range memories {
fmt.Printf("%d. [%s] %s\n", i+1, m.ContextType, m.Content)
if len(m.Tags) > 0 {
fmt.Printf("   Tags: %v\n", m.Tags)
}
if m.RelatedTo != "" {
fmt.Printf("   Related: %s\n", m.RelatedTo)
}
fmt.Println()
}
},
}

func init() {
// save-memory flags
saveMemoryCmd.Flags().StringP("type", "t", "note", "Context type (architectural_decision, bug_fix, performance, note, etc.)")
saveMemoryCmd.Flags().String("tags", "", "Comma-separated tags")
saveMemoryCmd.Flags().String("related-to", "", "Related code path")

// search-memory flags
searchMemoryCmd.Flags().StringP("type", "t", "", "Filter by context type")
searchMemoryCmd.Flags().String("tags", "", "Comma-separated tags to filter by")
searchMemoryCmd.Flags().IntP("limit", "l", 10, "Maximum results to return")

rootCmd.AddCommand(saveMemoryCmd)
rootCmd.AddCommand(searchMemoryCmd)
}
