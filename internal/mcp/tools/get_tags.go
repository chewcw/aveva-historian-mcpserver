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

func RegisterGetTags(server *mcp.Server, client *historian.Client, logger *slog.Logger) {
	inputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filter": map[string]any{
				"type":        "string",
				"description": "OData $filter expression to narrow the tag list",
			},
			"top": map[string]any{
				"type":        "number",
				"description": "Maximum number of tags to return (default 50)",
			},
			"skip": map[string]any{
				"type":        "number",
				"description": "Number of tags to skip for pagination (default 0)",
			},
		},
	}

	server.AddTool(&mcp.Tool{
		Name:        "get_tags",
		Description: "Retrieve a list of historian tags matching an optional filter",
		InputSchema: inputSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		filter := getStringArg(args, "filter")
		top := getIntArg(args, "top", 50)
		skip := getIntArg(args, "skip", 0)

		result, err := endpoints.GetTags(ctx, client, filter, top, skip)
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
		capped := capRows(rows, 100)

		var preview strings.Builder
		preview.WriteString("| TagName | FQN | Type | Unit | Description |\n")
		preview.WriteString("|---------|-----|------|------|-------------|\n")
		for _, t := range capped {
			fmt.Fprintf(&preview, "| %s | %s | %s | %s | %s |\n",
				t.TagName, t.FQN, t.TagType, t.Unit, t.Description)
		}

		resourceURI := fmt.Sprintf("historians://%s/tags/%s", client.BaseURL(), uuid.New().String())

		dualResult := types.DualToolResult{
			Preview:     preview.String(),
			RowCount:    len(capped),
			ResourceURI: resourceURI,
		}
		if result.Count != nil {
			dualResult.TotalCount = result.Count
		}

		dualJSON, _ := json.Marshal(dualResult)

		logger.Info("get_tags", "rows", len(rows), "top", top)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(dualJSON)},
				&mcp.ResourceLink{URI: resourceURI, Name: "tags"},
			},
		}, nil
	})
}
