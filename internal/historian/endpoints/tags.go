package endpoints

import (
    "context"
    "fmt"

    "github.com/chewcw/aveva-historian-mcpserver/internal/historian"
    "github.com/chewcw/aveva-historian-mcpserver/internal/types"
    odataqb "github.com/chewcw/odata-query-builder"
)

// TagsParams holds optional query parameters for the Tags endpoint.
// Pointer fields: nil = omit the parameter from the API request.
type TagsParams struct {
	Source            *string
	EngUnit           *string
	EngUnitMax        *float64
	EngUnitMin        *float64
	InterpolationType *string
	IntegralDivisor   *float64
	RolloverValue     *float64
	MessageOff        *string
	MessageOn         *string
	TagName           *string
	TagType           *string
	Description       *string
}

func GetTags(ctx context.Context, client *historian.Client, groups []types.FilterGroupDef, top, skip int, extra TagsParams) (*historian.ODataResponse[historian.Tag], error) {
    qb := odataqb.New().Top(top).Skip(skip)
    applyFilterGroups(qb, groups)

	// Custom AVEVA params via v0.1.2 Param()
	// client.Get's urlEncodeQueryValues handles URL-encoding of all values.
	if extra.Source != nil {
		qb.Param("Source", *extra.Source)
	}
	if extra.EngUnit != nil {
		qb.Param("EngUnit", *extra.EngUnit)
	}
	if extra.EngUnitMax != nil {
		qb.Param("EngUnitMax", fmt.Sprintf("%g", *extra.EngUnitMax))
	}
	if extra.EngUnitMin != nil {
		qb.Param("EngUnitMin", fmt.Sprintf("%g", *extra.EngUnitMin))
	}
	if extra.InterpolationType != nil {
		qb.Param("InterpolationType", *extra.InterpolationType)
	}
	if extra.IntegralDivisor != nil {
		qb.Param("IntegralDivisor", fmt.Sprintf("%g", *extra.IntegralDivisor))
	}
	if extra.RolloverValue != nil {
		qb.Param("RolloverValue", fmt.Sprintf("%g", *extra.RolloverValue))
	}
	if extra.MessageOff != nil {
		qb.Param("MessageOff", *extra.MessageOff)
	}
	if extra.MessageOn != nil {
		qb.Param("MessageOn", *extra.MessageOn)
	}
	if extra.TagName != nil {
		qb.Param("TagName", *extra.TagName)
	}
	if extra.TagType != nil {
		qb.Param("TagType", *extra.TagType)
	}
	if extra.Description != nil {
		qb.Param("Description", *extra.Description)
	}

    q := qb.Build()
    var result historian.ODataResponse[historian.Tag]
    if err := client.Get(ctx, "Tags", q, &result); err != nil {
        return nil, fmt.Errorf("fetch tags: %w", err)
    }
    return &result, nil
}
