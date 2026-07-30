package tools

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/chewcw/aveva-historian-mcpserver/internal/dataserver"
	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/historian/endpoints"
	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterGetTags(server *mcp.Server, client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) {
	logger = logger.With("tool", "get_tags")
	inputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"top": map[string]any{"type": "number", "description": "Max rows (default 50)"},
			"select": map[string]any{
				"type":        "string",
				"description": "Comma-separated fields to include in results (optional)",
			},
			"skip": map[string]any{"type": "number", "description": "Rows to skip (optional)"},
			"orderby": map[string]any{
				"type":        "array",
				"description": "Sort criteria. Array of {field: string, direction: \"asc\"|\"desc\"} (optional)",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"field":     map[string]any{"type": "string"},
						"direction": map[string]any{"type": "string", "enum": []string{"asc", "desc"}},
					},
				},
			},
			"count":              map[string]any{"type": "boolean", "description": "Include total count (optional)"},
			"search":             map[string]any{"type": "string", "description": "Free-text search (optional)"},
			"source":             map[string]any{"type": "string", "description": "Filter by data source (optional)"},
			"eng_unit":           map[string]any{"type": "string", "description": "Filter by engineering unit (optional)"},
			"eng_unit_max":       map[string]any{"type": "number", "description": "Filter by max engineering unit value (optional)"},
			"eng_unit_min":       map[string]any{"type": "number", "description": "Filter by min engineering unit value (optional)"},
			"interpolation_type": map[string]any{"type": "string", "description": "Filter by interpolation type (optional)"},
			"integral_divisor":   map[string]any{"type": "number", "description": "Filter by integral divisor (optional)"},
			"rollover_value":     map[string]any{"type": "number", "description": "Filter by rollover value (optional)"},
			"message_off":        map[string]any{"type": "string", "description": "Filter by discrete FALSE-state message (optional)"},
			"message_on":         map[string]any{"type": "string", "description": "Filter by discrete TRUE-state message (optional)"},
			"tag_type":           map[string]any{"type": "string", "description": "Filter by tag type (optional): Analog(1), Discrete(2), String(3), Event(5), Summary(7)"},
			"tag_name":           map[string]any{"type": "string", "description": "Filter by tag name (optional)"},
			"description":        map[string]any{"type": "string", "description": "Filter by tag description (optional)"},
			"filters": map[string]any{
				"type":        "array",
				"description": "Filters using OData expressions. Array of groups (AND-combined across groups). Each group has \"and\" or \"or\" with conditions. Condition: {\"field\":\"...\", \"operator\":\"...\", \"value\":...}. Operators: eq,ne,gt,ge,lt,le (str|num), startsWith,endsWith,contains (str), in (array), has (str). Example: [{\"and\":[{\"field\":\"FQN\",\"operator\":\"startsWith\",\"value\":\"CDE\"}]}] -> startswith(FQN,CDE)",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"and": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"field":    map[string]any{"type": "string"},
									"operator": map[string]any{"type": "string", "enum": []string{"eq", "ne", "gt", "ge", "lt", "le", "startsWith", "endsWith", "contains", "in", "has"}},
									"value": map[string]any{
										"oneOf": []any{
											map[string]any{"type": "string"},
											map[string]any{"type": "number"},
											map[string]any{"type": "boolean"},
											map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
										},
									},
								},
							},
						},
						"or": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"field":    map[string]any{"type": "string"},
									"operator": map[string]any{"type": "string", "enum": []string{"eq", "ne", "gt", "ge", "lt", "le", "startsWith", "endsWith", "contains", "in", "has"}},
									"value": map[string]any{
										"oneOf": []any{
											map[string]any{"type": "string"},
											map[string]any{"type": "number"},
											map[string]any{"type": "boolean"},
											map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	server.AddTool(&mcp.Tool{
		Name: "get_tags",
		Description: "Retrieve a list of historian tags matching an optional filter. " +
			"Supports tag property filters: eng_unit, eng_unit_max, eng_unit_min, " +
			"interpolation_type, integral_divisor, rollover_value, message_off, " +
			"message_on, tag_type, tag_name, description. " +
			"Note: do not URL-encode param values — client handles encoding automatically.",
		InputSchema: inputSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		var convConds []types.FilterCondition

		// Optional convenience params → FilterConditions
		if v := getOptionalStringArg(args, "source"); v != nil {
			convConds = append(convConds, types.FilterCondition{Field: "Source", Operator: "eq", Value: *v})
		}
		if v := getOptionalStringArg(args, "eng_unit"); v != nil {
			convConds = append(convConds, types.FilterCondition{Field: "EngUnit", Operator: "eq", Value: *v})
		}
		if v := getOptionalFloatArg(args, "eng_unit_max"); v != nil {
			convConds = append(convConds, types.FilterCondition{Field: "EngUnitMax", Operator: "eq", Value: *v})
		}
		if v := getOptionalFloatArg(args, "eng_unit_min"); v != nil {
			convConds = append(convConds, types.FilterCondition{Field: "EngUnitMin", Operator: "eq", Value: *v})
		}
		if v := getOptionalStringArg(args, "interpolation_type"); v != nil {
			convConds = append(convConds, types.FilterCondition{Field: "InterpolationType", Operator: "eq", Value: *v})
		}
		if v := getOptionalFloatArg(args, "integral_divisor"); v != nil {
			convConds = append(convConds, types.FilterCondition{Field: "IntegralDivisor", Operator: "eq", Value: *v})
		}
		if v := getOptionalFloatArg(args, "rollover_value"); v != nil {
			convConds = append(convConds, types.FilterCondition{Field: "RolloverValue", Operator: "eq", Value: *v})
		}
		if v := getOptionalStringArg(args, "message_off"); v != nil {
			convConds = append(convConds, types.FilterCondition{Field: "MessageOff", Operator: "eq", Value: *v})
		}
		if v := getOptionalStringArg(args, "message_on"); v != nil {
			convConds = append(convConds, types.FilterCondition{Field: "MessageOn", Operator: "eq", Value: *v})
		}
		if v := getOptionalStringArg(args, "tag_type"); v != nil {
			convConds = append(convConds, types.FilterCondition{Field: "TagType", Operator: "eq", Value: *v})
		}
		if v := getOptionalStringArg(args, "tag_name"); v != nil {
			convConds = append(convConds, types.FilterCondition{Field: "TagName", Operator: "eq", Value: *v})
		}
		if v := getOptionalStringArg(args, "description"); v != nil {
			convConds = append(convConds, types.FilterCondition{Field: "Description", Operator: "eq", Value: *v})
		}

		// Build filter groups from convConds + user filters
		var groups []types.FilterGroupDef
		if len(convConds) > 0 {
			groups = append(groups, types.FilterGroupDef{And: convConds})
		}
		userFilters := parseFilters(args["filters"])
		groups = append(groups, userFilters...)

		// OData system query options — top defaults to 50
		topVal := getIntArg(args, "top", 50)
		opts := types.QueryOptions{
			Top:     &topVal,
			Select:  getStringSliceArg(args, "select"),
			Skip:    getOptionalIntArg(args, "skip"),
			OrderBy: parseOrderByClauses(args, "orderby"),
			Count:   getOptionalBoolArg(args, "count"),
			Search:  getOptionalStringArg(args, "search"),
		}

		result, err := endpoints.GetTags(ctx, client, groups, opts)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to query tags: %v", err)}},
			}, nil
		}

		rows := result.Value
		if rows == nil {
			rows = []historian.Tag{}
		}

		// Build full dataset for data server
		allRows := make([][]string, 0, len(rows))
		for _, t := range rows {
				allRows = append(allRows, []string{
				t.FQN, t.TagName, t.TagType, t.EngUnit, t.Source,
				floatPtrStr(t.EngUnitMax), floatPtrStr(t.EngUnitMin),
				t.InterpolationType, t.MessageOff, t.MessageOn,
				t.Alias, t.Description,
			})
		}

		resourceURI, err := PushResult(store, dataServerBaseURL,
			fmt.Sprintf("Tags: %s", "query"),
			[]dataserver.Field{
				{Name: "FQN", Type: "string"},
				{Name: "TagName", Type: "string"},
				{Name: "TagType", Type: "string"},
				{Name: "EngUnit", Type: "string"},
				{Name: "Source", Type: "string"},
				{Name: "EngUnitMax", Type: "number"},
				{Name: "EngUnitMin", Type: "number"},
				{Name: "InterpolationType", Type: "string"},
				{Name: "MessageOff", Type: "string"},
				{Name: "MessageOn", Type: "string"},
				{Name: "Alias", Type: "string"},
				{Name: "Description", Type: "string"},
			},
			allRows)
		if err != nil {
			logger.Warn("failed to push result to data server", "error", err)
			resourceURI = ""
		}

		logger.Info("ok", "rows", len(rows), "top", topVal)

		return formatResult(rows, byteLimit, func(rows []historian.Tag) string {
			var preview strings.Builder
			preview.WriteString("| FQN | TagName | TagType | EngUnit | Source | EngUnitMax | EngUnitMin | InterpolationType | MessageOff | MessageOn | Alias | Description |\n")
			preview.WriteString("|-----|---------|---------|--------|--------|------------|------------|-------------------|------------|-----------|-------|-------------|\n")
			for _, t := range rows {
				fmt.Fprintf(&preview, "| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
					t.FQN, t.TagName, t.TagType, t.EngUnit, t.Source,
					floatPtrStr(t.EngUnitMax), floatPtrStr(t.EngUnitMin),
					t.InterpolationType, t.MessageOff, t.MessageOn,
					t.Alias, t.Description)
			}
			return preview.String()
		}, resourceURI, "tags", result.Count), nil
	})
}
