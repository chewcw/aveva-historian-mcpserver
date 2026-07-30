package types

// OrderDirection is a statically typed enum for sort direction.
type OrderDirection string

const (
	OrderAsc  OrderDirection = "asc"
	OrderDesc OrderDirection = "desc"
)

// String returns the string representation for use in OData query building.
func (d OrderDirection) String() string { return string(d) }

// OrderByClause represents a single $orderby expression: a field and direction.
type OrderByClause struct {
	Field     string         // field name, e.g. "DateTime"
	Direction OrderDirection // OrderAsc or OrderDesc
}

// QueryOptions holds standard OData system query options.
// Pointer fields: nil = omit the parameter from the request.
// Slice fields: nil or empty = omit.
type QueryOptions struct {
	Select  []string        // $select — field names, e.g. []string{"FQN","DateTime","Value"}
	Top     *int            // $top — nil = omit
	Skip    *int            // $skip — nil = omit
	OrderBy []OrderByClause // $orderby — nil or empty = omit
	Count   *bool           // $count — nil = omit
	Search  *string         // $search — nil = omit
}
