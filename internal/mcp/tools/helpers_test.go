package tools

import (
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestGetStringArg(t *testing.T) {
	args := map[string]any{"name": "CDE.OEE", "count": float64(42)}
	if got := getStringArg(args, "name"); got != "CDE.OEE" {
		t.Errorf("got %q, want %q", got, "CDE.OEE")
	}
	if got := getStringArg(args, "missing"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	if got := getStringArg(nil, "name"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestGetIntArg(t *testing.T) {
	args := map[string]any{"count": float64(42), "name": "test"}
	if got := getIntArg(args, "count", 10); got != 42 {
		t.Errorf("got %d, want 42", got)
	}
	if got := getIntArg(args, "missing", 99); got != 99 {
		t.Errorf("got %d, want 99", got)
	}
	if got := getIntArg(args, "name", 5); got != 5 {
		t.Errorf("got %d, want 5 (non-numeric)", got)
	}
	if got := getIntArg(nil, "count", 7); got != 7 {
		t.Errorf("got %d, want 7", got)
	}
}

func TestCapRows(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	if got := capRows(items, 3); len(got) != 3 {
		t.Errorf("got %d, want 3", len(got))
	}
	if got := capRows(items, 10); len(got) != 5 {
		t.Errorf("got %d, want 5 (no cap needed)", len(got))
	}
	if got := capRows([]int{}, 5); len(got) != 0 {
		t.Errorf("got %d, want 0 (empty)", len(got))
	}
}

func TestParseArgs(t *testing.T) {
	// nil arguments
	req := &mcp.CallToolRequest{}
	if got := parseArgs(req); got != nil {
		t.Errorf("expected nil for nil arguments")
	}

	// empty raw message
	req.Params = &mcp.CallToolParamsRaw{}
	req.Params.Arguments = json.RawMessage(`{}`)
	if got := parseArgs(req); got != nil {
		t.Errorf("expected nil for empty object")
	}

	// valid arguments
	req.Params.Arguments = json.RawMessage(`{"key": "val", "num": 42}`)
	args := parseArgs(req)
	if args == nil {
		t.Fatal("expected non-nil args")
	}
	if args["key"] != "val" {
		t.Errorf("got key=%v", args["key"])
	}

}
