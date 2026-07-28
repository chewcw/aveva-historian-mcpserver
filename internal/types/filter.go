package types

// FilterCondition represents a single OData filter expression: field operator value.
type FilterCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"` // eq, ne, gt, ge, lt, le, startsWith, endsWith, contains, in, has
	Value    any    `json:"value"`    // string, number, bool, or []any for "in"
}

// FilterGroupDef is a named group of AND or OR-joined conditions.
// Exactly one of And or Or should be populated per group.
// Maps to odataqb.FilterGroup for parenthesized grouping.
type FilterGroupDef struct {
	And []FilterCondition `json:"and,omitempty"`
	Or  []FilterCondition `json:"or,omitempty"`
}
