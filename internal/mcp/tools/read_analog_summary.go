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

func RegisterReadAnalogSummary(server *mcp.Server, client *historian.Client, logger *slog.Logger) {
	inputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tag_id":        map[string]any{"type": "string", "description": "Tag FQN (optional)"},
			"start_time":    map[string]any{"type": "string", "description": "Start time (optional)"},
			"end_time":      map[string]any{"type": "string", "description": "End time (optional)"},
			"resolution_ms": map[string]any{"type": "number", "description": "Resolution in ms (optional)"},
			"max_results":   map[string]any{"type": "number", "description": "Max rows (optional, default 100)"},
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
		var groups []types.FilterGroupDef

		tagID := getStringArg(args, "tag_id")
		if tagID != "" {
			groups = append(groups, types.FilterGroupDef{
				And: []types.FilterCondition{{Field: "FQN", Operator: "eq", Value: tagID}},
			})
		}

		startTime := getStringArg(args, "start_time")
		endTime := getStringArg(args, "end_time")
		var timeConds []types.FilterCondition
		if startTime != "" {
			timeConds = append(timeConds, types.FilterCondition{Field: "StartDateTime", Operator: "ge", Value: startTime})
		}
		if endTime != "" {
			timeConds = append(timeConds, types.FilterCondition{Field: "EndDateTime", Operator: "le", Value: endTime})
		}
		if len(timeConds) > 0 {
			groups = append(groups, types.FilterGroupDef{And: timeConds})
		}

		userFilters := parseFilters(args["filters"])
		groups = append(groups, userFilters...)

		resolutionMS := getIntArg(args, "resolution_ms", 3600000)
		maxResults := getIntArg(args, "max_results", 100)

		extraParams := endpoints.AnalogSummaryParams{
			RetrievalMode: getOptionalStringArg(args, "retrieval_mode"),
			SliceBy:       getOptionalStringArg(args, "slice_by"),
			SliceByValue:  getOptionalStringArg(args, "slice_by_value"),
			OPCQuality:    getOptionalIntArg(args, "opc_quality"),
			PercentGood:   getOptionalFloatArg(args, "percent_good"),
		}

		result, err := endpoints.GetAnalogSummary(ctx, client, groups, resolutionMS, maxResults, extraParams)
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
		capped := capRows(rows, 100)

		var preview strings.Builder
		preview.WriteString("| FQN | StartDateTime | EndDateTime | Min | Max | Avg | StdDev | Integral | Count | First | Last |\n")
		preview.WriteString("|-----|--------------|------------|-----|-----|-----|-------|---------|-------|-------|------|\n")
		for _, s := range capped {
			fmt.Fprintf(&preview, "| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
				s.FQN, s.StartDateTime, s.EndDateTime,
				floatPtrStr(s.Minimum), floatPtrStr(s.Maximum),
				floatPtrStr(s.Average), floatPtrStr(s.StdDev),
				floatPtrStr(s.Integral), intPtrStr(s.Count),
				floatPtrStr(s.First), floatPtrStr(s.Last))
		}

		resourceURI := fmt.Sprintf("historians://%s/summary/%s", client.BaseURL(), uuid.New().String())

		dualResult := types.DualToolResult{
			Preview:     preview.String(),
			RowCount:    len(capped),
			ResourceURI: resourceURI,
		}
		if result.Count != nil {
			dualResult.TotalCount = result.Count
		}

		dualJSON, _ := json.Marshal(dualResult)

		logger.Info("read_analog_summary", "rows", len(rows), "tag_id", tagID)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(dualJSON)},
				&mcp.ResourceLink{URI: resourceURI, Name: "analog-summary"},
			},
		}, nil
	})
}
