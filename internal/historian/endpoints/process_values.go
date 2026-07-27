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
	RetrievalMode *string  // "Average"|"Cyclic"|"Integral"|"Minimum"|"Maximum"|"BestFit"|"Delta"|"Interpolated"|"Slope"|"Counter"|"Full"
	ResolutionMS  *int     // nil = omit (API default: 0 = raw values)
	OPCQuality    *int     // OPC quality filter (Int32)
	Value         *float64 // Value filter: 0 or 1 for binary/discrete tags
	Bounding      *bool    // Include boundary data outside query range
	Text          *string  // Text value for string/discrete tags
	TagFilter     *string  // OData filter on tag attributes like FQN, description
	Expression    *string  // UOM conversion expression, e.g. UOM([FQN],[Unit])
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
	if extra.OPCQuality != nil {
		qb.Param("OPCQuality", fmt.Sprintf("%d", *extra.OPCQuality))
	}
	if extra.Value != nil {
		qb.Param("Value", fmt.Sprintf("%g", *extra.Value))
	}
	if extra.Bounding != nil {
		qb.Param("Bounding", fmt.Sprintf("%t", *extra.Bounding))
	}
	if extra.Text != nil {
		qb.Param("Text", *extra.Text)
	}
	if extra.TagFilter != nil {
		qb.Param("TagFilter", *extra.TagFilter)
	}
	if extra.Expression != nil {
		qb.Param("Expression", *extra.Expression)
	}

	q := qb.Build()

	var result historian.ODataResponse[historian.ProcessValue]
	if err := client.Get(ctx, "ProcessValues", q, &result); err != nil {
		return nil, fmt.Errorf("fetch process values: %w", err)
	}
	return &result, nil
}
