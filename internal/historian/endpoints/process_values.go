package endpoints

import (
	"context"
	"fmt"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
	odataqb "github.com/chewcw/odata-query-builder"
)

// ProcessValuesParams holds optional query parameters for the ProcessValues endpoint.
// Pointer fields: nil = omit the parameter from the API request.
type ProcessValuesParams struct {
	RetrievalMode *string // "Average"|"Cyclic"|"Integral"|"Minimum"|"Maximum"|"BestFit"|"Delta"|"Interpolated"|"Slope"|"Counter"|"Full"
	ResolutionMS  *int    // nil = omit (API default: 0 = raw values)
}

func GetProcessValues(ctx context.Context, client *historian.Client, groups []types.FilterGroupDef, top int, extra ProcessValuesParams) (*historian.ODataResponse[historian.ProcessValue], error) {
	qb := odataqb.New().Top(top)
	applyFilterGroups(qb, groups)

	// Custom AVEVA params via v0.1.2 Param()
	// client.Get's urlEncodeQueryValues handles URL-encoding of all values.
	if extra.RetrievalMode != nil {
		qb.Param("RetrievalMode", *extra.RetrievalMode)
	}
	if extra.ResolutionMS != nil {
		qb.Param("Resolution", fmt.Sprintf("%d", *extra.ResolutionMS))
	}

	q := qb.Build()

	var result historian.ODataResponse[historian.ProcessValue]
	if err := client.Get(ctx, "ProcessValues", q, &result); err != nil {
		return nil, fmt.Errorf("fetch process values: %w", err)
	}
	return &result, nil
}
