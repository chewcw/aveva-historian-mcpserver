package endpoints

import (
	"context"
	"fmt"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
	odataqb "github.com/chewcw/odata-query-builder"
)

func GetProcessValues(ctx context.Context, client *historian.Client, groups []types.FilterGroupDef, opts types.QueryOptions) (*historian.ODataResponse[historian.ProcessValue], error) {
	qb := odataqb.New()
	if opts.Top != nil {
		qb.Top(*opts.Top)
	}
	applyFilterGroups(qb, groups)
	if len(opts.Select) > 0 {
		qb.Select(opts.Select...)
	}
	if opts.Skip != nil {
		qb.Skip(*opts.Skip)
	}
	for _, ob := range opts.OrderBy {
		if ob.Direction == types.OrderDesc {
			qb.OrderByDesc(ob.Field)
		} else {
			qb.OrderBy(ob.Field)
		}
	}
	if opts.Count != nil && *opts.Count {
		qb.Count()
	}
	if opts.Search != nil {
		qb.Search(*opts.Search)
	}
	q := qb.Build()

	var result historian.ODataResponse[historian.ProcessValue]
	if err := client.Get(ctx, "ProcessValues", q, &result); err != nil {
		return nil, fmt.Errorf("fetch process values: %w", err)
	}
	return &result, nil
}
