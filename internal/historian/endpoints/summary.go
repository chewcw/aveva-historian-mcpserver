package endpoints

import (
	"context"
	"fmt"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
	odataqb "github.com/chewcw/odata-query-builder"
)

func GetAnalogSummary(ctx context.Context, client *historian.Client, groups []types.FilterGroupDef, resolutionMS, top int) (*historian.ODataResponse[historian.AnalogSummaryValue], error) {
	qb := odataqb.New().Top(top)
	applyFilterGroups(qb, groups)
	q := qb.Build()

	// Append custom AVEVA params not supported by odataqb
	if resolutionMS > 0 {
		if q != "" { q += "&" }
		q += fmt.Sprintf("Resolution=%d", resolutionMS)
	}

	var result historian.ODataResponse[historian.AnalogSummaryValue]
	if err := client.Get(ctx, "AnalogSummary", q, &result); err != nil {
		return nil, fmt.Errorf("fetch analog summary: %w", err)
	}
	return &result, nil
}
