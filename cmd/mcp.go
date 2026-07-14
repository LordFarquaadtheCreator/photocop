package cmd

import (
	"github.com/LordFarquaadtheCreator/photocop/internal/mcpserver"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run the photocop MCP server over stdio.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return mcpserver.Run()
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
