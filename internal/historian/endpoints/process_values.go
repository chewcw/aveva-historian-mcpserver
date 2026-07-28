package endpoints

import (
	"context"
	"fmt"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
)

func GetProcessValues(ctx context.Context, client *historian.Client, groups []types.FilterGroupDef, opts types.QueryOptions) (*historian.ODataResponse[historian.ProcessValue], error) {
	q := buildQuery(groups, opts)

	var result historian.ODataResponse[historian.ProcessValue]
	if err := client.Get(ctx, "ProcessValues", q, &result); err != nil {
		return nil, fmt.Errorf("fetch process values: %w", err)
	}
	return &result, nil
}
