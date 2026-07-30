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

func RegisterReadAnalogSummary(server *mcp.Server, client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) {
	logger = logger.With("tool", "read_analog_summary")
	inputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"fqn":             map[string]any{"type": "string", "description": "Tag FQN (required)"},
			"start_date_time": map[string]any{"type": "string", "description": "Start time (required)"},
			"end_date_time":   map[string]any{"type": "string", "description": "End time (required)"},
			"resolution_ms":   map[string]any{"type": "number", "description": "Resolution in ms (optional)"},
			"retrieval_mode": map[string]any{
				"type":        "string",
				"enum":        []string{"Cyclic", "Full"},
				"description": "Retrieval mode (optional): Cyclic or Full",
			},
			"slice_by": map[string]any{
				"type":        "string",
				"description": "Comma-separated FQNs (max 10) for dynamic cycle computation (optional)",
			},
			"slice_by_value": map[string]any{
				"type":        "string",
				"description": "Filter criterion for SliceBy results (optional)",
			},
			"opc_quality": map[string]any{
				"type":        "integer",
				"description": "OPC quality filter (Int32) (optional)",
			},
			"percent_good": map[string]any{
				"type":        "number",
				"description": "Percent good threshold (0-100) (optional)",
			},
			"top": map[string]any{"type": "number", "description": "Max rows (optional, default 100)"},
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
			"count":  map[string]any{"type": "boolean", "description": "Include total count (optional)"},
			"search": map[string]any{"type": "string", "description": "Free-text search (optional)"},
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
		Name: "read_analog_summary",
		Description: "Retrieve analog summary statistics for historian tags over a time range. " +
			"Returns per-cycle statistics: Min, Max, Avg, StdDev, Integral, Count, " +
			"First, Last, OPCQuality, PercentGood with timestamps. " +
			"Optional: retrieval_mode (Cyclic|Full), slice_by (comma-sep FQNs), " +
			"slice_by_value, opc_quality, percent_good. " +
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

		// Required: start_date_time → StartDateTime ge
		startDateTime := getStringArg(args, "start_date_time")
		if startDateTime == "" {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "start_date_time is required"}},
			}, nil
		}
		convConds = append(convConds, types.FilterCondition{Field: "StartDateTime", Operator: "ge", Value: startDateTime})

		// Required: end_date_time → EndDateTime le
		endDateTime := getStringArg(args, "end_date_time")
		if endDateTime == "" {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "end_date_time is required"}},
			}, nil
		}
		convConds = append(convConds, types.FilterCondition{Field: "EndDateTime", Operator: "le", Value: endDateTime})

		// Optional: retrieval_mode (validate against Cyclic|Full)
		if mode := getOptionalStringArg(args, "retrieval_mode"); mode != nil {
			if *mode != "Cyclic" && *mode != "Full" {
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

		// Optional: slice_by
		if sb := getOptionalStringArg(args, "slice_by"); sb != nil {
			convConds = append(convConds, types.FilterCondition{Field: "SliceBy", Operator: "eq", Value: *sb})
		}

		// Optional: slice_by_value
		if sbv := getOptionalStringArg(args, "slice_by_value"); sbv != nil {
			convConds = append(convConds, types.FilterCondition{Field: "SliceByValue", Operator: "eq", Value: *sbv})
		}

		// Optional: opc_quality
		if q := getOptionalIntArg(args, "opc_quality"); q != nil {
			convConds = append(convConds, types.FilterCondition{Field: "OPCQuality", Operator: "eq", Value: *q})
		}

		// Optional: percent_good
		if pg := getOptionalFloatArg(args, "percent_good"); pg != nil {
			convConds = append(convConds, types.FilterCondition{Field: "PercentGood", Operator: "eq", Value: *pg})
		}

		// Build filter groups from convConds + user filters
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

		result, err := endpoints.GetAnalogSummary(ctx, client, groups, opts)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to read analog summary: %v", err)}},
			}, nil
		}

		rows := result.Value
		if rows == nil {
			rows = []historian.AnalogSummaryValue{}
		}

		// Build full dataset for data server
		allRows := make([][]string, 0, len(rows))
		for _, s := range rows {
			allRows = append(allRows, []string{
				s.FQN, s.StartDateTime, s.EndDateTime, s.RetrievalMode,
				intPtrStr(s.Resolution), s.SliceBy, s.SliceByValue,
				intPtrStr(s.OPCQuality), floatPtrStr(s.PercentGood),
				floatPtrStr(s.First), s.FirstDateTime,
				floatPtrStr(s.Last), s.LastDateTime,
				floatPtrStr(s.Minimum), s.MinDateTime,
				floatPtrStr(s.Maximum), s.MaxDateTime,
				floatPtrStr(s.Average), floatPtrStr(s.StdDev),
				floatPtrStr(s.Integral), intPtrStr(s.Count),
			})
		}

		resourceURI, err := PushResult(store, dataServerBaseURL,
			fmt.Sprintf("AnalogSummary: %s", fqn),
			[]dataserver.Field{
				{Name: "FQN", Type: "string"},
				{Name: "StartDateTime", Type: "datetime"},
				{Name: "EndDateTime", Type: "datetime"},
				{Name: "RetrievalMode", Type: "string"},
				{Name: "Resolution", Type: "number"},
				{Name: "SliceBy", Type: "string"},
				{Name: "SliceByValue", Type: "string"},
				{Name: "OPCQuality", Type: "number"},
				{Name: "PercentGood", Type: "number"},
				{Name: "First", Type: "number"},
				{Name: "FirstDateTime", Type: "datetime"},
				{Name: "Last", Type: "number"},
				{Name: "LastDateTime", Type: "datetime"},
				{Name: "Minimum", Type: "number"},
				{Name: "MinDateTime", Type: "datetime"},
				{Name: "Maximum", Type: "number"},
				{Name: "MaxDateTime", Type: "datetime"},
				{Name: "Average", Type: "number"},
				{Name: "StandardDeviation", Type: "number"},
				{Name: "Integral", Type: "number"},
				{Name: "Count", Type: "number"},
			},
			allRows)
		if err != nil {
			logger.Warn("failed to push result to data server", "error", err)
			resourceURI = ""
		}

		logger.Info("ok", "rows", len(rows), "fqn", fqn)

		return formatResult(rows, byteLimit, func(rows []historian.AnalogSummaryValue) string {
			var preview strings.Builder
			preview.WriteString("| FQN | StartDateTime | EndDateTime | Min | Max | Avg | StdDev | Integral | Count | First | Last |\n")
			preview.WriteString("|-----|--------------|------------|-----|-----|-----|-------|---------|-------|-------|------|\n")
			for _, s := range rows {
				fmt.Fprintf(&preview, "| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
					s.FQN, s.StartDateTime, s.EndDateTime,
					floatPtrStr(s.Minimum), floatPtrStr(s.Maximum),
					floatPtrStr(s.Average), floatPtrStr(s.StdDev),
					floatPtrStr(s.Integral), intPtrStr(s.Count),
					floatPtrStr(s.First), floatPtrStr(s.Last))
			}
			return preview.String()
		}, resourceURI, "analog-summary", result.Count), nil
	})
}
