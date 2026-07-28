package tools

import (
	"encoding/json"
	"fmt"
	"strings"
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

func TestGetOptionalStringArg(t *testing.T) {
	args := map[string]any{"name": "CDE.OEE", "count": float64(42)}
	if got := getOptionalStringArg(args, "name"); got == nil || *got != "CDE.OEE" {
		t.Errorf("got %v, want %q", got, "CDE.OEE")
	}
	if got := getOptionalStringArg(args, "missing"); got != nil {
		t.Errorf("got %v, want nil", got)
	}
	if got := getOptionalStringArg(args, "count"); got != nil {
		t.Errorf("got %v, want nil (non-string)", got)
	}
	if got := getOptionalStringArg(nil, "name"); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestGetOptionalIntArg(t *testing.T) {
	args := map[string]any{"count": float64(42), "name": "test"}
	if got := getOptionalIntArg(args, "count"); got == nil || *got != 42 {
		t.Errorf("got %v, want 42", got)
	}
	if got := getOptionalIntArg(args, "missing"); got != nil {
		t.Errorf("got %v, want nil", got)
	}
	if got := getOptionalIntArg(args, "name"); got != nil {
		t.Errorf("got %v, want nil (non-numeric)", got)
	}
	if got := getOptionalIntArg(nil, "count"); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestGetOptionalFloatArg(t *testing.T) {
	args := map[string]any{"percent": float64(95.5), "name": "test"}
	if got := getOptionalFloatArg(args, "percent"); got == nil || *got != 95.5 {
		t.Errorf("got %v, want 95.5", got)
	}
	if got := getOptionalFloatArg(args, "missing"); got != nil {
		t.Errorf("got %v, want nil", got)
	}
	if got := getOptionalFloatArg(args, "name"); got != nil {
		t.Errorf("got %v, want nil (non-numeric)", got)
	}
	if got := getOptionalFloatArg(nil, "percent"); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestFloatPtrStr(t *testing.T) {
	tests := []struct {
		name string
		p    *float64
		want string
	}{
		{"nil", nil, ""},
		{"zero", ptrFloat(0), "0"},
		{"value", ptrFloat(3.14), "3.14"},
		{"large", ptrFloat(12345.6789), "12345.6789"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := floatPtrStr(tt.p); got != tt.want {
				t.Errorf("floatPtrStr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIntPtrStr(t *testing.T) {
	tests := []struct {
		name string
		p    *int
		want string
	}{
		{"nil", nil, ""},
		{"zero", ptrInt(0), "0"},
		{"value", ptrInt(42), "42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := intPtrStr(tt.p); got != tt.want {
				t.Errorf("intPtrStr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func ptrFloat(v float64) *float64 { return &v }
func ptrInt(v int) *int          { return &v }

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

// formatTestRow is a simple JSON-serializable row for testing formatResult.
type formatTestRow struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestFormatResultUnderLimit(t *testing.T) {
	rows := []formatTestRow{{"a", 1}, {"b", 2}}
	// each row ~25 bytes, 2 rows ~50 bytes, well under 10000
	result := formatResult(rows, 10000, func(rows []formatTestRow) string {
		var b strings.Builder
		for _, r := range rows {
			b.WriteString(r.Name)
		}
		return b.String()
	}, "historians://base/tags/id", "tags", nil)

	if len(result.Content) != 1 {
		t.Fatalf("under limit: want 1 content, got %d", len(result.Content))
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("under limit: content[0] is not TextContent")
	}
	var got []formatTestRow
	if err := json.Unmarshal([]byte(tc.Text), &got); err != nil {
		t.Fatalf("under limit: unmarshal: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("under limit: want 2 rows, got %d", len(got))
	}
	if got[0].Name != "a" || got[1].Value != 2 {
		t.Fatal("under limit: row data mismatch")
	}
}

func TestFormatResultOverLimit(t *testing.T) {
	rows := make([]formatTestRow, 10)
	for i := range rows {
		rows[i] = formatTestRow{fmt.Sprintf("k%d", i), i}
	}
	// 10 rows ~300 bytes, limit=50 forces preview
	result := formatResult(rows, 50, func(rows []formatTestRow) string {
		return fmt.Sprintf("%d rows", len(rows))
	}, "historians://base/summary/id", "summary", nil)

	if len(result.Content) != 2 {
		t.Fatalf("over limit: want 2 content, got %d", len(result.Content))
	}
	// Check TextContent is DualToolResult JSON
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("over limit: content[0] not TextContent")
	}
	var dual map[string]any
	if err := json.Unmarshal([]byte(tc.Text), &dual); err != nil {
		t.Fatalf("over limit: unmarshal dual: %v", err)
	}
	if preview := dual["preview"]; preview != "10 rows" {
		t.Fatalf("over limit: preview=%q, want %q", preview, "10 rows")
	}
	if rc := int(dual["rowCount"].(float64)); rc != 10 {
		t.Fatalf("over limit: rowCount=%d, want 10", rc)
	}
	if uri := dual["resourceUri"]; uri != "historians://base/summary/id" {
		t.Fatalf("over limit: resourceUri=%q", uri)
	}
	// Check ResourceLink
	rl, ok := result.Content[1].(*mcp.ResourceLink)
	if !ok {
		t.Fatal("over limit: content[1] not ResourceLink")
	}
	if rl.URI != "historians://base/summary/id" {
		t.Fatalf("over limit: link URI=%q", rl.URI)
	}
}

func TestFormatResultEmpty(t *testing.T) {
	result := formatResult([]formatTestRow{}, 1000, func(rows []formatTestRow) string {
		return "preview"
	}, "uri://test", "t", nil)

	if len(result.Content) != 1 {
		t.Fatalf("empty: want 1 content, got %d", len(result.Content))
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("empty: content[0] not TextContent")
	}
	if tc.Text != "[]" {
		t.Fatalf("empty: want '[]', got %q", tc.Text)
	}
}

func TestFormatResultZeroLimit(t *testing.T) {
	// limit <= 0 always triggers preview
	result := formatResult([]formatTestRow{{"x", 1}}, 0, func(rows []formatTestRow) string {
		return "preview-data"
	}, "uri://test", "t", nil)

	if len(result.Content) != 2 {
		t.Fatalf("zero limit: want 2 content, got %d", len(result.Content))
	}
}
