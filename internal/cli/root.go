package cli

import (
	"github.com/spf13/cobra"
)

// version is the build version, injected at build time:
// go build -ldflags "-X github.com/chewcw/aveva-historian-mcpserver/internal/cli.version=v1.2.3"
var version = "dev"

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "aveva-historian-mcpserver",
		Short:         "AVEVA Historian MCP server",
		SilenceUsage:  true,
		SilenceErrors: true,
		// Bare invocation runs serve with all flags at defaults.
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(cmd.Context(), serveOptions{})
		},
	}
	root.AddCommand(newServeCmd(), newVersionCmd(), NewHashCmd())
	return root
}

func Execute() error {
	return NewRootCmd().Execute()
}
