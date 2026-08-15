package prompts

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/chewcw/aveva-historian-mcpserver/internal/dataserver"
	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/mcp/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// promptArgs returns the prompt arguments map, or nil when absent.
func promptArgs(req *mcp.GetPromptRequest) map[string]string {
	if req == nil || req.Params == nil {
		return nil
	}
	return req.Params.Arguments
}

// requiredArg returns the trimmed value of a required argument, or an error
// naming the missing argument.
func requiredArg(args map[string]string, name string) (string, error) {
	if args == nil || strings.TrimSpace(args[name]) == "" {
		return "", fmt.Errorf("Missing required argument: %s", name)
	}
	return strings.TrimSpace(args[name]), nil
}

// errResult returns a prompt result whose only message is a user-visible error.
// GetPromptResult has no IsError flag, so errors are communicated as text.
func errResult(msg string) *mcp.GetPromptResult {
	return &mcp.GetPromptResult{
		Messages: []*mcp.PromptMessage{
			{Role: "user", Content: &mcp.TextContent{Text: msg}},
		},
	}
}

// resourceRef describes a pushed dataset for the resource-link message.
type resourceRef struct {
	URI  string
	Name string
}

// resultMessage builds a prompt result: one markdown message plus one
// resource-link message per non-empty ref (trend_report passes two).
func resultMessage(markdown string, refs ...resourceRef) *mcp.GetPromptResult {
	msgs := []*mcp.PromptMessage{
		{Role: "user", Content: &mcp.TextContent{Text: markdown}},
	}
	for _, r := range refs {
		if r.URI == "" {
			continue
		}
		msgs = append(msgs, &mcp.PromptMessage{
			Role:    "user",
			Content: &mcp.ResourceLink{URI: r.URI, Name: r.Name},
		})
	}
	return &mcp.GetPromptResult{Messages: msgs}
}

// pushRef pushes rows to the data server and returns a resourceRef for the
// result message. Returns an empty ref when the store is unavailable.
func pushRef(store *dataserver.Store, baseURL, label, name string, columns []dataserver.Field, rows [][]string) (resourceRef, error) {
	uri, err := tools.PushResult(store, baseURL, label, columns, rows)
	if err != nil {
		return resourceRef{}, fmt.Errorf("push to data server: %w", err)
	}
	return resourceRef{URI: uri, Name: name}, nil
}

// splitCSV splits a comma-separated string into trimmed, de-duplicated parts.
func splitCSV(s string) []string {
	seen := map[string]bool{}
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		out = append(out, part)
	}
	return out
}

// intArg returns the int value of an argument, or def when missing or invalid.
func intArg(args map[string]string, name string, def int) int {
	if args == nil {
		return def
	}
	v, ok := args[name]
	if !ok {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return n
}

// clampInt bounds v to [lo, hi].
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// capRows limits a slice to at most max elements.
func capRows[T any](rows []T, max int) []T {
	if len(rows) > max {
		return rows[:max]
	}
	return rows
}

// fmtFloat formats a *float64 as %g, or "" when nil.
func fmtFloat(p *float64) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%g", *p)
}

// fmtInt formats a *int, or "" when nil.
func fmtInt(p *int) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%d", *p)
}

// fmtStr dereferences a *string, or "" when nil.
func fmtStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// RegisterPrompts registers all six historian prompts on the server.
func RegisterPrompts(server *mcp.Server, client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) {
	logger = logger.With("feature", "prompts")
	RegisterCurrentStatus(server, client, store, dataServerBaseURL, logger, byteLimit)
	RegisterTagProfiler(server, client, store, dataServerBaseURL, logger, byteLimit)
	RegisterTagExplorer(server, client, store, dataServerBaseURL, logger, byteLimit)
	RegisterTrendReport(server, client, store, dataServerBaseURL, logger, byteLimit)
	RegisterCompareTags(server, client, store, dataServerBaseURL, logger, byteLimit)
	RegisterEnergyUsage(server, client, store, dataServerBaseURL, logger, byteLimit)
}

// isValidRetrievalMode reports whether mode is a supported process-values mode.
func isValidRetrievalMode(mode string) bool {
	switch mode {
	case "Average", "Cyclic", "Integral", "Minimum", "Maximum",
		"BestFit", "Delta", "Interpolated", "Slope", "Counter", "Full":
		return true
	default:
		return false
	}
}

const (
	// timeLayout is the UTC timestamp layout used for prompt time windows.
	timeLayout = "2006-01-02T15:04:05Z"
	// dayLayout is the YYYY-MM-DD date layout accepted by daily_ops_summary.
	dayLayout = "2006-01-02"
)

// resolveAlarmWindow returns the event window for alarm_review. Missing
// start_date_time defaults to now-24h and missing end_date_time to now; note
// carries "(default)" flags for whichever defaults were applied.
func resolveAlarmWindow(args map[string]string, now time.Time) (start, end, note string) {
	start = strings.TrimSpace(args["start_date_time"])
	end = strings.TrimSpace(args["end_date_time"])
	var flags []string
	if start == "" {
		start = now.Add(-24 * time.Hour).UTC().Format(timeLayout)
		flags = append(flags, "last 24h")
	}
	if end == "" {
		end = now.UTC().Format(timeLayout)
		flags = append(flags, "now")
	}
	if len(flags) > 0 {
		note = "(" + strings.Join(flags, ", ") + ", default)"
	}
	return start, end, note
}

// resolveDayWindow returns the day boundaries for daily_ops_summary. A missing
// date defaults to today (UTC); date must parse as YYYY-MM-DD. note carries
// "(today, default)" when the default was used.
func resolveDayWindow(args map[string]string, now time.Time) (start, end, date, note string, err error) {
	date = strings.TrimSpace(args["date"])
	if date == "" {
		date = now.UTC().Format(dayLayout)
		note = "(today, default)"
	} else if _, err := time.Parse(dayLayout, date); err != nil {
		return "", "", "", "", fmt.Errorf("date must be YYYY-MM-DD")
	}
	start = date + "T00:00:00Z"
	end = date + "T23:59:59Z"
	return start, end, date, note, nil
}

// boolArg returns the *bool value of an argument, or def when the argument is
// missing or not "true"/"false".
func boolArg(args map[string]string, name string, def *bool) *bool {
	if args == nil {
		return def
	}
	v, ok := args[name]
	if !ok {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true":
		b := true
		return &b
	case "false":
		b := false
		return &b
	}
	return def
}

// severityArg parses the severity argument, validating 1-4 (1=Critical,
// 2=Major, 3=Minor, 4=Informational). Returns (0, nil) when absent (no
// filter) and an error only when present but out of range.
func severityArg(args map[string]string) (int, error) {
	raw := strings.TrimSpace(args["severity"])
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > 4 {
		return 0, fmt.Errorf("severity must be 1-4 (1=Critical, 2=Major, 3=Minor, 4=Informational)")
	}
	return n, nil
}

// boolStr formats a *bool as "true"/"false", or "" when nil.
func boolStr(p *bool) string {
	if p == nil {
		return ""
	}
	if *p {
		return "true"
	}
	return "false"
}
