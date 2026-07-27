package tools

import (
	"encoding/json"
	"fmt"

	"github.com/chewcw/aveva-historian-mcpserver/internal/types"

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

// parseFilters converts a raw JSON value (from tool args) into structured filter groups.
func parseFilters(raw any) []types.FilterGroupDef {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	var groups []types.FilterGroupDef
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		var g types.FilterGroupDef
		if andArr, ok := m["and"].([]any); ok {
			for _, cond := range andArr {
				g.And = append(g.And, parseCondition(cond))
			}
		}
		if orArr, ok := m["or"].([]any); ok {
			for _, cond := range orArr {
				g.Or = append(g.Or, parseCondition(cond))
			}
		}
		groups = append(groups, g)
	}
	return groups
}

func parseCondition(raw any) types.FilterCondition {
	m, ok := raw.(map[string]any)
	if !ok {
		return types.FilterCondition{}
	}
	return types.FilterCondition{
		Field:    stringField(m, "field"),
		Operator: stringField(m, "operator"),
		Value:    m["value"],
	}
}

func stringField(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getOptionalStringArg returns a *string pointer, or nil if arg is missing or not a string.
func getOptionalStringArg(args map[string]any, name string) *string {
	if args == nil {
		return nil
	}
	v, ok := args[name]
	if !ok {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return nil
	}
	return &s
}

// getOptionalIntArg returns a *int pointer, or nil if arg is missing or not numeric.
func getOptionalIntArg(args map[string]any, name string) *int {
	if args == nil {
		return nil
	}
	v, ok := args[name]
	if !ok {
		return nil
	}
	// json.Unmarshal into map[string]any decodes numbers as float64
	f, ok := v.(float64)
	if !ok {
		return nil
	}
	i := int(f)
	return &i
}

// getOptionalFloatArg returns a *float64 pointer, or nil if arg is missing or not numeric.
func getOptionalFloatArg(args map[string]any, name string) *float64 {
	if args == nil {
		return nil
	}
	v, ok := args[name]
	if !ok {
		return nil
	}
	f, ok := v.(float64)
	if !ok {
		return nil
	}
	return &f
}

// floatPtrStr returns the formatted float64 value, or "" if p is nil.
func floatPtrStr(p *float64) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%g", *p)
}

// intPtrStr returns the formatted int value, or "" if p is nil.
func intPtrStr(p *int) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%d", *p)
}
