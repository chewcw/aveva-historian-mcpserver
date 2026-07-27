package endpoints

import (
	"context"
	"fmt"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
	odataqb "github.com/chewcw/odata-query-builder"
)

// AnalogSummaryParams holds optional query parameters for the AnalogSummary endpoint.
// Pointer fields: nil = omit the parameter from the API request.
type AnalogSummaryParams struct {
	RetrievalMode *string  // "Cyclic" or "Full"; nil = omit (API default: Cyclic)
	SliceBy       *string  // comma-separated FQNs; nil = omit
	SliceByValue  *string  // nil = omit
	OPCQuality    *int     // nil = omit
	PercentGood   *float64 // nil = omit
}

func GetAnalogSummary(ctx context.Context, client *historian.Client, groups []types.FilterGroupDef, resolutionMS, top int, extra AnalogSummaryParams) (*historian.ODataResponse[historian.AnalogSummaryValue], error) {
	qb := odataqb.New().Top(top)
	applyFilterGroups(qb, groups)

	// Custom AVEVA params via v0.1.2 Param()
	// client.Get's urlEncodeQueryValues handles URL-encoding of all values.
	if resolutionMS > 0 {
		qb.Param("Resolution", fmt.Sprintf("%d", resolutionMS))
	}
	if extra.RetrievalMode != nil {
		qb.Param("RetrievalMode", *extra.RetrievalMode)
	}
	if extra.SliceBy != nil {
		qb.Param("SliceBy", *extra.SliceBy)
	}
	if extra.SliceByValue != nil {
		qb.Param("SliceByValue", *extra.SliceByValue)
	}
	if extra.OPCQuality != nil {
		qb.Param("OPCQuality", fmt.Sprintf("%d", *extra.OPCQuality))
	}
	if extra.PercentGood != nil {
		qb.Param("PercentGood", fmt.Sprintf("%g", *extra.PercentGood))
	}

	q := qb.Build()

	var result historian.ODataResponse[historian.AnalogSummaryValue]
	if err := client.Get(ctx, "AnalogSummary", q, &result); err != nil {
		return nil, fmt.Errorf("fetch analog summary: %w", err)
	}
	return &result, nil
}
