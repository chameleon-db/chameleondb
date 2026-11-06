package mutation

import (
	"fmt"
	"sort"
	"strings"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

// likeEscaper escapes the characters that LIKE treats as wildcards, so a
// filter value is always matched as plain text.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// escapeLikePattern escapes backslash, % and _ for use with ESCAPE '\'.
func escapeLikePattern(s string) string {
	return likeEscaper.Replace(s)
}

// buildWhere turns mutation filters into SQL conditions with bound parameters.
//
// Filters are keyed by "field" (equality) or "field:op". Keys are processed in
// sorted order, so the same filters always produce the same SQL.
//
// Rules:
//   - eq / neq with a nil value become IS NULL / IS NOT NULL (no parameter).
//   - any other operator with a nil value is an error.
//   - like / ilike match the value as text anywhere in the field ("contains"),
//     with %, _ and \ escaped. This is the same behavior as the query builder.
//
// Parameters are numbered from firstParam. It returns the conditions and the
// values for the numbered parameters, in order.
func buildWhere(filters map[string]interface{}, firstParam int) ([]string, []interface{}, error) {
	keys := make([]string, 0, len(filters))
	for key := range filters {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var clauses []string
	var values []interface{}
	param := firstParam

	for _, key := range keys {
		parts := strings.SplitN(key, ":", 2)
		field := parts[0]
		op := "eq"
		if len(parts) == 2 && parts[1] != "" {
			op = strings.ToLower(parts[1])
		}

		sqlOp, err := mutationOperatorToSQL(op)
		if err != nil {
			return nil, nil, err
		}

		column, err := engine.QuoteIdent(field)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid field name %q", field)
		}

		value := filters[key]

		if value == nil {
			switch op {
			case "eq":
				clauses = append(clauses, column+" IS NULL")
			case "neq", "ne":
				clauses = append(clauses, column+" IS NOT NULL")
			default:
				return nil, nil, fmt.Errorf("operator %q cannot be used with a nil value on %q (only eq and neq match NULL)", op, field)
			}
			continue
		}

		switch op {
		case "like", "ilike":
			text, ok := value.(string)
			if !ok {
				return nil, nil, fmt.Errorf("operator %q needs a string value on %q, got %T", op, field, value)
			}
			clauses = append(clauses, fmt.Sprintf(`%s %s $%d ESCAPE '\'`, column, sqlOp, param))
			values = append(values, "%"+escapeLikePattern(text)+"%")
		default:
			clauses = append(clauses, fmt.Sprintf("%s %s $%d", column, sqlOp, param))
			values = append(values, value)
		}
		param++
	}

	return clauses, values, nil
}
