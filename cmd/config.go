package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/fitz/associate/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration settings",
	Long:  `View and modify configuration settings for Associate.`,
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Long:  `Set a configuration value in the .env file (local or global).
	
Use --global flag to set in the global configuration (~/.associate/config).
Otherwise, sets in the local .env file.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]
		global, _ := cmd.Flags().GetBool("global")
		
		var err error
		if global {
			err = config.SetGlobalConfig(key, value)
			if err == nil {
				fmt.Printf("✓ Set %s (global)\n", key)
			}
		} else {
			dir, _ := cmd.Flags().GetString("dir")
			absDir, err := filepath.Abs(dir)
			if err != nil {
				exitWithError(fmt.Errorf("invalid directory: %w", err))
			}
			
			err = config.Set(absDir, key, value)
			if err == nil {
				fmt.Printf("✓ Set %s (local)\n", key)
			}
		}
		
		if err != nil {
			exitWithError(err)
		}
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Long:  `Retrieve a configuration value from the .env file.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir, _ := cmd.Flags().GetString("dir")
		absDir, err := filepath.Abs(dir)
		if err != nil {
			exitWithError(fmt.Errorf("invalid directory: %w", err))
		}

		key := args[0]
		value, err := config.Get(absDir, key)
		if err != nil {
			exitWithError(err)
		}

		fmt.Printf("%s=%s\n", key, value)
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	Long:  `Display all configuration values from the current .env file.`,
	Run: func(cmd *cobra.Command, args []string) {
		dir, _ := cmd.Flags().GetString("dir")
		absDir, err := filepath.Abs(dir)
		if err != nil {
			exitWithError(fmt.Errorf("invalid directory: %w", err))
		}

		cfg, err := config.Load(absDir)
		if err != nil {
			// If validation fails, still show what we can load
			fmt.Println("Configuration (some required values may be missing):")
		} else {
			fmt.Println("Configuration:")
		}

		// Display all values (masking sensitive ones)
		fmt.Printf("  NEO4J_URI: %s\n", cfg.Neo4jURI)
		fmt.Printf("  NEO4J_USERNAME: %s\n", cfg.Neo4jUsername)
		fmt.Printf("  NEO4J_PASSWORD: %s\n", maskPassword(cfg.Neo4jPassword))
		fmt.Printf("  NEO4J_DATABASE: %s\n", cfg.Neo4jDatabase)
		fmt.Printf("  NEO4J_IMAGE: %s\n", cfg.Neo4jImage)
		fmt.Printf("  NEO4J_CONTAINER_NAME: %s\n", cfg.ContainerName)
		if cfg.CopilotToken != "" {
			fmt.Printf("  GITHUB_COPILOT_TOKEN: %s\n", maskPassword(cfg.CopilotToken))
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configListCmd)
	
	// Add --global flag to config set
	configSetCmd.Flags().Bool("global", false, "Set in global config instead of local")
}

// maskPassword masks a password string for display.
func maskPassword(s string) string {
	if s == "" {
		return "(not set)"
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}
