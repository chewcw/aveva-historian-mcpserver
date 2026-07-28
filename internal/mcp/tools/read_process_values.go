package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/historian/endpoints"
	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterReadProcessValues(server *mcp.Server, client *historian.Client, logger *slog.Logger) {
	logger = logger.With("tool", "read_process_values")
	inputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"fqn":          map[string]any{"type": "string", "description": "Tag FQN"},
			"start_date_time": map[string]any{"type": "string", "description": "Start date time"},
			"end_date_time":   map[string]any{"type": "string", "description": "End date time"},
			"retrieval_mode": map[string]any{
				"type":        "string",
				"enum":        []string{"Average", "Cyclic", "Integral", "Minimum", "Maximum", "BestFit", "Delta", "Interpolated", "Slope", "Counter", "Full"},
				"description": "Retrieval mode (optional)",
			},
			"resolution_ms": map[string]any{"type": "number", "description": "Granularity in ms (optional)"},
			"opc_quality":   map[string]any{"type": "integer", "description": "OPC quality filter (optional)"},
			"value":         map[string]any{"type": "number", "description": "Value filter: 0 or 1 (optional)"},
			"bounding":      map[string]any{"type": "boolean", "description": "Include boundary data outside query range (optional)"},
			"text":          map[string]any{"type": "string", "description": "Text value for string/discrete tags (optional)"},
			"tag_filter":    map[string]any{"type": "string", "description": "OData filter on tag attributes like FQN, description (optional)"},
			"expression":    map[string]any{"type": "string", "description": "UOM conversion expression, e.g. UOM([FQN],[Unit]) (optional)"},
			"top":           map[string]any{"type": "number", "description": "OData $top — max results (optional, default 100)"},
			"select":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "OData $select — field names, e.g. [\"FQN\",\"DateTime\"] (optional)"},
			"skip":          map[string]any{"type": "number", "description": "OData $skip — number of records to skip (optional)"},
			"orderby":       map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"field": map[string]any{"type": "string"}, "direction": map[string]any{"type": "string", "enum": []string{"asc", "desc"}}}}, "description": "OData $orderby — array of {field, direction}, e.g. [{\"field\":\"DateTime\",\"direction\":\"desc\"}] (optional)"},
			"count":         map[string]any{"type": "boolean", "description": "OData $count — include total count (optional)"},
			"search":        map[string]any{"type": "string", "description": "OData $search — search expression (optional)"},
			"filters": map[string]any{
				"type":        "array",
				"description": "Optional. Additional Filters using OData expressions. Array of groups (AND-combined across groups). Each group has \"and\" or \"or\" with conditions. Condition: {\"field\":\"...\", \"operator\":\"...\", \"value\":...}. Operators: eq,ne,gt,ge,lt,le (str|num), startsWith,endsWith,contains (str), in (array), has (str). Example: [{\"and\":[{\"field\":\"FQN\",\"operator\":\"startsWith\",\"value\":\"CDE\"}]}] -> startswith(FQN,CDE)",
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
		Name: "read_process_values",
		Description: "Retrieve time-series process values for a historian tag over a time range. " +
			"Optional: retrieval_mode (Average|Cyclic|Integral|Minimum|Maximum|BestFit|Delta|Interpolated|Slope|Counter|Full), " +
			"resolution_ms. " +
			"Note: do not URL-encode param values — client handles encoding automatically.",
		InputSchema: inputSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		var convConds []types.FilterCondition

		// Required: FQN eq
		fqn := getStringArg(args, "fqn")
		if fqn == "" {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "FQN is required"}},
			}, nil
		}
		convConds = append(convConds, types.FilterCondition{Field: "FQN", Operator: "eq", Value: fqn})

		// Required: start_date_time → DateTime ge
		startDateTime := getStringArg(args, "start_date_time")
		if startDateTime == "" {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "start_date_time is required"}},
			}, nil
		}
		convConds = append(convConds, types.FilterCondition{Field: "DateTime", Operator: "ge", Value: startDateTime})

		// Required: end_date_time → DateTime le
		endDateTime := getStringArg(args, "end_date_time")
		if endDateTime == "" {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "end_date_time is required"}},
			}, nil
		}
		convConds = append(convConds, types.FilterCondition{Field: "DateTime", Operator: "le", Value: endDateTime})

		// Optional: retrieval_mode
		if mode := getOptionalStringArg(args, "retrieval_mode"); mode != nil {
			if !isValidRetrievalMode(*mode) {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("unsupported retrieval mode: %s", *mode)}},
				}, nil
			}
			convConds = append(convConds, types.FilterCondition{Field: "RetrievalMode", Operator: "eq", Value: *mode})
		}

		// Optional: resolution_ms
		if ms := getOptionalIntArg(args, "resolution_ms"); ms != nil {
			convConds = append(convConds, types.FilterCondition{Field: "Resolution", Operator: "eq", Value: *ms})
		}

		// Optional: opc_quality
		if q := getOptionalIntArg(args, "opc_quality"); q != nil {
			convConds = append(convConds, types.FilterCondition{Field: "OPCQuality", Operator: "eq", Value: *q})
		}

		// Optional: value
		if v := getOptionalFloatArg(args, "value"); v != nil {
			convConds = append(convConds, types.FilterCondition{Field: "Value", Operator: "eq", Value: *v})
		}

		// Optional: bounding
		if b := getOptionalBoolArg(args, "bounding"); b != nil {
			convConds = append(convConds, types.FilterCondition{Field: "Bounding", Operator: "eq", Value: *b})
		}

		// Optional: text
		if txt := getOptionalStringArg(args, "text"); txt != nil {
			convConds = append(convConds, types.FilterCondition{Field: "Text", Operator: "eq", Value: *txt})
		}

		// Optional: tag_filter
		if tf := getOptionalStringArg(args, "tag_filter"); tf != nil {
			convConds = append(convConds, types.FilterCondition{Field: "TagFilter", Operator: "eq", Value: *tf})
		}

		// Optional: expression
		if expr := getOptionalStringArg(args, "expression"); expr != nil {
			convConds = append(convConds, types.FilterCondition{Field: "Expression", Operator: "eq", Value: *expr})
		}

		var groups []types.FilterGroupDef
		if len(convConds) > 0 {
			groups = append(groups, types.FilterGroupDef{And: convConds})
		}
		userFilters := parseFilters(args["filters"])
		groups = append(groups, userFilters...)

		// OData system query options
		topVal := getIntArg(args, "top", 100)
		opts := types.QueryOptions{
			Top:     &topVal,
			Select:  getStringSliceArg(args, "select"),
			Skip:    getOptionalIntArg(args, "skip"),
			OrderBy: parseOrderByClauses(args, "orderby"),
			Count:   getOptionalBoolArg(args, "count"),
			Search:  getOptionalStringArg(args, "search"),
		}

		result, err := endpoints.GetProcessValues(ctx, client, groups, opts)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to read trends: %v", err)}},
			}, nil
		}

		rows := result.Value
		if rows == nil {
			rows = []historian.ProcessValue{}
		}
		capped := capRows(rows, 100)

		var preview strings.Builder
		preview.WriteString("| FQN | DateTime | Value | OpcQuality | Unit |\n")
		preview.WriteString("|-----|----------|-------|------------|------|\n")
		for _, pv := range capped {
			fmt.Fprintf(&preview, "| %s | %s | %s | %s | %s |\n",
				pv.FQN, pv.DateTime, floatPtrStr(pv.Value),
				intPtrStr(pv.OpcQuality), pv.Unit)
		}

		resourceURI := fmt.Sprintf("historians://%s/trends/%s", client.BaseURL(), uuid.New().String())

		dualResult := types.DualToolResult{
			Preview:     preview.String(),
			RowCount:    len(capped),
			ResourceURI: resourceURI,
		}
		if result.Count != nil {
			dualResult.TotalCount = result.Count
		}

		dualJSON, _ := json.Marshal(dualResult)

		logger.Info("ok", "rows", len(rows), "fqn", fqn)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(dualJSON)},
				&mcp.ResourceLink{URI: resourceURI, Name: "trends"},
			},
		}, nil
	})
}

func isValidRetrievalMode(mode string) bool {
	switch mode {
	case "Average", "Cyclic", "Integral", "Minimum", "Maximum",
		"BestFit", "Delta", "Interpolated", "Slope", "Counter", "Full":
		return true
	default:
		return false
	}
}
