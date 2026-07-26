package endpoints

import (
	"context"
	"fmt"
	"net/url"

	"github.com/chewcw/aveva-historian-mcpserver/internal/historian"
)

func GetAnalogSummary(ctx context.Context, client *historian.Client, fqn, startTime, endTime string, resolutionMS, top int) (*historian.ODataResponse[historian.AnalogSummaryValue], error) {
	filter := fmt.Sprintf("FQN eq '%s' and StartDateTime ge %s and EndDateTime ge %s", fqn, startTime, endTime)
	q := fmt.Sprintf("$filter=%s", url.QueryEscape(filter))
	if resolutionMS > 0 {
		q += fmt.Sprintf("&Resolution=%d", resolutionMS)
	}
	q += fmt.Sprintf("&$top=%d", top)
	var result historian.ODataResponse[historian.AnalogSummaryValue]
	if err := client.Get(ctx, "AnalogSummary", q, &result); err != nil {
		return nil, fmt.Errorf("analog summary query: %w", err)
	}
	return &result, nil
}
