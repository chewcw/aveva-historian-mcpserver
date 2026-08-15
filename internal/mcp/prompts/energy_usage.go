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

// defaultEnergySliceMs is the per-slice period used when resolution_ms is omitted.
const defaultEnergySliceMs = 3600000

func RegisterEnergyUsage(server *mcp.Server, client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "energy_usage",
		Title:       "Energy Usage Analysis",
		Description: "Return per-period integral (energy) statistics for one tag over a time range, with a configurable slice length.",
		Arguments: []*mcp.PromptArgument{
			{Name: "fqn", Description: "Tag FQN", Required: true},
			{Name: "start_date_time", Description: "Start of the range", Required: true},
			{Name: "end_date_time", Description: "End of the range", Required: true},
			{Name: "resolution_ms", Description: "Slice length in ms (default 3600000)"},
		},
	}, handleEnergyUsage(client, store, dataServerBaseURL, logger, byteLimit))
}

func handleEnergyUsage(client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) func(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
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
		if start > end {
			return errResult("start_date_time must be before end_date_time"), nil
		}
		resolution := defaultEnergySliceMs
		sliceNote := "(default)"
		if v := strings.TrimSpace(args["resolution_ms"]); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				return errResult("resolution_ms must be a positive integer"), nil
			}
			resolution = n
			sliceNote = ""
		}

		conds := []types.FilterCondition{
			{Field: "FQN", Operator: "eq", Value: fqn},
			{Field: "StartDateTime", Operator: "ge", Value: start},
			{Field: "EndDateTime", Operator: "le", Value: end},
			{Field: "Resolution", Operator: "eq", Value: resolution},
		}
		result, err := endpoints.GetAnalogSummary(ctx, client, []types.FilterGroupDef{{And: conds}}, types.QueryOptions{})
		if err != nil {
			return nil, fmt.Errorf("fetch energy summary: %w", err)
		}
		rows := result.Value
		if rows == nil {
			rows = []historian.AnalogSummaryValue{}
		}
		if len(rows) == 0 {
			return errResult("No energy data found for the given tag and range"), nil
		}

		var total float64
		for _, s := range rows {
			if s.Integral != nil {
				total += *s.Integral
			}
		}
		var b strings.Builder
		fmt.Fprintf(&b, "## Energy Usage: %s\n\n", fqn)
		fmt.Fprintf(&b, "Range: %s .. %s | Slice: %d ms%s\n\n", start, end, resolution, sliceNote)
		b.WriteString("| Slice Start | Slice End | Integral | Avg | Max | Min | %Good |\n")
		b.WriteString("|-------------|-----------|----------|-----|-----|-----|-------|\n")
		for _, s := range capRows(rows, 100) {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n",
				s.StartDateTime, s.EndDateTime, fmtFloat(s.Integral), fmtFloat(s.Average),
				fmtFloat(s.Maximum), fmtFloat(s.Minimum), fmtFloat(s.PercentGood))
		}
		fmt.Fprintf(&b, "\n**Total integral over range: %g**\n", total)

		allRows := make([][]string, 0, len(rows))
		for _, s := range rows {
			allRows = append(allRows, []string{s.FQN, s.StartDateTime, s.EndDateTime,
				fmtFloat(s.Integral), fmtFloat(s.Average), fmtFloat(s.Maximum),
				fmtFloat(s.Minimum), fmtFloat(s.PercentGood), fmtInt(s.Count)})
		}
		ref, err := pushRef(store, dataServerBaseURL, "EnergyUsage: "+fqn, "energy",
			[]dataserver.Field{
				{Name: "FQN", Type: "string"},
				{Name: "StartDateTime", Type: "datetime"},
				{Name: "EndDateTime", Type: "datetime"},
				{Name: "Integral", Type: "number"},
				{Name: "Average", Type: "number"},
				{Name: "Maximum", Type: "number"},
				{Name: "Minimum", Type: "number"},
				{Name: "PercentGood", Type: "number"},
				{Name: "Count", Type: "number"},
			},
			allRows)
		if err != nil {
			logger.Warn("failed to push result to data server", "error", err)
			ref = resourceRef{}
		}

		logger.Info("prompt ok", "name", "energy_usage", "fqn", fqn, "slices", len(rows))
		return resultMessage(b.String(), ref), nil
	}
}
