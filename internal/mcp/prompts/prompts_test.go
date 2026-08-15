package prompts

import (
	"io"
	"log/slog"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// discardLogger returns a logger that writes nowhere, for handler tests.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestPromptArgs(t *testing.T) {
	req := &mcp.GetPromptRequest{Params: &mcp.GetPromptParams{Arguments: map[string]string{"fqn": "CDE.OEE"}}}
	if got := promptArgs(req); got["fqn"] != "CDE.OEE" {
		t.Errorf("promptArgs = %v", got)
	}
	if got := promptArgs(&mcp.GetPromptRequest{}); got != nil {
		t.Errorf("promptArgs(nil params) = %v, want nil", got)
	}
	if got := promptArgs(nil); got != nil {
		t.Errorf("promptArgs(nil) = %v, want nil", got)
	}
}

func TestRequiredArg(t *testing.T) {
	args := map[string]string{"fqn": " CDE.OEE ", "empty": ""}
	v, err := requiredArg(args, "fqn")
	if err != nil || v != "CDE.OEE" {
		t.Errorf("requiredArg(fqn) = %q, %v; want trimmed value", v, err)
	}
	if _, err := requiredArg(args, "missing"); err == nil {
		t.Error("requiredArg(missing) = nil error, want error")
	}
	if _, err := requiredArg(args, "empty"); err == nil {
		t.Error("requiredArg(empty) = nil error, want error")
	}
	if _, err := requiredArg(nil, "fqn"); err == nil {
		t.Error("requiredArg(nil args) = nil error, want error")
	}
}

func TestErrResult(t *testing.T) {
	res := errResult("Missing required argument: fqn")
	if len(res.Messages) != 1 {
		t.Fatalf("want 1 message, got %d", len(res.Messages))
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || tc.Text != "Missing required argument: fqn" {
		t.Errorf("unexpected content: %#v", res.Messages[0].Content)
	}
}

func TestResultMessage(t *testing.T) {
	res := resultMessage("preview", resourceRef{URI: "http://x/resources/1", Name: "tags"}, resourceRef{})
	if len(res.Messages) != 2 {
		t.Fatalf("want 2 messages, got %d", len(res.Messages))
	}
	tc, ok := res.Messages[0].Content.(*mcp.TextContent)
	if !ok || tc.Text != "preview" {
		t.Errorf("msg[0] = %#v", res.Messages[0].Content)
	}
	rl, ok := res.Messages[1].Content.(*mcp.ResourceLink)
	if !ok || rl.URI != "http://x/resources/1" {
		t.Errorf("msg[1] = %#v", res.Messages[1].Content)
	}
	if got := len(resultMessage("p", resourceRef{}).Messages); got != 1 {
		t.Errorf("empty ref not dropped: %d messages", got)
	}
}

func TestSplitCSV(t *testing.T) {
	got := splitCSV("A, B ,B, , C")
	want := []string{"A", "B", "C"}
	if len(got) != len(want) {
		t.Fatalf("splitCSV = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("splitCSV = %v, want %v", got, want)
		}
	}
	if got := splitCSV(""); len(got) != 0 {
		t.Errorf("splitCSV('') = %v, want empty", got)
	}
}

func TestIntArgAndClamp(t *testing.T) {
	args := map[string]string{"top": "120"}
	if got := intArg(args, "top", 50); got != 120 {
		t.Errorf("intArg = %d", got)
	}
	if got := intArg(args, "nope", 50); got != 50 {
		t.Errorf("intArg default = %d", got)
	}
	if got := intArg(map[string]string{"top": "abc"}, "top", 50); got != 50 {
		t.Errorf("intArg invalid = %d", got)
	}
	if got := clampInt(5000, 1, 1000); got != 1000 {
		t.Errorf("clampInt hi = %d", got)
	}
	if got := clampInt(0, 1, 1000); got != 1 {
		t.Errorf("clampInt lo = %d", got)
	}
}

func TestCapRows(t *testing.T) {
	items := []int{1, 2, 3}
	if got := capRows(items, 2); len(got) != 2 {
		t.Errorf("capRows = %d", len(got))
	}
	if got := capRows(items, 10); len(got) != 3 {
		t.Errorf("capRows no-op = %d", len(got))
	}
}

func TestFmtHelpers(t *testing.T) {
	f := 3.14
	i := 42
	s := "x"
	if got := fmtFloat(&f); got != "3.14" {
		t.Errorf("fmtFloat = %q", got)
	}
	if got := fmtFloat(nil); got != "" {
		t.Errorf("fmtFloat(nil) = %q", got)
	}
	if got := fmtInt(&i); got != "42" {
		t.Errorf("fmtInt = %q", got)
	}
	if got := fmtInt(nil); got != "" {
		t.Errorf("fmtInt(nil) = %q", got)
	}
	if got := fmtStr(&s); got != "x" {
		t.Errorf("fmtStr = %q", got)
	}
	if got := fmtStr(nil); got != "" {
		t.Errorf("fmtStr(nil) = %q", got)
	}
}

func TestIsValidRetrievalMode(t *testing.T) {
	if !isValidRetrievalMode("Average") || !isValidRetrievalMode("Full") {
		t.Error("valid modes rejected")
	}
	if isValidRetrievalMode("Bogus") {
		t.Error("invalid mode accepted")
	}
}
