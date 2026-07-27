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

func RegisterReadTrends(server *mcp.Server, client *historian.Client, logger *slog.Logger) {
	inputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tag_id":     map[string]any{"type": "string", "description": "Tag FQN"},
			"start_time": map[string]any{"type": "string", "description": "Start time"},
			"end_time":   map[string]any{"type": "string", "description": "End time"},
			"retrieval_mode": map[string]any{
				"type":        "string",
				"enum":        []string{"Average", "Cyclic", "Integral", "Minimum", "Maximum", "BestFit", "Delta", "Interpolated", "Slope", "Counter", "Full"},
				"description": "How data is calculated for retrieval",
			},
			"resolution_ms":  map[string]any{"type": "number", "description": "Granularity in ms"},
			"max_results":    map[string]any{"type": "number", "description": "Max rows (default 100)"},
			"filters": map[string]any{
				"type": "array",
				"description": "Additional Filters using OData expressions. Array of groups (AND-combined across groups). Each group has \"and\" or \"or\" with conditions. Condition: {\"field\":\"...\", \"operator\":\"...\", \"value\":...}. Operators: eq,ne,gt,ge,lt,le (str|num), startsWith,endsWith,contains (str), in (array), has (str). Example: [{\"and\":[{\"field\":\"FQN\",\"operator\":\"startsWith\",\"value\":\"CDE\"}]}] -> startswith(FQN,CDE)",
				"items": map[string]any{"type": "object"},
			},
		},
	}

	server.AddTool(&mcp.Tool{
		Name:        "read_trends",
		Description: "Retrieve time-series process values for a historian tag over a time range",
		InputSchema: inputSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		var groups []types.FilterGroupDef

		// Build FQN filter from convenience param
		tagID := getStringArg(args, "tag_id")
		if tagID != "" {
			groups = append(groups, types.FilterGroupDef{
				And: []types.FilterCondition{{Field: "FQN", Operator: "eq", Value: tagID}},
			})
		}

		// Build time range filters from convenience params
		startTime := getStringArg(args, "start_time")
		endTime := getStringArg(args, "end_time")
		var timeConds []types.FilterCondition
		if startTime != "" {
			timeConds = append(timeConds, types.FilterCondition{Field: "DateTime", Operator: "ge", Value: startTime})
		}
		if endTime != "" {
			timeConds = append(timeConds, types.FilterCondition{Field: "DateTime", Operator: "le", Value: endTime})
		}
		if len(timeConds) > 0 {
			groups = append(groups, types.FilterGroupDef{And: timeConds})
		}

		retrievalMode := getStringArg(args, "retrieval_mode")
		if retrievalMode != "" && !isValidRetrievalMode(retrievalMode) {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("unsupported retrieval mode: %s", retrievalMode)}},
			}, nil
		}
		userFilters := parseFilters(args["filters"])
		groups = append(groups, userFilters...)
		resolutionMS := getIntArg(args, "resolution_ms", 3600000)
		maxResults := getIntArg(args, "max_results", 100)

		result, err := endpoints.GetProcessValues(ctx, client, groups, retrievalMode, resolutionMS, maxResults)

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
		preview.WriteString("| FQN | DateTime | Value | Quality | TagPath |\n")
		preview.WriteString("|-----|----------|-------|---------|--------|\n")
		for _, pv := range capped {
			fmt.Fprintf(&preview, "| %s | %s | %g | %s | %s |\n",
				pv.FQN, pv.DateTime, pv.Value, pv.Quality, pv.TagPath)
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

		logger.Info("read_trends", "rows", len(rows), "tag_id", tagID)

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
