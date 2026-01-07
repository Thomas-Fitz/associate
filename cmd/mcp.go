package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fitz/associate/internal/config"
	"github.com/fitz/associate/internal/graph"
	"github.com/fitz/associate/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp [path]",
	Short: "Start MCP server for AI agent integration",
	Long: `Start the Model Context Protocol (MCP) server that allows AI agents
to interact with the graph memory system via JSON-RPC over stdio.

This command is typically invoked by AI agents (like Copilot CLI) rather than
directly by users. It enables agents to:
  - Save memories and context to the graph
  - Search for relevant memories
  - Save and retrieve architectural learnings
  - Access repository-specific context (AGENTS.md)

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

		// Verify path exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			exitWithError(fmt.Errorf("path does not exist: %s", absPath))
		}

		// Load configuration (will use global fallback)
		cfg, err := config.Load(absPath)
		if err != nil {
			exitWithError(fmt.Errorf("failed to load config: %w\nPlease run 'associate config set NEO4J_PASSWORD <password>' to configure", err))
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

		// Verify repository is initialized
		_, err = client.GetRepo(ctx, absPath)
		if err != nil {
			exitWithError(fmt.Errorf("repository not initialized. Run 'associate init' first"))
		}

		// Create and start MCP server
		server := mcp.NewServer(client, absPath)
		
		// Log to stderr (stdout is for MCP protocol)
		fmt.Fprintf(os.Stderr, "MCP server started for repository: %s\n", absPath)
		
		if err := server.Start(ctx); err != nil {
			exitWithError(fmt.Errorf("MCP server error: %w", err))
		}
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
