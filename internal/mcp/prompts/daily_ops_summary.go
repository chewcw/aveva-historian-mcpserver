package prompts

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/chewcw/aveva-historian-mcpserver/internal/dataserver"
	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/historian/endpoints"
	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxDailyOpsFqns = 10

func RegisterDailyOpsSummary(server *mcp.Server, client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "daily_ops_summary",
		Title:       "Daily Operations Summary",
		Description: "Return a one-day digest of events and alarms, plus per-tag daily statistics for an optional list of tags.",
		Arguments: []*mcp.PromptArgument{
			{Name: "date", Description: "Day as YYYY-MM-DD (default today, UTC)"},
			{Name: "namespace", Description: "Namespace filter for events (optional)"},
			{Name: "severity", Description: "1=Critical, 2=Major, 3=Minor, 4=Informational (optional)"},
			{Name: "fqns", Description: "Comma-separated tag FQNs for daily stats (max 10, optional)"},
		},
	}, handleDailyOpsSummary(client, store, dataServerBaseURL, logger, byteLimit))
}

func handleDailyOpsSummary(client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) func(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := promptArgs(req)
		start, end, date, dateNote, err := resolveDayWindow(args, time.Now())
		if err != nil {
			return errResult(err.Error()), nil
		}
		severity, err := severityArg(args)
		if err != nil {
			return errResult(err.Error()), nil
		}

		// Events half.
		evConds := []types.FilterCondition{
			{Field: "EventTime", Operator: "ge", Value: start},
			{Field: "EventTime", Operator: "le", Value: end},
		}
		if severity != 0 {
			evConds = append(evConds, types.FilterCondition{Field: "Severity", Operator: "eq", Value: severity})
		}
		if ns := strings.TrimSpace(args["namespace"]); ns != "" {
			evConds = append(evConds, types.FilterCondition{Field: "Namespace", Operator: "eq", Value: ns})
		}
		evRes, err := endpoints.GetEvents(ctx, client, []types.FilterGroupDef{{And: evConds}}, types.QueryOptions{})
		if err != nil {
			return nil, fmt.Errorf("fetch daily events: %w", err)
		}
		events := evRes.Value
		if events == nil {
			events = []historian.Event{}
		}

		// Summary half — only when fqns given.
		var fqns []string
		if raw := strings.TrimSpace(args["fqns"]); raw != "" {
			fqns = splitCSV(raw)
			if len(fqns) > maxDailyOpsFqns {
				return errResult(fmt.Sprintf("Too many FQNs: %d (max %d)", len(fqns), maxDailyOpsFqns)), nil
			}
		}
		summaries := make([]historian.AnalogSummaryValue, 0, len(fqns))
		for _, fqn := range fqns {
			conds := []types.FilterCondition{
				{Field: "FQN", Operator: "eq", Value: fqn},
				{Field: "StartDateTime", Operator: "ge", Value: start},
				{Field: "EndDateTime", Operator: "le", Value: end},
			}
			res, err := endpoints.GetAnalogSummary(ctx, client, []types.FilterGroupDef{{And: conds}}, types.QueryOptions{})
			if err != nil {
				return nil, fmt.Errorf("fetch daily summary for %s: %w", fqn, err)
			}
			if len(res.Value) > 0 {
				summaries = append(summaries, res.Value[0])
			}
		}

		var b strings.Builder
		fmt.Fprintf(&b, "## Daily Ops Summary: %s %s\n\n", date, dateNote)

		b.WriteString("### Events\n")
		if len(events) == 0 {
			b.WriteString("No events in the given period\n")
		} else {
			typeCounts := map[string]int{}
			sevCounts := map[int]int{}
			for _, ev := range events {
				if ev.Type != "" {
					typeCounts[ev.Type]++
				}
				if ev.Severity != nil {
					sevCounts[*ev.Severity]++
				}
			}
			b.WriteString("By type:\n")
			sortedTypes := make([]string, 0, len(typeCounts))
			for typ := range typeCounts {
				sortedTypes = append(sortedTypes, typ)
			}
			sort.Strings(sortedTypes)
			for _, typ := range sortedTypes {
				fmt.Fprintf(&b, "- %s: %d\n", typ, typeCounts[typ])
			}
			b.WriteString("By severity:\n")
			for s := 1; s <= 4; s++ {
				fmt.Fprintf(&b, "- %d: %d\n", s, sevCounts[s])
			}
		}

		b.WriteString("\n### Daily stats\n")
		if len(summaries) == 0 {
			if len(fqns) == 0 {
				b.WriteString("No fqns provided — run tag_explorer to discover tags, then re-run with fqns\n")
			} else {
				b.WriteString("No summary data found for the given tags and day\n")
			}
		} else {
			b.WriteString("| FQN | Min | Max | Avg | StdDev | Integral | Count |\n")
			b.WriteString("|-----|-----|-----|-----|--------|----------|-------|\n")
			for _, s := range summaries {
				fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n",
					s.FQN, fmtFloat(s.Minimum), fmtFloat(s.Maximum), fmtFloat(s.Average),
					fmtFloat(s.StdDev), fmtFloat(s.Integral), fmtInt(s.Count))
			}
		}

		var evRef resourceRef
		if len(events) > 0 {
			evRows := make([][]string, 0, len(events))
			for _, ev := range events {
				evRows = append(evRows, []string{ev.ID, ev.EventTime, ev.Type,
					fmtInt(ev.Severity), ev.SourceName, ev.Namespace})
			}
			evRef, err = pushRef(store, dataServerBaseURL, "DailyOpsEvents: "+date, "events",
				[]dataserver.Field{
					{Name: "ID", Type: "string"},
					{Name: "EventTime", Type: "datetime"},
					{Name: "Type", Type: "string"},
					{Name: "Severity", Type: "number"},
					{Name: "SourceName", Type: "string"},
					{Name: "Namespace", Type: "string"},
				},
				evRows)
			if err != nil {
				logger.Warn("failed to push events to data server", "error", err)
				evRef = resourceRef{}
			}
		}

		var sumRef resourceRef
		if len(summaries) > 0 {
			sumRows := make([][]string, 0, len(summaries))
			for _, s := range summaries {
				sumRows = append(sumRows, []string{s.FQN, s.StartDateTime, s.EndDateTime,
					fmtFloat(s.Minimum), fmtFloat(s.Maximum), fmtFloat(s.Average),
					fmtFloat(s.StdDev), fmtFloat(s.Integral), fmtInt(s.Count)})
			}
			sumRef, err = pushRef(store, dataServerBaseURL, "DailyOpsSummary: "+date, "summary",
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
				},
				sumRows)
			if err != nil {
				logger.Warn("failed to push summary to data server", "error", err)
				sumRef = resourceRef{}
			}
		}

		logger.Info("prompt ok", "name", "daily_ops_summary", "date", date, "events", len(events), "tags", len(summaries))
		return resultMessage(b.String(), evRef, sumRef), nil
	}
}
