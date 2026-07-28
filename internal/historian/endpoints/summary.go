package endpoints

import (
	"context"
	"fmt"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
)

func GetAnalogSummary(ctx context.Context, client *historian.Client, groups []types.FilterGroupDef, opts types.QueryOptions) (*historian.ODataResponse[historian.AnalogSummaryValue], error) {
	q := buildQuery(groups, opts)

	var result historian.ODataResponse[historian.AnalogSummaryValue]
	if err := client.Get(ctx, "AnalogSummary", q, &result); err != nil {
		return nil, fmt.Errorf("fetch analog summary: %w", err)
	}
	return &result, nil
}
