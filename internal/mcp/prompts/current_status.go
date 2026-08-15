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

const maxCurrentStatusTags = 10

func RegisterCurrentStatus(server *mcp.Server, client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "current_status",
		Title:       "Current Tag Status",
		Description: "Return the latest process value, timestamp, OPC quality, and unit for one or more historian tags.",
		Arguments: []*mcp.PromptArgument{
			{Name: "fqn", Description: "Comma-separated tag FQNs (max 10)", Required: true},
		},
	}, handleCurrentStatus(client, store, dataServerBaseURL, logger, byteLimit))
}

func handleCurrentStatus(client *historian.Client, store *dataserver.Store, dataServerBaseURL string, logger *slog.Logger, byteLimit int) func(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := promptArgs(req)
		fqnRaw, err := requiredArg(args, "fqn")
		if err != nil {
			return errResult(err.Error()), nil
		}
		fqns := splitCSV(fqnRaw)
		if len(fqns) == 0 {
			return errResult("Missing required argument: fqn"), nil
		}
		if len(fqns) > maxCurrentStatusTags {
			return errResult(fmt.Sprintf("Too many FQNs: %d (max %d)", len(fqns), maxCurrentStatusTags)), nil
		}

		top := 1
		opts := types.QueryOptions{
			Top:     &top,
			OrderBy: []types.OrderByClause{{Field: "DateTime", Direction: types.OrderDesc}},
		}
		latest := make([]historian.ProcessValue, 0, len(fqns))
		for _, fqn := range fqns {
			groups := []types.FilterGroupDef{{And: []types.FilterCondition{{Field: "FQN", Operator: "eq", Value: fqn}}}}
			result, err := endpoints.GetProcessValues(ctx, client, groups, opts)
			if err != nil {
				return nil, fmt.Errorf("fetch current status for %s: %w", fqn, err)
			}
			if len(result.Value) > 0 {
				latest = append(latest, result.Value[0])
			}
		}
		if len(latest) == 0 {
			return errResult("No values found for the given tags"), nil
		}

		var b strings.Builder
		b.WriteString("| FQN | DateTime | Value | OpcQuality | Unit |\n")
		b.WriteString("|-----|----------|-------|------------|------|\n")
		for _, pv := range latest {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				pv.FQN, pv.DateTime, fmtFloat(pv.Value), fmtInt(pv.OpcQuality), pv.Unit)
		}

		allRows := make([][]string, 0, len(latest))
		for _, pv := range latest {
			allRows = append(allRows, []string{pv.FQN, pv.DateTime, fmtFloat(pv.Value), fmtInt(pv.OpcQuality), fmtStr(pv.Text), pv.Unit})
		}
		ref, err := pushRef(store, dataServerBaseURL, "CurrentStatus", "status",
			[]dataserver.Field{
				{Name: "FQN", Type: "string"},
				{Name: "DateTime", Type: "datetime"},
				{Name: "Value", Type: "number"},
				{Name: "OpcQuality", Type: "number"},
				{Name: "Text", Type: "string"},
				{Name: "Unit", Type: "string"},
			},
			allRows)
		if err != nil {
			logger.Warn("failed to push result to data server", "error", err)
			ref = resourceRef{}
		}

		logger.Info("prompt ok", "name", "current_status", "tags", len(fqns))
		return resultMessage(b.String(), ref), nil
	}
}
