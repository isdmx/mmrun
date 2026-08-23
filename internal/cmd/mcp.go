package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/config"
	"github.com/isdmx/mmrun/internal/mcp"
)

func newMcpCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "mcp", Short: "MCP server for AI coding agents"}

	var toolsTier, team, skillsDir string
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server on stdio",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()
			s, err := mcp.New(mcp.ServerConfig{ToolsTier: toolsTier, SkillsDir: skillsDir, Team: team})
			if err != nil {
				return err
			}
			return s.Run(ctx)
		},
	}
	serveCmd.Flags().StringVar(&toolsTier, "tools", "read", "tools to expose: read, write, admin")
	serveCmd.Flags().StringVar(&team, "team", "", "default team for channel resolution")
	serveCmd.Flags().StringVar(&skillsDir, "skills-dir", "", "skills directory")
	cmd.AddCommand(serveCmd)

	var configTools, configTeam string
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Print MCP client config JSON for AI hosts",
		RunE: func(cmd *cobra.Command, args []string) error {
			base := []string{"mcp", "serve"}
			if configTools != "read" {
				base = append(base, "--tools", configTools)
			}
			if configTeam != "" {
				base = append(base, "--team", configTeam)
			}
			cfg := map[string]any{
				"mcpServers": map[string]any{"mmrun": map[string]any{"command": "mmrun", "args": base}},
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(cfg)
		},
	}
	configCmd.Flags().StringVar(&configTools, "tools", "read", "tools tier for the config snippet")
	configCmd.Flags().StringVar(&configTeam, "team", "", "default team for the config snippet")
	cmd.AddCommand(configCmd)

	var skillsDirFlag string
	skillsCmd := &cobra.Command{Use: "skills", Short: "Manage MCP skills"}

	skillsListCmd := &cobra.Command{
		Use: "list", Short: "Show available skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := mcp.ListSkills(skillsDirOrDefault(skillsDirFlag))
			if err != nil {
				return err
			}
			if len(names) == 0 {
				fmt.Println("No skills installed.")
				fmt.Println("Run 'mmrun mcp skills init' to install the default skill.")
				return nil
			}
			for _, name := range names {
				fmt.Println(name)
			}
			return nil
		},
	}
	skillsCmd.AddCommand(skillsListCmd)

	skillsInitCmd := &cobra.Command{
		Use: "init", Short: "Copy default skills to the skills directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mcp.InitSkills(skillsDirOrDefault(skillsDirFlag))
		},
	}
	skillsCmd.AddCommand(skillsInitCmd)
	cmd.AddCommand(skillsCmd)

	return cmd
}

func skillsDirOrDefault(dir string) string {
	if dir != "" {
		return dir
	}
	return filepath.Join(filepath.Dir(config.Paths().ConfigFile), "skills")
}
