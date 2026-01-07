package cmd

import (
"bufio"
"context"
"fmt"
"os"
"path/filepath"
"strings"

"github.com/fitz/associate/internal/config"
"github.com/fitz/associate/internal/graph"
"github.com/spf13/cobra"
)

var refreshMemoryCmd = &cobra.Command{
Use:   "refresh-memory [path]",
Short: "Refresh the graph memory by scanning the codebase",
Long: `Refresh the graph memory by scanning the codebase and updating nodes.

This command will:
  - Compare the current codebase structure with the stored graph
  - Add new code elements discovered
  - Update changed elements
  - Remove elements that no longer exist

Path defaults to the current directory if not specified.`,
Args: cobra.MaximumNArgs(1),
Run: func(cmd *cobra.Command, args []string) {
// Determine the path
path := "."
if len(args) > 0 {
path = args[0]
}

// Get absolute path
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

// Verify repository exists in graph
_, err = client.GetRepo(ctx, absPath)
if err != nil {
exitWithError(fmt.Errorf("repository not initialized. Run 'associate init' first"))
}

fmt.Printf("Refreshing memory for: %s\n", absPath)
fmt.Printf("Scanning codebase...\n")

// TODO: Implement actual code scanning logic
// For now, this is a placeholder that demonstrates the concept
fmt.Printf("✓ Memory refresh complete\n")
fmt.Printf("\nNote: Full code scanning will be implemented in the next phase.\n")
},
}

var resetMemoryCmd = &cobra.Command{
Use:   "reset-memory [path]",
Short: "Reset the graph memory for a repository",
Long: `Reset (clear) all graph memory for a repository and rebuild from scratch.

WARNING: This will delete ALL stored knowledge about the repository including:
  - Code structure nodes
  - Architectural patterns
  - Dependencies
  - All relationships

This operation cannot be undone. You will be prompted for confirmation.`,
Args: cobra.MaximumNArgs(1),
Run: func(cmd *cobra.Command, args []string) {
// Determine the path
path := "."
if len(args) > 0 {
path = args[0]
}

// Get absolute path
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
repo, err := client.GetRepo(ctx, absPath)
if err != nil {
exitWithError(fmt.Errorf("repository not found in graph database"))
}

// Confirmation prompt
fmt.Printf("⚠️  WARNING: This will permanently delete all memory for repository '%s'\n", repo.Name)
fmt.Printf("   Path: %s\n\n", absPath)
fmt.Printf("Are you sure you want to continue? (yes/no): ")

reader := bufio.NewReader(os.Stdin)
response, err := reader.ReadString('\n')
if err != nil {
exitWithError(fmt.Errorf("failed to read confirmation: %w", err))
}

response = strings.TrimSpace(strings.ToLower(response))
if response != "yes" && response != "y" {
fmt.Println("Reset cancelled.")
return
}

fmt.Printf("\nDeleting memory for '%s'...\n", repo.Name)

// Delete the repository and all its nodes
if err := client.DeleteRepo(ctx, absPath); err != nil {
exitWithError(fmt.Errorf("failed to reset memory: %w", err))
}

fmt.Printf("✓ Memory reset complete\n")
fmt.Printf("\nRun 'associate init' to re-initialize the repository.\n")
},
}

func init() {
rootCmd.AddCommand(refreshMemoryCmd)
rootCmd.AddCommand(resetMemoryCmd)
}
