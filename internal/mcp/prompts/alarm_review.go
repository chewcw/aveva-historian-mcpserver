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

func RegisterAlarmReview(server *mcp.Server, client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "alarm_review",
		Title:       "Alarm Review",
		Description: "Return alarm activity for a time window: counts by severity and type, plus an unacknowledged alarm list. Window defaults to the last 24 hours.",
		Arguments: []*mcp.PromptArgument{
			{Name: "start_date_time", Description: "Window start (default last 24h)"},
			{Name: "end_date_time", Description: "Window end (default now)"},
			{Name: "severity", Description: "1=Critical, 2=Major, 3=Minor, 4=Informational (optional)"},
			{Name: "namespace", Description: "Namespace filter (optional)"},
			{Name: "acknowledged", Description: "\"true\" or \"false\" (optional)"},
		},
	}, handleAlarmReview(client, store, dataServerBaseURL, logger, byteLimit))
}

func handleAlarmReview(client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) func(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := promptArgs(req)
		start, end, windowNote := resolveAlarmWindow(args, time.Now())
		severity, err := severityArg(args)
		if err != nil {
			return errResult(err.Error()), nil
		}

		conds := []types.FilterCondition{
			{Field: "IsAlarm", Operator: "eq", Value: true},
			{Field: "EventTime", Operator: "ge", Value: start},
			{Field: "EventTime", Operator: "le", Value: end},
		}
		if severity != 0 {
			conds = append(conds, types.FilterCondition{Field: "Severity", Operator: "eq", Value: severity})
		}
		if ns := strings.TrimSpace(args["namespace"]); ns != "" {
			conds = append(conds, types.FilterCondition{Field: "Namespace", Operator: "eq", Value: ns})
		}
		if ack := boolArg(args, "acknowledged", nil); ack != nil {
			conds = append(conds, types.FilterCondition{Field: "Alarm_Acknowledged", Operator: "eq", Value: *ack})
		}
		result, err := endpoints.GetEvents(ctx, client, []types.FilterGroupDef{{And: conds}}, types.QueryOptions{})
		if err != nil {
			return nil, fmt.Errorf("fetch alarms: %w", err)
		}
		rows := result.Value
		if rows == nil {
			rows = []historian.Event{}
		}
		if len(rows) == 0 {
			return errResult("No alarms in the given window"), nil
		}

		sevCounts := map[int]int{}
		typeCounts := map[string]int{}
		var unacked []historian.Event
		for _, ev := range rows {
			if ev.Severity != nil {
				sevCounts[*ev.Severity]++
			}
			if ev.Type != "" {
				typeCounts[ev.Type]++
			}
			if ev.AlarmAcknowledged != nil && !*ev.AlarmAcknowledged {
				unacked = append(unacked, ev)
			}
		}

		var b strings.Builder
		fmt.Fprintf(&b, "## Alarm Review\n\n")
		if windowNote != "" {
			fmt.Fprintf(&b, "Window: %s .. %s %s\n\n", start, end, windowNote)
		} else {
			fmt.Fprintf(&b, "Window: %s .. %s\n\n", start, end)
		}
		b.WriteString("### By severity\n| Severity | Count |\n|----------|-------|\n")
		sevNames := map[int]string{1: "1 Critical", 2: "2 Major", 3: "3 Minor", 4: "4 Informational"}
		for s := 1; s <= 4; s++ {
			fmt.Fprintf(&b, "| %s | %d |\n", sevNames[s], sevCounts[s])
		}
		b.WriteString("\n### By type\n| Type | Count |\n|------|-------|\n")
		sortedTypes := make([]string, 0, len(typeCounts))
		for typ := range typeCounts {
			sortedTypes = append(sortedTypes, typ)
		}
		sort.Strings(sortedTypes)
		for _, typ := range sortedTypes {
			fmt.Fprintf(&b, "| %s | %d |\n", typ, typeCounts[typ])
		}
		b.WriteString("\n### Unacknowledged\n| Severity | EventTime | Source | Condition |\n|----------|-----------|--------|-----------|\n")
		for _, ev := range capRows(unacked, 100) {
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
				fmtInt(ev.Severity), ev.EventTime, ev.SourceName, ev.AlarmCondition)
		}

		allRows := make([][]string, 0, len(rows))
		for _, ev := range rows {
			allRows = append(allRows, []string{ev.ID, ev.EventTime, ev.Type,
				fmtInt(ev.Severity), ev.SourceName, ev.AlarmCondition,
				ev.Namespace, boolStr(ev.AlarmAcknowledged)})
		}
		ref, err := pushRef(store, dataServerBaseURL, "AlarmReview", "alarms",
			[]dataserver.Field{
				{Name: "ID", Type: "string"},
				{Name: "EventTime", Type: "datetime"},
				{Name: "Type", Type: "string"},
				{Name: "Severity", Type: "number"},
				{Name: "SourceName", Type: "string"},
				{Name: "AlarmCondition", Type: "string"},
				{Name: "Namespace", Type: "string"},
				{Name: "AlarmAcknowledged", Type: "boolean"},
			},
			allRows)
		if err != nil {
			logger.Warn("failed to push result to data server", "error", err)
			ref = resourceRef{}
		}

		logger.Info("prompt ok", "name", "alarm_review", "alarms", len(rows), "unacked", len(unacked))
		return resultMessage(b.String(), ref), nil
	}
}
