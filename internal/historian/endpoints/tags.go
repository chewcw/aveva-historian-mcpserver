package endpoints

import (
	"context"
	"fmt"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
)

func GetTags(ctx context.Context, client *historian.Client, groups []types.FilterGroupDef, opts types.QueryOptions) (*historian.ODataResponse[historian.Tag], error) {
	q := buildQuery(groups, opts)
	var result historian.ODataResponse[historian.Tag]
	if err := client.Get(ctx, "Tags", q, &result); err != nil {
		return nil, fmt.Errorf("fetch tags: %w", err)
	}
	return &result, nil
}
