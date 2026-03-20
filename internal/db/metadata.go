package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DatabaseMeta holds the full schema metadata for a database connection.
type DatabaseMeta struct {
	Schemas []SchemaMeta
}

// SchemaMeta holds metadata for a single database schema.
type SchemaMeta struct {
	Name      string
	Tables    []TableMeta
	Functions []string
}

// TableMeta holds metadata for a single table.
type TableMeta struct {
	Name        string
	Columns     []ColumnMeta
	Indexes     []string
	Constraints []string
	Triggers    []string
}

// ColumnMeta holds metadata for a single column.
type ColumnMeta struct {
	Name     string
	DataType string
	Nullable bool
}

const excludedSchemas = `('pg_catalog', 'information_schema', 'pg_toast')`

// FetchDatabaseMeta queries PostgreSQL system catalogs and returns the full
// schema tree for the connected database.
func FetchDatabaseMeta(ctx context.Context, pool *pgxpool.Pool) (*DatabaseMeta, error) {
	schemas, err := fetchSchemas(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("db: fetch schemas: %w", err)
	}

	// Index schemas by name for easy lookup.
	schemaMap := make(map[string]*SchemaMeta, len(schemas))
	meta := &DatabaseMeta{Schemas: schemas}
	for i := range meta.Schemas {
		schemaMap[meta.Schemas[i].Name] = &meta.Schemas[i]
	}

	if err := fetchTables(ctx, pool, schemaMap); err != nil {
		return nil, fmt.Errorf("db: fetch tables: %w", err)
	}

	// Build table index: schema.table -> *TableMeta
	tableMap := make(map[string]*TableMeta)
	for i := range meta.Schemas {
		for j := range meta.Schemas[i].Tables {
			key := meta.Schemas[i].Name + "." + meta.Schemas[i].Tables[j].Name
			tableMap[key] = &meta.Schemas[i].Tables[j]
		}
	}

	if err := fetchColumns(ctx, pool, tableMap); err != nil {
		return nil, fmt.Errorf("db: fetch columns: %w", err)
	}
	if err := fetchIndexes(ctx, pool, tableMap); err != nil {
		return nil, fmt.Errorf("db: fetch indexes: %w", err)
	}
	if err := fetchConstraints(ctx, pool, tableMap); err != nil {
		return nil, fmt.Errorf("db: fetch constraints: %w", err)
	}
	if err := fetchTriggers(ctx, pool, tableMap); err != nil {
		return nil, fmt.Errorf("db: fetch triggers: %w", err)
	}
	if err := fetchFunctions(ctx, pool, schemaMap); err != nil {
		return nil, fmt.Errorf("db: fetch functions: %w", err)
	}

	return meta, nil
}

func fetchSchemas(ctx context.Context, pool *pgxpool.Pool) ([]SchemaMeta, error) {
	rows, err := pool.Query(ctx,
		`SELECT schema_name FROM information_schema.schemata
		 WHERE schema_name NOT IN `+excludedSchemas+`
		 ORDER BY schema_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schemas []SchemaMeta
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		schemas = append(schemas, SchemaMeta{Name: name})
	}
	return schemas, rows.Err()
}

func fetchTables(ctx context.Context, pool *pgxpool.Pool, schemas map[string]*SchemaMeta) error {
	rows, err := pool.Query(ctx,
		`SELECT table_schema, table_name FROM information_schema.tables
		 WHERE table_schema NOT IN `+excludedSchemas+`
		 AND table_type = 'BASE TABLE'
		 ORDER BY table_schema, table_name`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table string
		if err := rows.Scan(&schema, &table); err != nil {
			return err
		}
		if s, ok := schemas[schema]; ok {
			s.Tables = append(s.Tables, TableMeta{Name: table})
		}
	}
	return rows.Err()
}

func fetchColumns(ctx context.Context, pool *pgxpool.Pool, tables map[string]*TableMeta) error {
	rows, err := pool.Query(ctx,
		`SELECT table_schema, table_name, column_name, data_type, is_nullable
		 FROM information_schema.columns
		 WHERE table_schema NOT IN `+excludedSchemas+`
		 ORDER BY table_schema, table_name, ordinal_position`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table, col, dtype, nullable string
		if err := rows.Scan(&schema, &table, &col, &dtype, &nullable); err != nil {
			return err
		}
		key := schema + "." + table
		if t, ok := tables[key]; ok {
			t.Columns = append(t.Columns, ColumnMeta{
				Name:     col,
				DataType: dtype,
				Nullable: nullable == "YES",
			})
		}
	}
	return rows.Err()
}

func fetchIndexes(ctx context.Context, pool *pgxpool.Pool, tables map[string]*TableMeta) error {
	rows, err := pool.Query(ctx,
		`SELECT schemaname, tablename, indexname FROM pg_indexes
		 WHERE schemaname NOT IN `+excludedSchemas+`
		 ORDER BY schemaname, tablename, indexname`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table, idx string
		if err := rows.Scan(&schema, &table, &idx); err != nil {
			return err
		}
		key := schema + "." + table
		if t, ok := tables[key]; ok {
			t.Indexes = append(t.Indexes, idx)
		}
	}
	return rows.Err()
}

func fetchConstraints(ctx context.Context, pool *pgxpool.Pool, tables map[string]*TableMeta) error {
	rows, err := pool.Query(ctx,
		`SELECT table_schema, table_name, constraint_name || ' (' || constraint_type || ')'
		 FROM information_schema.table_constraints
		 WHERE table_schema NOT IN `+excludedSchemas+`
		 ORDER BY table_schema, table_name, constraint_name`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table, con string
		if err := rows.Scan(&schema, &table, &con); err != nil {
			return err
		}
		key := schema + "." + table
		if t, ok := tables[key]; ok {
			t.Constraints = append(t.Constraints, con)
		}
	}
	return rows.Err()
}

func fetchTriggers(ctx context.Context, pool *pgxpool.Pool, tables map[string]*TableMeta) error {
	rows, err := pool.Query(ctx,
		`SELECT trigger_schema, event_object_table, trigger_name
		 FROM information_schema.triggers
		 WHERE trigger_schema NOT IN `+excludedSchemas+`
		 ORDER BY trigger_schema, event_object_table, trigger_name`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table, trig string
		if err := rows.Scan(&schema, &table, &trig); err != nil {
			return err
		}
		key := schema + "." + table
		if t, ok := tables[key]; ok {
			t.Triggers = append(t.Triggers, trig)
		}
	}
	return rows.Err()
}

func fetchFunctions(ctx context.Context, pool *pgxpool.Pool, schemas map[string]*SchemaMeta) error {
	rows, err := pool.Query(ctx,
		`SELECT routine_schema, routine_name
		 FROM information_schema.routines
		 WHERE routine_schema NOT IN `+excludedSchemas+`
		 AND routine_type = 'FUNCTION'
		 ORDER BY routine_schema, routine_name`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var schema, fn string
		if err := rows.Scan(&schema, &fn); err != nil {
			return err
		}
		if s, ok := schemas[schema]; ok {
			s.Functions = append(s.Functions, fn)
		}
	}
	return rows.Err()
}
