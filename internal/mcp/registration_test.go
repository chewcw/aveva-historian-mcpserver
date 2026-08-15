package mcp

import (
	"context"
	"testing"

	"github.com/chewcw/aveva-historian-mcpserver/internal/config"
	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewServerRegistersEightPrompts(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:32569", ServerName: "test-server"}
	client := historian.New(cfg.BaseURL, "u", "p", "/Historian/v2", nil)
	srv := NewServer(cfg, client, nil, nil)

	ct, st := sdkmcp.NewInMemoryTransports()
	if _, err := srv.Connect(context.Background(), st, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	c := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test-client"}, nil)
	cs, err := c.Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	res, err := cs.ListPrompts(context.Background(), &sdkmcp.ListPromptsParams{})
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}
	want := []string{"current_status", "tag_profiler", "tag_explorer", "trend_report", "compare_tags", "energy_usage", "batch_summary", "alarm_review"}
	if len(res.Prompts) != len(want) {
		t.Fatalf("got %d prompts, want %d: %v", len(res.Prompts), len(want), res.Prompts)
	}
	seen := map[string]bool{}
	for _, p := range res.Prompts {
		if p.Title == "" {
			t.Errorf("prompt %q missing Title", p.Name)
		}
		if p.Description == "" {
			t.Errorf("prompt %q missing Description", p.Name)
		}
		seen[p.Name] = true
	}
	for _, name := range want {
		if !seen[name] {
			t.Errorf("missing prompt %q", name)
		}
	}
}
