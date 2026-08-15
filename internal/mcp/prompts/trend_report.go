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

func RegisterTrendReport(server *mcp.Server, client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "trend_report",
		Title:       "Tag Trend Report",
		Description: "Return analog summary statistics plus a raw value trend for one tag over a time range, with optional resolution and retrieval mode.",
		Arguments: []*mcp.PromptArgument{
			{Name: "fqn", Description: "Tag FQN", Required: true},
			{Name: "start_date_time", Description: "Start of the range", Required: true},
			{Name: "end_date_time", Description: "End of the range", Required: true},
			{Name: "resolution_ms", Description: "Granularity in ms (optional)"},
			{Name: "retrieval_mode", Description: "Average|Cyclic|Integral|Minimum|Maximum|BestFit|Delta|Interpolated|Slope|Counter|Full (optional)"},
		},
	}, handleTrendReport(client, store, dataServerBaseURL, logger, byteLimit))
}

func handleTrendReport(client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) func(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
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
		var resolution *int
		if v := strings.TrimSpace(args["resolution_ms"]); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				return errResult("resolution_ms must be a positive integer"), nil
			}
			resolution = &n
		}
		var mode *string
		if v := strings.TrimSpace(args["retrieval_mode"]); v != "" {
			if !isValidRetrievalMode(v) {
				return errResult(fmt.Sprintf("unsupported retrieval mode: %s", v)), nil
			}
			mode = &v
		}

		baseConds := []types.FilterCondition{
			{Field: "FQN", Operator: "eq", Value: fqn},
		}
		if resolution != nil {
			baseConds = append(baseConds, types.FilterCondition{Field: "Resolution", Operator: "eq", Value: *resolution})
		}
		if mode != nil {
			baseConds = append(baseConds, types.FilterCondition{Field: "RetrievalMode", Operator: "eq", Value: *mode})
		}

		summaryConds := append([]types.FilterCondition{
			{Field: "StartDateTime", Operator: "ge", Value: start},
			{Field: "EndDateTime", Operator: "le", Value: end},
		}, baseConds...)
		summaryRes, err := endpoints.GetAnalogSummary(ctx, client,
			[]types.FilterGroupDef{{And: summaryConds}}, types.QueryOptions{})
		if err != nil {
			return nil, fmt.Errorf("fetch analog summary: %w", err)
		}

		valueConds := append([]types.FilterCondition{
			{Field: "DateTime", Operator: "ge", Value: start},
			{Field: "DateTime", Operator: "le", Value: end},
		}, baseConds...)
		top := 100
		valuesRes, err := endpoints.GetProcessValues(ctx, client,
			[]types.FilterGroupDef{{And: valueConds}}, types.QueryOptions{Top: &top})
		if err != nil {
			return nil, fmt.Errorf("fetch process values: %w", err)
		}
		values := valuesRes.Value
		if values == nil {
			values = []historian.ProcessValue{}
		}

		var b strings.Builder
		fmt.Fprintf(&b, "## Trend Report: %s\n\n", fqn)
		if len(summaryRes.Value) > 0 {
			s := summaryRes.Value[0]
			b.WriteString("| Stat | Value | Timestamp |\n|---|---|---|\n")
			fmt.Fprintf(&b, "| First | %s | %s |\n", fmtFloat(s.First), s.FirstDateTime)
			fmt.Fprintf(&b, "| Last | %s | %s |\n", fmtFloat(s.Last), s.LastDateTime)
			fmt.Fprintf(&b, "| Min | %s | %s |\n", fmtFloat(s.Minimum), s.MinDateTime)
			fmt.Fprintf(&b, "| Max | %s | %s |\n", fmtFloat(s.Maximum), s.MaxDateTime)
			fmt.Fprintf(&b, "| Avg | %s | |\n", fmtFloat(s.Average))
			fmt.Fprintf(&b, "| StdDev | %s | |\n", fmtFloat(s.StdDev))
			fmt.Fprintf(&b, "| Integral | %s | |\n", fmtFloat(s.Integral))
			fmt.Fprintf(&b, "| Count | %s | |\n", fmtInt(s.Count))
		} else {
			fmt.Fprintf(&b, "No summary data for range %s .. %s\n", start, end)
		}
		if len(values) > 0 {
			b.WriteString("\n| DateTime | Value | OpcQuality |\n|---|---|---|\n")
			for _, pv := range capRows(values, 100) {
				fmt.Fprintf(&b, "| %s | %s | %s |\n", pv.DateTime, fmtFloat(pv.Value), fmtInt(pv.OpcQuality))
			}
		}

		summaryRows := make([][]string, 0, len(summaryRes.Value))
		for _, s := range summaryRes.Value {
			summaryRows = append(summaryRows, []string{s.FQN, s.StartDateTime, s.EndDateTime,
				fmtFloat(s.Minimum), fmtFloat(s.Maximum), fmtFloat(s.Average),
				fmtFloat(s.StdDev), fmtFloat(s.Integral), fmtInt(s.Count),
				fmtFloat(s.First), fmtFloat(s.Last)})
		}
		summaryRef, err := pushRef(store, dataServerBaseURL, "TrendReportSummary: "+fqn, "summary",
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
			summaryRows)
		if err != nil {
			logger.Warn("failed to push summary to data server", "error", err)
			summaryRef = resourceRef{}
		}

		valueRows := make([][]string, 0, len(values))
		for _, pv := range values {
			valueRows = append(valueRows, []string{pv.FQN, pv.DateTime, fmtFloat(pv.Value), fmtInt(pv.OpcQuality), fmtStr(pv.Text), pv.Unit})
		}
		valueRef, err := pushRef(store, dataServerBaseURL, "TrendReportValues: "+fqn, "values",
			[]dataserver.Field{
				{Name: "FQN", Type: "string"},
				{Name: "DateTime", Type: "datetime"},
				{Name: "Value", Type: "number"},
				{Name: "OpcQuality", Type: "number"},
				{Name: "Text", Type: "string"},
				{Name: "Unit", Type: "string"},
			},
			valueRows)
		if err != nil {
			logger.Warn("failed to push values to data server", "error", err)
			valueRef = resourceRef{}
		}

		logger.Info("prompt ok", "name", "trend_report", "fqn", fqn, "values", len(values))
		return resultMessage(b.String(), summaryRef, valueRef), nil
	}
}
