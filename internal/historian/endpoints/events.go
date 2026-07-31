package endpoints

import (
	"context"
	"fmt"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
)

func GetEvents(ctx context.Context, client *historian.Client, groups []types.FilterGroupDef, opts types.QueryOptions) (*historian.ODataResponse[historian.Event], error) {
	q := buildQuery(groups, opts)

	var result historian.ODataResponse[historian.Event]
	if err := client.Get(ctx, "Events", q, &result); err != nil {
		return nil, fmt.Errorf("fetch events: %w", err)
	}
	return &result, nil
}
