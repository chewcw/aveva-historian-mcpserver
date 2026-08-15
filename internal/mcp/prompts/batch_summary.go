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

const maxBatchSliceBy = 10

func RegisterBatchSummary(server *mcp.Server, client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "batch_summary",
		Title:       "Batch Summary",
		Description: "Return per-batch analog summary statistics for one tag, where batch boundaries are defined by one or more slice_by tags.",
		Arguments: []*mcp.PromptArgument{
			{Name: "fqn", Description: "Tag FQN", Required: true},
			{Name: "start_date_time", Description: "Start of the range", Required: true},
			{Name: "end_date_time", Description: "End of the range", Required: true},
			{Name: "slice_by", Description: "Comma-separated FQNs defining batch boundaries (max 10)", Required: true},
			{Name: "slice_by_value", Description: "Filter criterion for SliceBy results (optional)"},
		},
	}, handleBatchSummary(client, store, dataServerBaseURL, logger, byteLimit))
}

func handleBatchSummary(client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) func(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := promptArgs(req)
		fqn, err := requiredArg(args, "fqn")
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
		sliceByRaw, err := requiredArg(args, "slice_by")
		if err != nil {
			return errResult(err.Error()), nil
		}
		sliceBy := splitCSV(sliceByRaw)
		if len(sliceBy) == 0 {
			return errResult("Missing required argument: slice_by"), nil
		}
		if len(sliceBy) > maxBatchSliceBy {
			return errResult(fmt.Sprintf("Too many FQNs: %d (max %d)", len(sliceBy), maxBatchSliceBy)), nil
		}
		if start > end {
			return errResult("start_date_time must be before end_date_time"), nil
		}

		conds := []types.FilterCondition{
			{Field: "FQN", Operator: "eq", Value: fqn},
			{Field: "StartDateTime", Operator: "ge", Value: start},
			{Field: "EndDateTime", Operator: "le", Value: end},
			{Field: "SliceBy", Operator: "eq", Value: sliceByRaw},
		}
		if v := strings.TrimSpace(args["slice_by_value"]); v != "" {
			conds = append(conds, types.FilterCondition{Field: "SliceByValue", Operator: "eq", Value: v})
		}
		result, err := endpoints.GetAnalogSummary(ctx, client, []types.FilterGroupDef{{And: conds}}, types.QueryOptions{})
		if err != nil {
			return nil, fmt.Errorf("fetch batch summary: %w", err)
		}
		rows := result.Value
		if rows == nil {
			rows = []historian.AnalogSummaryValue{}
		}
		if len(rows) == 0 {
			return errResult("No batch data found for the given tag and range"), nil
		}

		var b strings.Builder
		fmt.Fprintf(&b, "## Batch Summary: %s\n\n", fqn)
		fmt.Fprintf(&b, "Range: %s .. %s | SliceBy: %s\n", start, end, sliceByRaw)
		if v := strings.TrimSpace(args["slice_by_value"]); v != "" {
			fmt.Fprintf(&b, "SliceByValue: %s\n", v)
		}
		b.WriteString("\n| Start | End | First | Last | Min | Max | Avg | StdDev | Integral | Count |\n")
		b.WriteString("|-------|-----|-------|------|-----|-----|-----|--------|----------|-------|\n")
		for _, s := range capRows(rows, 100) {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
				s.StartDateTime, s.EndDateTime, fmtFloat(s.First), fmtFloat(s.Last),
				fmtFloat(s.Minimum), fmtFloat(s.Maximum), fmtFloat(s.Average),
				fmtFloat(s.StdDev), fmtFloat(s.Integral), fmtInt(s.Count))
		}

		allRows := make([][]string, 0, len(rows))
		for _, s := range rows {
			allRows = append(allRows, []string{s.FQN, s.StartDateTime, s.EndDateTime,
				fmtFloat(s.First), fmtFloat(s.Last), fmtFloat(s.Minimum),
				fmtFloat(s.Maximum), fmtFloat(s.Average), fmtFloat(s.StdDev),
				fmtFloat(s.Integral), fmtInt(s.Count), s.SliceByValue})
		}
		ref, err := pushRef(store, dataServerBaseURL, "BatchSummary: "+fqn, "batch",
			[]dataserver.Field{
				{Name: "FQN", Type: "string"},
				{Name: "StartDateTime", Type: "datetime"},
				{Name: "EndDateTime", Type: "datetime"},
				{Name: "First", Type: "number"},
				{Name: "Last", Type: "number"},
				{Name: "Minimum", Type: "number"},
				{Name: "Maximum", Type: "number"},
				{Name: "Average", Type: "number"},
				{Name: "StandardDeviation", Type: "number"},
				{Name: "Integral", Type: "number"},
				{Name: "Count", Type: "number"},
				{Name: "SliceByValue", Type: "string"},
			},
			allRows)
		if err != nil {
			logger.Warn("failed to push result to data server", "error", err)
			ref = resourceRef{}
		}

		logger.Info("prompt ok", "name", "batch_summary", "fqn", fqn, "batches", len(rows))
		return resultMessage(b.String(), ref), nil
	}
}
