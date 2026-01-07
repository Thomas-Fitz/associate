// Package cmd contains all CLI command definitions.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fitz/associate/internal/config"
	"github.com/fitz/associate/internal/docker"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "associate",
	Short: "Associate - Terminal AI Agent with Graph Memory",
	Long: `Associate is a terminal-based AI agent that wraps GitHub Copilot
and enhances it with persistent graph-based memory using Neo4j.

It maintains architectural knowledge, code patterns, and dependencies
in a graph database for intelligent context-aware assistance.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip Docker initialization for these commands
		skipCommands := map[string]bool{
			"completion": true,
			"help":       true,
			"config":     true,
			"set":        true,
			"get":        true,
			"list":       true,
		}
		
		if skipCommands[cmd.Name()] {
			return nil
		}

		return ensureNeo4jContainer(cmd)
	},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags can be added here
	rootCmd.PersistentFlags().StringP("dir", "d", ".", "Working directory for repository operations")
}

// exitWithError prints an error message and exits with code 1.
func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

// ensureNeo4jContainer ensures that the Neo4j Docker container is running.
func ensureNeo4jContainer(cmd *cobra.Command) error {
	dir, _ := cmd.Flags().GetString("dir")
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("invalid directory: %w", err)
	}

	// Load configuration
	cfg, err := config.Load(absDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w\nPlease run 'associate config set NEO4J_PASSWORD <password>' to configure", err)
	}

	// Prepare Docker container configuration
	containerCfg := &docker.ContainerConfig{
		Name:     cfg.ContainerName,
		Image:    cfg.Neo4jImage,
		URI:      cfg.Neo4jURI,
		Username: cfg.Neo4jUsername,
		Password: cfg.Neo4jPassword,
	}

	// Ensure container is running
	created, err := docker.EnsureContainer(containerCfg)
	if err != nil {
		return fmt.Errorf("failed to ensure Neo4j container: %w", err)
	}

	if created {
		fmt.Fprintf(os.Stderr, "✓ Created Neo4j container '%s'\n", containerCfg.Name)
		fmt.Fprintf(os.Stderr, "  Waiting for Neo4j to be ready...\n")
		
		// Wait for container to be ready
		if err := docker.WaitForContainer(containerCfg.Name, 30); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "  ✓ Neo4j is ready\n")
		}
	}

	return nil
}
