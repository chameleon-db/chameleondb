package introspect

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type postgresIntrospector struct {
	conn *pgx.Conn
}

func newPostgresIntrospector(ctx context.Context, connStr string) (Introspector, error) {
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	return &postgresIntrospector{conn: conn}, nil
}

func (pi *postgresIntrospector) Detect(ctx context.Context) (bool, error) {
	var version string
	err := pi.conn.QueryRow(ctx, "SELECT version()").Scan(&version)
	return err == nil, err
}

func (pi *postgresIntrospector) ListTables(ctx context.Context) ([]string, error) {
	rows, err := pi.conn.Query(ctx, `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}

	return tables, rows.Err()
}

func (pi *postgresIntrospector) InspectTable(ctx context.Context, tableName string) (*TableInfo, error) {
	// Query columns
	rows, err := pi.conn.Query(ctx, `
		SELECT 
			c.column_name,
			c.data_type,
			c.is_nullable,
			COALESCE(tc.constraint_type = 'PRIMARY KEY', false) as is_primary,
			COALESCE(tc.constraint_type = 'UNIQUE', false) as is_unique,
			c.column_default
		FROM information_schema.columns c
		LEFT JOIN information_schema.table_constraints tc
			ON c.table_name = tc.table_name
			AND c.column_name = ANY(
				SELECT column_name 
				FROM information_schema.key_column_usage 
				WHERE constraint_name = tc.constraint_name
			)
		WHERE c.table_name = $1
		ORDER BY c.ordinal_position
	`, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	table := &TableInfo{
		Name:    tableName,
		Columns: []ColumnInfo{},
	}

	for rows.Next() {
		var col ColumnInfo
		var nullable string
		var defaultVal *string
		var isPrimary, isUnique bool

		if err := rows.Scan(
			&col.Name,
			&col.Type,
			&nullable,
			&isPrimary,
			&isUnique,
			&defaultVal,
		); err != nil {
			return nil, err
		}

		col.Nullable = nullable == "YES"
		col.DefaultVal = defaultVal
		col.PrimaryKey = isPrimary
		col.Unique = isUnique

		// Query foreign keys
		fkRows, err := pi.conn.Query(ctx, `
			SELECT ccu.table_name, ccu.column_name, tc.constraint_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			JOIN information_schema.constraint_column_usage ccu
				ON ccu.constraint_name = tc.constraint_name
				AND ccu.table_schema = tc.table_schema
			WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_name = $1
			AND kcu.column_name = $2
		`, tableName, col.Name)
		if err == nil {
			defer fkRows.Close()
			for fkRows.Next() {
				var refTable, refCol, fkName string
				if err := fkRows.Scan(&refTable, &refCol, &fkName); err == nil {
					col.ForeignKey = &ForeignKeyInfo{
						ReferencedTable:  refTable,
						ReferencedColumn: refCol,
						ConstraintName:   fkName,
					}
				}
			}
		}

		table.Columns = append(table.Columns, col)
	}

	return table, rows.Err()
}

func (pi *postgresIntrospector) GetAllTables(ctx context.Context) ([]TableInfo, error) {
	tables, err := pi.ListTables(ctx)
	if err != nil {
		return nil, err
	}

	var result []TableInfo
	for _, tableName := range tables {
		table, err := pi.InspectTable(ctx, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect table %s: %w", tableName, err)
		}
		result = append(result, *table)
	}

	return result, nil
}

func (pi *postgresIntrospector) Close() error {
	return pi.conn.Close(context.Background())
}
