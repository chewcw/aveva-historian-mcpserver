package prompts

import (
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

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

func TestResolveAlarmWindow(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	t.Run("both defaults", func(t *testing.T) {
		start, end, note := resolveAlarmWindow(nil, now)
		if start != "2026-08-14T12:00:00Z" {
			t.Errorf("start = %q, want 2026-08-14T12:00:00Z", start)
		}
		if end != "2026-08-15T12:00:00Z" {
			t.Errorf("end = %q, want 2026-08-15T12:00:00Z", end)
		}
		if note != "(last 24h, now, default)" {
			t.Errorf("note = %q, want (last 24h, now, default)", note)
		}
	})
	t.Run("explicit values no note", func(t *testing.T) {
		args := map[string]string{
			"start_date_time": "2026-08-01T00:00:00Z",
			"end_date_time":   "2026-08-02T00:00:00Z",
		}
		start, end, note := resolveAlarmWindow(args, now)
		if start != "2026-08-01T00:00:00Z" || end != "2026-08-02T00:00:00Z" || note != "" {
			t.Errorf("got (%q, %q, %q), want explicit values and empty note", start, end, note)
		}
	})
}

func TestResolveDayWindow(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	t.Run("explicit date", func(t *testing.T) {
		start, end, date, note, err := resolveDayWindow(map[string]string{"date": "2026-08-01"}, now)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if start != "2026-08-01T00:00:00Z" || end != "2026-08-01T23:59:59Z" || date != "2026-08-01" || note != "" {
			t.Errorf("got (%q, %q, %q, %q), want day boundaries with empty note", start, end, date, note)
		}
	})
	t.Run("today default", func(t *testing.T) {
		start, end, date, note, err := resolveDayWindow(nil, now)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if date != "2026-08-15" || note != "(today, default)" {
			t.Errorf("date = %q note = %q, want today + (today, default)", date, note)
		}
		if start != "2026-08-15T00:00:00Z" || end != "2026-08-15T23:59:59Z" {
			t.Errorf("bounds = (%q, %q)", start, end)
		}
	})
	t.Run("bad date", func(t *testing.T) {
		_, _, _, _, err := resolveDayWindow(map[string]string{"date": "15/08/2026"}, now)
		if err == nil || !strings.Contains(err.Error(), "date must be YYYY-MM-DD") {
			t.Errorf("want date format error, got %v", err)
		}
	})
}

func TestBoolArg(t *testing.T) {
	args := map[string]string{"ack": "true", "no": "false", "junk": "yes"}
	if got := boolArg(args, "ack", nil); got == nil || !*got {
		t.Errorf("boolArg(ack) = %v, want true", got)
	}
	if got := boolArg(args, "no", nil); got == nil || *got {
		t.Errorf("boolArg(no) = %v, want false", got)
	}
	if got := boolArg(args, "junk", nil); got != nil {
		t.Errorf("boolArg(junk) = %v, want nil", got)
	}
	if got := boolArg(args, "missing", nil); got != nil {
		t.Errorf("boolArg(missing) = %v, want nil", got)
	}
}

func TestSeverityArg(t *testing.T) {
	if n, err := severityArg(nil); n != 0 || err != nil {
		t.Errorf("absent: (%d, %v), want (0, nil)", n, err)
	}
	if n, err := severityArg(map[string]string{"severity": "1"}); n != 1 || err != nil {
		t.Errorf("severity 1: (%d, %v)", n, err)
	}
	if _, err := severityArg(map[string]string{"severity": "5"}); err == nil {
		t.Error("severity 5: want error")
	}
	if _, err := severityArg(map[string]string{"severity": "abc"}); err == nil {
		t.Error("severity abc: want error")
	}
}

func TestBoolStr(t *testing.T) {
	if got := boolStr(nil); got != "" {
		t.Errorf("boolStr(nil) = %q, want ''", got)
	}
	v := true
	if got := boolStr(&v); got != "true" {
		t.Errorf("boolStr(true) = %q, want 'true'", got)
	}
}
