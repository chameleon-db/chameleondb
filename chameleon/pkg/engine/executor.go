package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Executor runs queries against PostgreSQL
type Executor struct {
	connector *Connector
}

// NewExecutor creates an executor from a connector
func NewExecutor(connector *Connector) *Executor {
	return &Executor{connector: connector}
}

// Execute runs a generated query against the database.
func (ex *Executor) Execute(ctx context.Context, qb *QueryBuilder, generated *GeneratedSQL) (*QueryResult, error) {
	if !ex.connector.IsConnected() {
		return nil, fmt.Errorf("not connected to database")
	}

	debugCtx := qb.getDebugContext()

	// Create identity map for this query.
	identityMap := NewIdentityMap()

	// Execute main query
	mainArgs, err := bindArgs(generated.MainParams, nil)
	if err != nil {
		return nil, fmt.Errorf("main query failed: %w", err)
	}
	mainRows, err := ex.executeQuery(ctx, generated.MainQuery, mainArgs...)
	if err != nil {
		return nil, fmt.Errorf("main query failed: %w", err)
	}

	// Deduplicate main rows.
	mainRows = identityMap.Deduplicate(qb.query.Entity, mainRows)

	// Execute eager queries
	relations := make(map[string][]Row)
	relationIDs := map[string][]interface{}{
		"": extractIDs(mainRows, "id"),
	}

	for i, eager := range generated.EagerQueries {
		if len(eager) < 2 {
			return nil, fmt.Errorf("invalid eager query format")
		}

		relName := eager[0]
		relSQL := eager[1]

		parentIDs := relationIDs[""]
		if parentPath, ok := relationParentPath(relName); ok {
			if ids, found := relationIDs[parentPath]; found {
				parentIDs = ids
			}
		}

		if i >= len(generated.EagerParams) {
			return nil, fmt.Errorf("eager query '%s' failed: missing parameters", relName)
		}

		// Parent ids are bound as an array parameter.
		args, err := bindArgs(generated.EagerParams[i], parentIDs)
		if err != nil {
			return nil, fmt.Errorf("eager query '%s' failed: %w", relName, err)
		}

		debugCtx.LogEagerSQL(relName, relSQL, generated.EagerParams[i])

		eagerStart := time.Now()
		eagerRows, err := ex.executeQuery(ctx, relSQL, args...)
		if err == nil {
			debugCtx.LogEagerTrace(relName, time.Since(eagerStart), len(eagerRows))
		}
		if err != nil {
			return nil, fmt.Errorf("eager query '%s' failed: %w", relName, err)
		}

		// Deduplicate eager rows.
		entityName := inferEntityNameFromRelation(relName)
		eagerRows = identityMap.Deduplicate(entityName, eagerRows)

		relations[relName] = eagerRows
		if leaf := relationLeafName(relName); leaf != relName {
			if _, exists := relations[leaf]; !exists {
				relations[leaf] = eagerRows
			}
		}
		relationIDs[relName] = extractIDs(eagerRows, "id")
	}

	return &QueryResult{
		Entity:    qb.query.Entity,
		Rows:      mainRows,
		Relations: relations,
	}, nil
}

// inferEntityNameFromRelation infers entity name from relation name.
// Example: "posts" -> "Post", "orderItems" -> "OrderItem".
func inferEntityNameFromRelation(relName string) string {
	if relName == "" {
		return relName
	}

	segment := relName
	if idx := strings.LastIndex(relName, "."); idx >= 0 && idx+1 < len(relName) {
		segment = relName[idx+1:]
	}

	// Remove trailing 's' if present.
	singular := segment
	if strings.HasSuffix(segment, "s") && len(segment) > 1 {
		singular = segment[:len(segment)-1]
	}

	// Capitalize first letter.
	if len(singular) > 0 {
		return strings.ToUpper(singular[:1]) + singular[1:]
	}

	return singular
}

func relationParentPath(relName string) (string, bool) {
	idx := strings.LastIndex(relName, ".")
	if idx <= 0 {
		return "", false
	}
	return relName[:idx], true
}

func relationLeafName(relName string) string {
	idx := strings.LastIndex(relName, ".")
	if idx == -1 {
		return relName
	}
	return relName[idx+1:]
}

// bindArgs converts generated parameters into pgx arguments. parentIDs
// supplies the value of a ParentIds parameter.
func bindArgs(params []SQLParam, parentIDs []interface{}) ([]interface{}, error) {
	args := make([]interface{}, len(params))
	for i, param := range params {
		switch param.Kind {
		case ParamParentIDs:
			if parentIDs == nil {
				return nil, fmt.Errorf("parameter $%d needs parent ids", i+1)
			}
			args[i] = parentIDs
		case ParamNull:
			args[i] = nil
		case ParamString, ParamInt, ParamFloat, ParamBool:
			args[i] = param.Value
		default:
			return nil, fmt.Errorf("unsupported parameter kind %q", param.Kind)
		}
	}
	return args, nil
}

// executeQuery runs a single SQL query with bind arguments and returns rows.
func (ex *Executor) executeQuery(ctx context.Context, sql string, args ...interface{}) ([]Row, error) {
	rows, err := ex.connector.Pool().Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRows(rows)
}

// scanRows converts pgx rows into Row.
func scanRows(rows pgx.Rows) ([]Row, error) {
	var result []Row
	columns := rows.FieldDescriptions()

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(Row)
		for i, col := range columns {
			row[col.Name] = values[i]
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// extractIDs pulls a field from all rows and converts UUID values to string.
func extractIDs(rows []Row, field string) []interface{} {
	ids := make([]interface{}, 0, len(rows))
	for _, row := range rows {
		if id, ok := row[field]; ok {
			switch v := id.(type) {
			case []byte:
				ids = append(ids, string(v))
			case [16]byte:
				ids = append(ids, uuidToString(v))
			case string:
				ids = append(ids, v)
			default:
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// uuidToString converts a [16]byte UUID to standard format.
func uuidToString(uuid [16]byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uuid[0:4],
		uuid[4:6],
		uuid[6:8],
		uuid[8:10],
		uuid[10:16])
}
