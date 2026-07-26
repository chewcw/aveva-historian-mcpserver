package endpoints

import (
	"context"
	"fmt"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
	odataqb "github.com/chewcw/odata-query-builder"
)

func GetProcessValues(ctx context.Context, client *historian.Client, groups []types.FilterGroupDef, retrievalMode string, resolutionMS, top int) (*historian.ODataResponse[historian.ProcessValue], error) {
	qb := odataqb.New().Top(top)
	applyFilterGroups(qb, groups)
	q := qb.Build()

	// Append custom AVEVA params not supported by odataqb
	if retrievalMode != "" {
		if q != "" { q += "&" }
		q += "RetrievalMode=" + retrievalMode
	}
	if resolutionMS > 0 {
		if q != "" { q += "&" }
		q += fmt.Sprintf("Resolution=%d", resolutionMS)
	}

	var result historian.ODataResponse[historian.ProcessValue]
	if err := client.Get(ctx, "ProcessValues", q, &result); err != nil {
		return nil, fmt.Errorf("fetch process values: %w", err)
	}
	return &result, nil
}
