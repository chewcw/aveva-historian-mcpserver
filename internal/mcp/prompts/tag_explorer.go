package prompts

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

func RegisterTagExplorer(server *mcp.Server, client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "tag_explorer",
		Title:       "Discover Historian Tags",
		Description: "Search or browse the historian tag catalog by free-text search, tag type, or data source. Returns matching tags with FQN, name, type, engineering unit, source, and description.",
		Arguments: []*mcp.PromptArgument{
			{Name: "search", Description: "Free-text search (optional)"},
			{Name: "tag_type", Description: "Tag type: Analog(1), Discrete(2), String(3), Event(5), Summary(7) (optional)"},
			{Name: "source", Description: "Data source filter (optional)"},
			{Name: "top", Description: "Max rows (default 50, max 1000)"},
		},
	}, handleTagExplorer(client, store, dataServerBaseURL, logger, byteLimit))
}

func handleTagExplorer(client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) func(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := promptArgs(req)

		var convConds []types.FilterCondition
		if v := strings.TrimSpace(args["tag_type"]); v != "" {
			convConds = append(convConds, types.FilterCondition{Field: "TagType", Operator: "eq", Value: v})
		}
		if v := strings.TrimSpace(args["source"]); v != "" {
			convConds = append(convConds, types.FilterCondition{Field: "Source", Operator: "eq", Value: v})
		}
		var groups []types.FilterGroupDef
		if len(convConds) > 0 {
			groups = append(groups, types.FilterGroupDef{And: convConds})
		}

		top := clampInt(intArg(args, "top", 50), 1, 1000)
		opts := types.QueryOptions{Top: &top}
		if search := strings.TrimSpace(args["search"]); search != "" {
			opts.Search = &search
		}

		result, err := endpoints.GetTags(ctx, client, groups, opts)
		if err != nil {
			return nil, fmt.Errorf("fetch tags: %w", err)
		}
		rows := result.Value
		if rows == nil {
			rows = []historian.Tag{}
		}
		if len(rows) == 0 {
			return errResult("No tags matched the given criteria"), nil
		}

		var b strings.Builder
		b.WriteString("| FQN | TagName | TagType | EngUnit | Source | Description |\n")
		b.WriteString("|-----|---------|---------|--------|--------|-------------|\n")
		for _, t := range capRows(rows, 100) {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n",
				t.FQN, t.TagName, t.TagType, t.EngUnit, t.Source, t.Description)
		}

		allRows := make([][]string, 0, len(rows))
		for _, t := range rows {
			allRows = append(allRows, []string{t.FQN, t.TagName, t.TagType, t.EngUnit, t.Source,
				fmtFloat(t.EngUnitMin), fmtFloat(t.EngUnitMax), t.InterpolationType,
				t.MessageOff, t.MessageOn, t.Alias, t.Description})
		}
		ref, err := pushRef(store, dataServerBaseURL, "TagExplorer", "tags",
			[]dataserver.Field{
				{Name: "FQN", Type: "string"},
				{Name: "TagName", Type: "string"},
				{Name: "TagType", Type: "string"},
				{Name: "EngUnit", Type: "string"},
				{Name: "Source", Type: "string"},
				{Name: "EngUnitMin", Type: "number"},
				{Name: "EngUnitMax", Type: "number"},
				{Name: "InterpolationType", Type: "string"},
				{Name: "MessageOff", Type: "string"},
				{Name: "MessageOn", Type: "string"},
				{Name: "Alias", Type: "string"},
				{Name: "Description", Type: "string"},
			},
			allRows)
		if err != nil {
			logger.Warn("failed to push result to data server", "error", err)
			ref = resourceRef{}
		}

		logger.Info("prompt ok", "name", "tag_explorer", "rows", len(rows), "top", top)
		return resultMessage(b.String(), ref), nil
	}
}
