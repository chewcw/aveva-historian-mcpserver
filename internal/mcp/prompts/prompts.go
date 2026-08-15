package prompts

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chewcw/aveva-historian-mcpserver/internal/dataserver"
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
