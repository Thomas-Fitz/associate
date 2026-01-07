package cmd

import (
"context"
"fmt"
"os"
"path/filepath"
"strings"

"github.com/fitz/associate/internal/config"
"github.com/fitz/associate/internal/graph"
"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
Use:   "init [path]",
Short: "Initialize a repository in the graph database",
Long: `Initialize a repository by registering it in the Neo4j graph database.

This creates a Repo node that will be used to store all architectural knowledge
and code patterns for this specific repository.

Path can be:
  - Absolute path: /Users/name/projects/myrepo
  - Relative path: ../myrepo
  - Current directory: . (default)`,
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

// Verify path exists
if _, err := os.Stat(absPath); os.IsNotExist(err) {
exitWithError(fmt.Errorf("path does not exist: %s", absPath))
}

// Get repository name from path
repoName := filepath.Base(absPath)

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

// Verify connectivity
ctx := context.Background()
if err := client.VerifyConnectivity(ctx); err != nil {
exitWithError(fmt.Errorf("failed to connect to Neo4j: %w\nEnsure the Neo4j container is running", err))
}

// Detect primary language (simple heuristic)
language := detectLanguage(absPath)

// Create repo node
repo := &graph.RepoNode{
Path:        absPath,
Name:        repoName,
Description: fmt.Sprintf("Repository initialized on %s", filepath.Base(absPath)),
Language:    language,
}

if err := client.CreateRepo(ctx, repo); err != nil {
exitWithError(fmt.Errorf("failed to initialize repository: %w", err))
}

fmt.Printf("✓ Initialized repository '%s'\n", repoName)
fmt.Printf("  Path: %s\n", absPath)
if language != "" {
fmt.Printf("  Language: %s\n", language)
}
fmt.Printf("\nRepository is now registered in the graph database.\n")
fmt.Printf("Use 'associate refresh-memory' to scan and index the codebase.\n")
},
}

func init() {
rootCmd.AddCommand(initCmd)
}

// detectLanguage attempts to detect the primary language of a repository.
func detectLanguage(path string) string {
// Check for common language indicators
indicators := map[string]string{
"go.mod":         "Go",
"package.json":   "JavaScript",
"Cargo.toml":     "Rust",
"pom.xml":        "Java",
"Gemfile":        "Ruby",
"requirements.txt": "Python",
"setup.py":       "Python",
"pyproject.toml": "Python",
}

for file, lang := range indicators {
if _, err := os.Stat(filepath.Join(path, file)); err == nil {
return lang
}
}

// Check for .git to at least confirm it's a repository
if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
// Try to detect from file extensions
entries, err := os.ReadDir(path)
if err != nil {
return ""
}

extCount := make(map[string]int)
for _, entry := range entries {
if entry.IsDir() {
continue
}
ext := strings.ToLower(filepath.Ext(entry.Name()))
if ext != "" {
extCount[ext]++
}
}

// Map extensions to languages
extLang := map[string]string{
".go":   "Go",
".js":   "JavaScript",
".ts":   "TypeScript",
".py":   "Python",
".rb":   "Ruby",
".rs":   "Rust",
".java": "Java",
".c":    "C",
".cpp":  "C++",
".cs":   "C#",
}

maxCount := 0
maxLang := ""
for ext, count := range extCount {
if lang, ok := extLang[ext]; ok && count > maxCount {
maxCount = count
maxLang = lang
}
}

return maxLang
}

return ""
}
