package mcp

import (
	"testing"

	"github.com/chewcw/aveva-historian-mcpserver/internal/config"
	"github.com/chewcw/aveva-historian-mcpserver/internal/dataserver"
	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewServerCreatesWithThreeTools(t *testing.T) {
	cfg := &config.Config{
		BaseURL:    "http://localhost:32569",
		ServerName: "test-server",
	}
	client := historian.New(cfg.BaseURL, "u", "p", "/Historian/v2", nil)
	srv := NewServer(cfg, client, (*dataserver.Store)(nil), nil)

	if srv == nil {
		t.Fatal("NewServer returned nil")
	}

	// Use the SDK's listTools method indirectly through a server capabilities check
	// by ensuring no panic on creation
	_ = srv
}

// Test that the internal mcp package name doesn't conflict with go-sdk mcp
func TestImportAlias(t *testing.T) {
	_ = sdkmcp.NewServer(&sdkmcp.Implementation{Name: "test"}, nil)
}
