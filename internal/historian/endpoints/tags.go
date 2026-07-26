package endpoints

import (
    "context"
    "fmt"

    "github.com/chewcw/aveva-historian-mcpserver/internal/historian"
    "github.com/chewcw/aveva-historian-mcpserver/internal/types"
    odataqb "github.com/chewcw/odata-query-builder"
)

func GetTags(ctx context.Context, client *historian.Client, groups []types.FilterGroupDef, top, skip int) (*historian.ODataResponse[historian.Tag], error) {
    qb := odataqb.New().Top(top).Skip(skip)
    applyFilterGroups(qb, groups)
    q := qb.Build()
    var result historian.ODataResponse[historian.Tag]
    if err := client.Get(ctx, "Tags", q, &result); err != nil {
        return nil, fmt.Errorf("fetch tags: %w", err)
    }
    return &result, nil
}
