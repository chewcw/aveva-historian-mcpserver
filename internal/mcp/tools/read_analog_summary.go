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
			"tag_id":        map[string]any{"type": "string", "description": "Tag FQN"},
			"start_time":    map[string]any{"type": "string", "description": "Start time"},
			"end_time":      map[string]any{"type": "string", "description": "End time"},
			"resolution_ms": map[string]any{"type": "number", "description": "Granularity in ms"},
			"max_results":   map[string]any{"type": "number", "description": "Max rows (default 100)"},
			"filters": map[string]any{
				"type":        "array",
				"description": "Additional Filters using OData expressions. Array of groups (AND-combined across groups). Each group has \"and\" or \"or\" with conditions. Condition: {\"field\":\"...\", \"operator\":\"...\", \"value\":...}. Operators: eq,ne,gt,ge,lt,le (str|num), startsWith,endsWith,contains (str), in (array), has (str). Example: [{\"and\":[{\"field\":\"FQN\",\"operator\":\"startsWith\",\"value\":\"CDE\"}]}] -> startswith(FQN,CDE)",
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
		Name:        "read_analog_summary",
		Description: "Retrieve analog summary statistics for a historian tag over a time range",
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

		result, err := endpoints.GetAnalogSummary(ctx, client, groups, resolutionMS, maxResults)
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
		preview.WriteString("| FQN | StartDateTime | EndDateTime | Min | Max | Avg | StdDev | Count | TagPath |\n")
		preview.WriteString("|-----|--------------|------------|-----|-----|-----|-------|-------|--------|\n")
		for _, s := range capped {
			fmt.Fprintf(&preview, "| %s | %s | %s | %g | %g | %g | %g | %d | %s |\n",
				s.FQN, s.StartDateTime, s.EndDateTime, s.Minimum, s.Maximum, s.Average, s.StdDev, s.Count, s.TagPath)
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
