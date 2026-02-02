package engine

import "fmt"

// Row represents a single result row as a map of field name → value
// Values are typed: string, int64, float64, bool, nil, time.Time
type Row map[string]interface{}

// Get returns the value of a field
func (r Row) Get(field string) interface{} {
	return r[field]
}

// String returns the string value of a field, or empty string if not found/not string
func (r Row) String(field string) string {
	v, ok := r[field]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}

// Int returns the int64 value of a field, or 0 if not found/not int
func (r Row) Int(field string) int64 {
	v, ok := r[field]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int64:
		return n
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

// QueryResult holds the result of a query execution
type QueryResult struct {
	// Entity name this result belongs to
	Entity string
	// Rows returned by the main query
	Rows []Row
	// Eager-loaded relations: relation name → rows
	Relations map[string][]Row
}

// Count returns the number of rows in the main result
func (qr *QueryResult) Count() int {
	return len(qr.Rows)
}

// IsEmpty returns true if no rows were returned
func (qr *QueryResult) IsEmpty() bool {
	return len(qr.Rows) == 0
}
