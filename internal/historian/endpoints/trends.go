package endpoints

import (
	"context"
	"fmt"
	"net/url"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
)

func GetProcessValues(ctx context.Context, client *historian.Client, fqn, startTime, endTime, retrievalMode string, resolutionMS, top int) (*historian.ODataResponse[historian.ProcessValue], error) {
	filter := fmt.Sprintf("FQN eq '%s' and DateTime ge %s and DateTime le %s", fqn, startTime, endTime)
	q := fmt.Sprintf("$filter=%s", url.QueryEscape(filter))
	if retrievalMode != "" {
		q += fmt.Sprintf("&RetrievalMode=%s", url.QueryEscape(retrievalMode))
	}
	if resolutionMS > 0 {
		q += fmt.Sprintf("&Resolution=%d", resolutionMS)
	}
	q += fmt.Sprintf("&$top=%d", top)
	var result historian.ODataResponse[historian.ProcessValue]
	if err := client.Get(ctx, "ProcessValues", q, &result); err != nil {
		return nil, fmt.Errorf("trends query: %w", err)
	}
	return &result, nil
}
