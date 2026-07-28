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
	if logger == nil {
		logger = slog.Default()
	}
	server := mcp.NewServer(&mcp.Implementation{Name: cfg.ServerName}, nil)
	logger = logger.With("source", "mcp.Server")
	limit := cfg.ResultByteLimit
	tools.RegisterGetTags(server, client, logger, limit)
	tools.RegisterReadProcessValues(server, client, logger, limit)
	tools.RegisterReadAnalogSummary(server, client, logger, limit)
	return server
}
