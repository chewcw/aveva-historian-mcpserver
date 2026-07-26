package endpoints

import (
	"strings"

	"github.com/chewcw/aveva-historian-mcpserver/internal/types"
	odataqb "github.com/chewcw/odata-query-builder"
)

// applyFilterGroups applies structured filter groups to a QueryBuilder.
func applyFilterGroups(qb *odataqb.QueryBuilder, groups []types.FilterGroupDef) {
	for _, g := range groups {
		switch {
		case len(g.Or) > 0 && len(g.And) == 0:
			qb.OrFilterGroup(func(fg *odataqb.FilterGroup) {
				addCondGroup(fg, g.Or, "or")
			})
		default:
			qb.FilterGroup(func(fg *odataqb.FilterGroup) {
				addCondGroup(fg, g.And, "and")
				addCondGroup(fg, g.Or, "or")
			})
		}
	}
}

func addCondGroup(fg *odataqb.FilterGroup, conds []types.FilterCondition, logic string) {
	for _, c := range conds {
		var gfb *odataqb.GroupFilterBuilder
		if logic == "or" {
			gfb = fg.OrFilter(c.Field)
		} else {
			gfb = fg.Filter(c.Field)
		}
		addCond(gfb, c)
	}
}

func addCond(gfb *odataqb.GroupFilterBuilder, c types.FilterCondition) {
	switch strings.ToLower(c.Operator) {
	case "eq":
		gfb.Eq(c.Value)
	case "ne":
		gfb.Ne(c.Value)
	case "gt":
		gfb.Gt(c.Value)
	case "ge":
		gfb.Ge(c.Value)
	case "lt":
		gfb.Lt(c.Value)
	case "le":
		gfb.Le(c.Value)
	case "startswith":
		if s, ok := c.Value.(string); ok {
			gfb.StartsWith(s)
		}
	case "endswith":
		if s, ok := c.Value.(string); ok {
			gfb.EndsWith(s)
		}
	case "contains":
		if s, ok := c.Value.(string); ok {
			gfb.Contains(s)
		}
	case "in":
		if arr, ok := c.Value.([]any); ok {
			gfb.In(arr...)
		}
	case "has":
		if s, ok := c.Value.(string); ok {
			gfb.Has(s)
		}
	}
}
