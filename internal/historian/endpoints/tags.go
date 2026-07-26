package endpoints

import (
    "context"
    "fmt"

    "github.com/chewcw/aveva-historian-mcpserver/internal/historian"

    odataqb "github.com/chewcw/odata-query-builder"
)

func GetTags(ctx context.Context, client *historian.Client, filter string, top, skip int) (*historian.ODataResponse[historian.Tag], error) {
    qb := odataqb.New().Top(top).Skip(skip)
    q := qb.Build()
    if filter != "" {
        filterParam := "$filter=" + filter
        if q != "" {
            q = filterParam + "&" + q
        } else {
            q = filterParam
        }
    }
    var result historian.ODataResponse[historian.Tag]
    if err := client.Get(ctx, "Tags", q, &result); err != nil {
        return nil, fmt.Errorf("fetch tags: %w", err)
    }
    return &result, nil
}
