package prompts

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/chewcw/aveva-historian-mcpserver/internal/dataserver"
	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/historian/endpoints"
	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	minCompareTags = 2
	maxCompareTags = 10
)

func RegisterCompareTags(server *mcp.Server, client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "compare_tags",
		Title:       "Compare Tags",
		Description: "Return side-by-side analog summary statistics (min, max, avg, std dev, integral, count, first, last) for 2-10 tags over the same time range.",
		Arguments: []*mcp.PromptArgument{
			{Name: "fqn", Description: "Comma-separated tag FQNs (2-10)", Required: true},
			{Name: "start_date_time", Description: "Start of the range", Required: true},
			{Name: "end_date_time", Description: "End of the range", Required: true},
			{Name: "resolution_ms", Description: "Cycle resolution in ms (optional)"},
		},
	}, handleCompareTags(client, store, dataServerBaseURL, logger, byteLimit))
}

func handleCompareTags(client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) func(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := promptArgs(req)
		fqnRaw, err := requiredArg(args, "fqn")
		if err != nil {
			return errResult(err.Error()), nil
		}
		start, err := requiredArg(args, "start_date_time")
		if err != nil {
			return errResult(err.Error()), nil
		}
		end, err := requiredArg(args, "end_date_time")
		if err != nil {
			return errResult(err.Error()), nil
		}
		fqns := splitCSV(fqnRaw)
		if len(fqns) < minCompareTags {
			return errResult(fmt.Sprintf("compare_tags needs at least %d FQNs, got %d", minCompareTags, len(fqns))), nil
		}
		if len(fqns) > maxCompareTags {
			return errResult(fmt.Sprintf("Too many FQNs: %d (max %d)", len(fqns), maxCompareTags)), nil
		}
		if start > end {
			return errResult("start_date_time must be before end_date_time"), nil
		}
		var resolution *int
		if v := strings.TrimSpace(args["resolution_ms"]); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				return errResult("resolution_ms must be a positive integer"), nil
			}
			resolution = &n
		}

		rows := make([]historian.AnalogSummaryValue, 0, len(fqns))
		for _, fqn := range fqns {
			conds := []types.FilterCondition{
				{Field: "FQN", Operator: "eq", Value: fqn},
				{Field: "StartDateTime", Operator: "ge", Value: start},
				{Field: "EndDateTime", Operator: "le", Value: end},
			}
			if resolution != nil {
				conds = append(conds, types.FilterCondition{Field: "Resolution", Operator: "eq", Value: *resolution})
			}
			groups := []types.FilterGroupDef{{And: conds}}
			result, err := endpoints.GetAnalogSummary(ctx, client, groups, types.QueryOptions{})
			if err != nil {
				return nil, fmt.Errorf("fetch summary for %s: %w", fqn, err)
			}
			if len(result.Value) > 0 {
				rows = append(rows, result.Value[0])
			}
		}
		if len(rows) == 0 {
			return errResult("No summary data found for the given tags and range"), nil
		}

		var b strings.Builder
		b.WriteString("| FQN | Min | Max | Avg | StdDev | Integral | Count | First | Last |\n")
		b.WriteString("|-----|-----|-----|-----|--------|----------|-------|-------|------|\n")
		for _, s := range rows {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
				s.FQN, fmtFloat(s.Minimum), fmtFloat(s.Maximum), fmtFloat(s.Average),
				fmtFloat(s.StdDev), fmtFloat(s.Integral), fmtInt(s.Count), fmtFloat(s.First), fmtFloat(s.Last))
		}

		allRows := make([][]string, 0, len(rows))
		for _, s := range rows {
			allRows = append(allRows, []string{s.FQN, s.StartDateTime, s.EndDateTime,
				fmtFloat(s.Minimum), fmtFloat(s.Maximum), fmtFloat(s.Average),
				fmtFloat(s.StdDev), fmtFloat(s.Integral), fmtInt(s.Count),
				fmtFloat(s.First), fmtFloat(s.Last)})
		}
		ref, err := pushRef(store, dataServerBaseURL, "CompareTags", "summary",
			[]dataserver.Field{
				{Name: "FQN", Type: "string"},
				{Name: "StartDateTime", Type: "datetime"},
				{Name: "EndDateTime", Type: "datetime"},
				{Name: "Minimum", Type: "number"},
				{Name: "Maximum", Type: "number"},
				{Name: "Average", Type: "number"},
				{Name: "StandardDeviation", Type: "number"},
				{Name: "Integral", Type: "number"},
				{Name: "Count", Type: "number"},
				{Name: "First", Type: "number"},
				{Name: "Last", Type: "number"},
			},
			allRows)
		if err != nil {
			logger.Warn("failed to push result to data server", "error", err)
			ref = resourceRef{}
		}

		logger.Info("prompt ok", "name", "compare_tags", "tags", len(fqns))
		return resultMessage(b.String(), ref), nil
	}
}
