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
			"tag_id": map[string]any{
				"type":        "string",
				"description": "Fully qualified tag name (FQN) to retrieve process values for",
			},
			"start_time": map[string]any{
				"type":        "string",
				"description": "Start time for the trend data (ISO 8601 format)",
			},
			"end_time": map[string]any{
				"type":        "string",
				"description": "End time for the trend data (ISO 8601 format)",
			},
			"retrieval_mode": map[string]any{
				"type":        "string",
				"description": "Retrieval mode (e.g., RawMode, SnapshotMode)",
			},
			"resolution_ms": map[string]any{
				"type":        "number",
				"description": "Resolution in milliseconds for interpolation",
			},
			"max_results": map[string]any{
				"type":        "number",
				"description": "Maximum number of process values to return (default 100)",
			},
		},
		"required": []string{"tag_id", "start_time", "end_time"},
	}

	server.AddTool(&mcp.Tool{
		Name:        "read_trends",
		Description: "Retrieve time-series process values for a historian tag over a time range",
		InputSchema: inputSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		tagID := getStringArg(args, "tag_id")
		startTime := getStringArg(args, "start_time")
		endTime := getStringArg(args, "end_time")

		if tagID == "" || startTime == "" || endTime == "" {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: "tag_id, start_time, and end_time are required"}},
			}, nil
		}

		retrievalMode := getStringArg(args, "retrieval_mode")
		resolutionMS := getIntArg(args, "resolution_ms", 0)
		maxResults := getIntArg(args, "max_results", 100)

		result, err := endpoints.GetProcessValues(ctx, client, tagID, startTime, endTime, retrievalMode, resolutionMS, maxResults)
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
