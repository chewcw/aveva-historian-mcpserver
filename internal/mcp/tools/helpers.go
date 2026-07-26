package tools

import (
	"encoding/json"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// parseArgs unmarshals the raw JSON arguments from a CallToolRequest into a map.
// Returns nil if Arguments is nil, empty, or unmarshals to an empty map.
func parseArgs(req *mcp.CallToolRequest) map[string]any {
	if req == nil || req.Params == nil || req.Params.Arguments == nil || len(req.Params.Arguments) == 0 {
		return nil
	}
	var args map[string]any
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil
	}
	if len(args) == 0 {
		return nil
	}
	return args
}

// getStringArg returns the string value of a named argument, or "" if missing or not a string.
func getStringArg(args map[string]any, name string) string {
	if args == nil {
		return ""
	}
	if v, ok := args[name]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getIntArg returns the int value of a named argument, or def if missing or not numeric.
func getIntArg(args map[string]any, name string, def int) int {
	if args == nil {
		return def
	}
	if v, ok := args[name]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return def
}

// capRows limits a slice to at most max elements.
func capRows[T any](rows []T, max int) []T {
	if len(rows) > max {
		return rows[:max]
	}
	return rows
}
