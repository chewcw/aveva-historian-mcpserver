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

func RegisterTagProfiler(server *mcp.Server, client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "tag_profiler",
		Title:       "Tag Metadata Profile",
		Description: "Return the full metadata profile of a single historian tag by exact FQN: name, type, engineering unit, source, engineering limits, interpolation type, discrete messages, alias, and description.",
		Arguments: []*mcp.PromptArgument{
			{Name: "fqn", Description: "Exact tag FQN", Required: true},
		},
	}, handleTagProfiler(client, store, dataServerBaseURL, logger, byteLimit))
}

func handleTagProfiler(client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) func(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := promptArgs(req)
		fqn, err := requiredArg(args, "fqn")
		if err != nil {
			return errResult(err.Error()), nil
		}

		top := 1
		groups := []types.FilterGroupDef{{And: []types.FilterCondition{{Field: "FQN", Operator: "eq", Value: fqn}}}}
		result, err := endpoints.GetTags(ctx, client, groups, types.QueryOptions{Top: &top})
		if err != nil {
			return nil, fmt.Errorf("fetch tag profile: %w", err)
		}
		if len(result.Value) == 0 {
			return errResult(fmt.Sprintf("No tag matched %q", fqn)), nil
		}
		t := result.Value[0]

		var b strings.Builder
		fmt.Fprintf(&b, "## Tag Profile: %s\n\n", t.FQN)
		b.WriteString("| Field | Value |\n|-------|-------|\n")
		fmt.Fprintf(&b, "| FQN | %s |\n", t.FQN)
		fmt.Fprintf(&b, "| TagName | %s |\n", t.TagName)
		fmt.Fprintf(&b, "| TagType | %s |\n", t.TagType)
		fmt.Fprintf(&b, "| EngUnit | %s |\n", t.EngUnit)
		fmt.Fprintf(&b, "| Source | %s |\n", t.Source)
		fmt.Fprintf(&b, "| EngUnitMin | %s |\n", fmtFloat(t.EngUnitMin))
		fmt.Fprintf(&b, "| EngUnitMax | %s |\n", fmtFloat(t.EngUnitMax))
		fmt.Fprintf(&b, "| InterpolationType | %s |\n", t.InterpolationType)
		fmt.Fprintf(&b, "| MessageOff | %s |\n", t.MessageOff)
		fmt.Fprintf(&b, "| MessageOn | %s |\n", t.MessageOn)
		fmt.Fprintf(&b, "| Alias | %s |\n", t.Alias)
		fmt.Fprintf(&b, "| Description | %s |\n", t.Description)

		ref, err := pushRef(store, dataServerBaseURL, "TagProfile: "+fqn, "profile",
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
			[][]string{{t.FQN, t.TagName, t.TagType, t.EngUnit, t.Source,
				fmtFloat(t.EngUnitMin), fmtFloat(t.EngUnitMax), t.InterpolationType,
				t.MessageOff, t.MessageOn, t.Alias, t.Description}})
		if err != nil {
			logger.Warn("failed to push result to data server", "error", err)
			ref = resourceRef{}
		}

		logger.Info("prompt ok", "name", "tag_profiler", "fqn", fqn)
		return resultMessage(b.String(), ref), nil
	}
}
