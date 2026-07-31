package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chewcw/aveva-historian-mcpserver/internal/dataserver"
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

// getOptionalBoolArg returns a *bool pointer, or nil if arg is missing or not a bool.
func getOptionalBoolArg(args map[string]any, name string) *bool {
	if args == nil {
		return nil
	}
	v, ok := args[name]
	if !ok {
		return nil
	}
	b, ok := v.(bool)
	if !ok {
		return nil
	}
	return &b
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

// strPtrStr returns the string value, or "" if p is nil.
func strPtrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// boolPtrStr formats a *bool as "true"/"false"/"" for nil.
func boolPtrStr(b *bool) string {
	if b == nil {
		return ""
	}
	if *b {
		return "true"
	}
	return "false"
}

// getStringSliceArg returns []string from a JSON array arg, or nil if missing or not an array.
func getStringSliceArg(args map[string]any, name string) []string {
	if args == nil {
		return nil
	}
	v, ok := args[name]
	if !ok {
		return nil
	}
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	strs := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			strs = append(strs, s)
		}
	}
	return strs
}

// parseOrderByClauses parses an array of {"field": ..., "direction": ...} objects.
// Returns nil (zero value) if missing or not a valid array.
// formatResult returns full JSON if rows fit within limit, otherwise a preview.
func formatResult[T any](
	rows []T,
	limit int,
	buildPreview func([]T) string,
	resourceURI string,
	linkName string,
	count *int,
) *mcp.CallToolResult {
	fullJSON, err := json.Marshal(rows)
	if err != nil || len(fullJSON) > limit {
		capped := capRows(rows, 100)
		preview := buildPreview(capped)
		dual := types.DualToolResult{
			Preview:     preview,
			RowCount:    len(capped),
			ResourceURI: resourceURI,
		}
		if count != nil {
			dual.TotalCount = count
		}
		dualJSON, _ := json.Marshal(dual)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(dualJSON)},
				&mcp.ResourceLink{URI: resourceURI, Name: linkName},
			},
		}
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(fullJSON)},
		},
	}
}

// PushResult stores a full dataset in the data server and returns its ResourceURI.
// If store is nil (data server unavailable), returns empty URI without error.
func PushResult(store *dataserver.Store, baseURL, label string, columns []dataserver.Field, rows [][]string) (string, error) {
	if store == nil {
		return "", nil // data server not available, skip
	}
	res, err := store.Put(label, "application/json", columns, rows)
	if err != nil {
		return "", fmt.Errorf("push to data server: %w", err)
	}
	return fmt.Sprintf("%s/resources/%s", strings.TrimRight(baseURL, "/"), res.ID), nil
}

func parseOrderByClauses(args map[string]any, name string) []types.OrderByClause {
	if args == nil {
		return nil
	}
	v, ok := args[name]
	if !ok {
		return nil
	}
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	result := make([]types.OrderByClause, 0, len(raw))
	for _, item := range raw {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		field, _ := obj["field"].(string)
		if field == "" {
			continue
		}
		dirStr, _ := obj["direction"].(string)
		var d types.OrderDirection
		switch strings.ToLower(dirStr) {
		case "desc":
			d = types.OrderDesc
		default:
			d = types.OrderAsc
		}
		result = append(result, types.OrderByClause{Field: field, Direction: d})
	}
	return result
}
