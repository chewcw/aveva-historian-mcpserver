package mcp

import (
	"fmt"
	"log/slog"

	"github.com/chewcw/aveva-historian-mcpserver/internal/config"
	"github.com/chewcw/aveva-historian-mcpserver/internal/dataserver"
	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/mcp/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer creates an MCP server with the three historian tools registered.
func NewServer(cfg *config.Config, client *historian.Client, store *dataserver.Store, logger *slog.Logger) *mcp.Server {
	if logger == nil {
		logger = slog.Default()
	}
	server := mcp.NewServer(&mcp.Implementation{Name: cfg.ServerName}, nil)
	logger = logger.With("source", "mcp.Server")
	limit := cfg.ResultByteLimit
	dataServerBaseURL := fmt.Sprintf("http://%s:%d", cfg.DataServerBind, cfg.DataServerPort)
	tools.RegisterGetTags(server, client, store, dataServerBaseURL, logger, limit)
	tools.RegisterReadProcessValues(server, client, store, dataServerBaseURL, logger, limit)
	tools.RegisterReadAnalogSummary(server, client, store, dataServerBaseURL, logger, limit)
	return server
}
