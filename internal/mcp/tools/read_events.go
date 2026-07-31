package tools

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

func RegisterReadEvents(server *mcp.Server, client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) {
	logger = logger.With("tool", "read_events")
	inputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"event_id":           map[string]any{"type": "string", "description": "Event GUID"},
			"event_time_gt":      map[string]any{"type": "string", "description": "EventTime start (UTC)"},
			"event_time_lt":      map[string]any{"type": "string", "description": "EventTime end (UTC)"},
			"received_time_gt":   map[string]any{"type": "string", "description": "ReceivedTime start (UTC)"},
			"received_time_lt":   map[string]any{"type": "string", "description": "ReceivedTime end (UTC)"},
			"type":               map[string]any{"type": "string", "description": "Event type, e.g. Alarm.Set, Alarm.Clear, User.Write"},
			"is_alarm":           map[string]any{"type": "boolean", "description": "Filter by alarm status"},
			"severity":           map[string]any{"type": "number", "description": "Severity (1=Critical, 2=Major, 3=Minor, 4=Informational)"},
			"priority":           map[string]any{"type": "number", "description": "Priority (1-999, lower = higher)"},
			"namespace":          map[string]any{"type": "string", "description": "Namespace for the event tag"},
			"in_touch_type":      map[string]any{"type": "string", "description": "InTouch type (ALM, RTN, ACK, SYS)"},
			"alarm_id":           map[string]any{"type": "string", "description": "Alarm GUID"},
			"alarm_class":        map[string]any{"type": "string", "description": "Alarm classification (DSC, VALUE, DEV, ROC)"},
			"alarm_condition":    map[string]any{"type": "string", "description": "Alarm condition (Limit.Hi, ROC.Lo, System)"},
			"alarm_type":         map[string]any{"type": "string", "description": "Alarm type (LoLo, Hi, etc.)"},
			"alarm_acknowledged": map[string]any{"type": "boolean", "description": "Alarm acknowledged state"},
			"top":                map[string]any{"type": "number", "description": "OData $top — max results (optional, default 100)"},
			"select":             map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "OData $select — field names (optional)"},
			"skip":               map[string]any{"type": "number", "description": "OData $skip (optional)"},
			"orderby": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{
				"field":     map[string]any{"type": "string"},
				"direction": map[string]any{"type": "string", "enum": []string{"asc", "desc"}},
			}}, "description": "OData $orderby — array of {field, direction}"},
			"count":  map[string]any{"type": "boolean", "description": "OData $count (optional)"},
			"search": map[string]any{"type": "string", "description": "OData $search (optional)"},
			"filters": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]any{
						"and": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type":                 "object",
								"additionalProperties": false,
								"properties": map[string]any{
									"field":    map[string]any{"type": "string"},
									"operator": map[string]any{"type": "string"},
									"value":    map[string]any{"oneOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "number"}, map[string]any{"type": "boolean"}, map[string]any{"type": "array", "items": map[string]any{"type": "string"}}}},
								},
							},
						},
						"or": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type":                 "object",
								"additionalProperties": false,
								"properties": map[string]any{
									"field":    map[string]any{"type": "string"},
									"operator": map[string]any{"type": "string"},
									"value":    map[string]any{"oneOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "number"}, map[string]any{"type": "boolean"}, map[string]any{"type": "array", "items": map[string]any{"type": "string"}}}},
								},
							},
						},
					},
				},
			},
		},
	}

	server.AddTool(&mcp.Tool{
		Name: "read_events",
		Description: "Retrieve events and alarms from the AVEVA Historian Events endpoint. " +
			"Filter by event_id, event_time range, type, severity, priority, namespace, " +
			"alarm fields (alarm_id, alarm_class, alarm_condition, alarm_type, alarm_acknowledged), " +
			"and standard OData query options. " +
			"Note: do not URL-encode param values — client handles encoding automatically.",
		InputSchema: inputSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArgs(req)
		var convConds []types.FilterCondition

		if id := getOptionalStringArg(args, "event_id"); id != nil {
			convConds = append(convConds, types.FilterCondition{Field: "ID", Operator: "eq", Value: *id})
		}
		if t := getOptionalStringArg(args, "event_time_gt"); t != nil {
			convConds = append(convConds, types.FilterCondition{Field: "EventTime", Operator: "ge", Value: *t})
		}
		if t := getOptionalStringArg(args, "event_time_lt"); t != nil {
			convConds = append(convConds, types.FilterCondition{Field: "EventTime", Operator: "le", Value: *t})
		}
		if t := getOptionalStringArg(args, "received_time_gt"); t != nil {
			convConds = append(convConds, types.FilterCondition{Field: "ReceivedTime", Operator: "ge", Value: *t})
		}
		if t := getOptionalStringArg(args, "received_time_lt"); t != nil {
			convConds = append(convConds, types.FilterCondition{Field: "ReceivedTime", Operator: "le", Value: *t})
		}
		if typ := getOptionalStringArg(args, "type"); typ != nil {
			convConds = append(convConds, types.FilterCondition{Field: "Type", Operator: "eq", Value: *typ})
		}
		if b := getOptionalBoolArg(args, "is_alarm"); b != nil {
			convConds = append(convConds, types.FilterCondition{Field: "IsAlarm", Operator: "eq", Value: *b})
		}
		if s := getOptionalIntArg(args, "severity"); s != nil {
			convConds = append(convConds, types.FilterCondition{Field: "Severity", Operator: "eq", Value: *s})
		}
		if p := getOptionalIntArg(args, "priority"); p != nil {
			convConds = append(convConds, types.FilterCondition{Field: "Priority", Operator: "eq", Value: *p})
		}
		if ns := getOptionalStringArg(args, "namespace"); ns != nil {
			convConds = append(convConds, types.FilterCondition{Field: "Namespace", Operator: "eq", Value: *ns})
		}
		if itt := getOptionalStringArg(args, "in_touch_type"); itt != nil {
			convConds = append(convConds, types.FilterCondition{Field: "InTouchType", Operator: "eq", Value: *itt})
		}
		if aid := getOptionalStringArg(args, "alarm_id"); aid != nil {
			convConds = append(convConds, types.FilterCondition{Field: "Alarm_ID", Operator: "eq", Value: *aid})
		}
		if ac := getOptionalStringArg(args, "alarm_class"); ac != nil {
			convConds = append(convConds, types.FilterCondition{Field: "Alarm_Class", Operator: "eq", Value: *ac})
		}
		if ac := getOptionalStringArg(args, "alarm_condition"); ac != nil {
			convConds = append(convConds, types.FilterCondition{Field: "Alarm_Condition", Operator: "eq", Value: *ac})
		}
		if at := getOptionalStringArg(args, "alarm_type"); at != nil {
			convConds = append(convConds, types.FilterCondition{Field: "Alarm_Type", Operator: "eq", Value: *at})
		}
		if aa := getOptionalBoolArg(args, "alarm_acknowledged"); aa != nil {
			convConds = append(convConds, types.FilterCondition{Field: "Alarm_Acknowledged", Operator: "eq", Value: *aa})
		}

		var groups []types.FilterGroupDef
		if len(convConds) > 0 {
			groups = append(groups, types.FilterGroupDef{And: convConds})
		}
		userFilters := parseFilters(args["filters"])
		groups = append(groups, userFilters...)

		topVal := getIntArg(args, "top", 100)
		opts := types.QueryOptions{
			Top:     &topVal,
			Select:  getStringSliceArg(args, "select"),
			Skip:    getOptionalIntArg(args, "skip"),
			OrderBy: parseOrderByClauses(args, "orderby"),
			Count:   getOptionalBoolArg(args, "count"),
			Search:  getOptionalStringArg(args, "search"),
		}

		result, err := endpoints.GetEvents(ctx, client, groups, opts)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to fetch events: %v", err)}},
			}, nil
		}

		rows := result.Value
		if rows == nil {
			rows = []historian.Event{}
		}

		allRows := make([][]string, 0, len(rows))
		for _, ev := range rows {
			allRows = append(allRows, []string{
				ev.ID, ev.EventTime, ev.Type, intPtrStr(ev.Severity),
				ev.SourceName, ev.AlarmCondition, ev.Namespace,
				boolPtrStr(ev.IsAlarm), boolPtrStr(ev.AlarmAcknowledged),
				ev.Comment, ev.ValueString,
			})
		}

		resourceURI, err := PushResult(store, dataServerBaseURL,
			"Events",
			[]dataserver.Field{
				{Name: "ID", Type: "string"},
				{Name: "EventTime", Type: "datetime"},
				{Name: "Type", Type: "string"},
				{Name: "Severity", Type: "number"},
				{Name: "SourceName", Type: "string"},
				{Name: "AlarmCondition", Type: "string"},
				{Name: "Namespace", Type: "string"},
				{Name: "IsAlarm", Type: "boolean"},
				{Name: "AlarmAcknowledged", Type: "boolean"},
				{Name: "Comment", Type: "string"},
				{Name: "ValueString", Type: "string"},
			},
			allRows)
		if err != nil {
			logger.Warn("failed to push result to data server", "error", err)
			resourceURI = ""
		}

		logger.Info("ok", "rows", len(rows))

		return formatResult(rows, byteLimit, func(rows []historian.Event) string {
			var preview strings.Builder
			preview.WriteString("| ID | EventTime | Type | Severity | SourceName | AlarmCondition |\n")
			preview.WriteString("|----|----------|------|----------|------------|----------------|\n")
			for _, ev := range rows {
				fmt.Fprintf(&preview, "| %s | %s | %s | %s | %s | %s |\n",
					ev.ID, ev.EventTime, ev.Type, intPtrStr(ev.Severity),
					ev.SourceName, ev.AlarmCondition)
			}
			return preview.String()
		}, resourceURI, "events", result.Count), nil
	})
}

// boolPtrStr formats a *bool as "true"/"false"/"" for nil.
func boolPtrStr(b *bool) string {
	if b == nil {
		return ""
	}
	if *b {
		return "true"
	}
	return "false"
}
