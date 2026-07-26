package mcp

import (
	"log/slog"

	"github.com/chewcw/aveva-historian-mcpserver/internal/config"
	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/mcp/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer creates an MCP server with the three historian tools registered.
func NewServer(cfg *config.Config, client *historian.Client, logger *slog.Logger) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: cfg.ServerName}, nil)
	tools.RegisterGetTags(server, client, logger)
	tools.RegisterReadTrends(server, client, logger)
	tools.RegisterReadAnalogSummary(server, client, logger)
	return server
}
