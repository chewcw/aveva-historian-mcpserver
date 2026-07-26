package endpoints

import (
	"context"
	"fmt"
	"net/url"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
)

func GetTags(ctx context.Context, client *historian.Client, filter string, top, skip int) (*historian.ODataResponse[historian.Tag], error) {
	q := fmt.Sprintf("$filter=%s&$top=%d&$skip=%d", url.QueryEscape(filter), top, skip)
	var result historian.ODataResponse[historian.Tag]
	if err := client.Get(ctx, "Tags", q, &result); err != nil {
		return nil, fmt.Errorf("tags query: %w", err)
	}
	return &result, nil
}
