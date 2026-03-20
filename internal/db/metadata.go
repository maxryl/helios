package db

import (
	"context"
	"strings"

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
	Loaded    bool // true once tables+functions have been fetched
}

// TableMeta holds metadata for a single table.
type TableMeta struct {
	Name        string
	Columns     []ColumnMeta
	Indexes     []string
	Constraints []string
	Triggers    []string
	Loaded      bool // true once columns/indexes/etc. have been fetched
}

// ColumnMeta holds metadata for a single column.
type ColumnMeta struct {
	Name     string
	DataType string
	Nullable bool
}

const excludedSchemas = `('pg_catalog', 'information_schema', 'pg_toast')`

// FetchSchemas returns the list of user schemas (fast — single lightweight query).
func FetchSchemas(ctx context.Context, pool *pgxpool.Pool) ([]SchemaMeta, error) {
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

// FetchSchemaContent populates tables and functions for a single schema.
func FetchSchemaContent(ctx context.Context, pool *pgxpool.Pool, schema *SchemaMeta) error {
	// Tables
	rows, err := pool.Query(ctx,
		`SELECT table_name FROM information_schema.tables
		 WHERE table_schema = $1 AND table_type = 'BASE TABLE'
		 ORDER BY table_name`, schema.Name)
	if err != nil {
		return err
	}
	defer rows.Close()

	schema.Tables = nil
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		schema.Tables = append(schema.Tables, TableMeta{Name: name})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Functions
	rows2, err := pool.Query(ctx,
		`SELECT p.proname || '(' || COALESCE(pg_get_function_arguments(p.oid), '') || ')'
		 FROM pg_proc p
		 JOIN pg_namespace n ON n.oid = p.pronamespace
		 WHERE n.nspname = $1 AND p.prokind = 'f'
		 ORDER BY 1`, schema.Name)
	if err != nil {
		return err
	}
	defer rows2.Close()

	schema.Functions = nil
	for rows2.Next() {
		var name string
		if err := rows2.Scan(&name); err != nil {
			return err
		}
		schema.Functions = append(schema.Functions, name)
	}
	if err := rows2.Err(); err != nil {
		return err
	}

	schema.Loaded = true
	return nil
}

// FetchTableDetail populates columns, indexes, constraints, and triggers for a single table.
func FetchTableDetail(ctx context.Context, pool *pgxpool.Pool, schemaName string, table *TableMeta) error {
	// Columns
	rows, err := pool.Query(ctx,
		`SELECT column_name, data_type, is_nullable
		 FROM information_schema.columns
		 WHERE table_schema = $1 AND table_name = $2
		 ORDER BY ordinal_position`, schemaName, table.Name)
	if err != nil {
		return err
	}
	defer rows.Close()

	table.Columns = nil
	for rows.Next() {
		var col, dtype, nullable string
		if err := rows.Scan(&col, &dtype, &nullable); err != nil {
			return err
		}
		table.Columns = append(table.Columns, ColumnMeta{
			Name: col, DataType: dtype, Nullable: nullable == "YES",
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Indexes
	rows2, err := pool.Query(ctx,
		`SELECT indexname FROM pg_indexes
		 WHERE schemaname = $1 AND tablename = $2
		 ORDER BY indexname`, schemaName, table.Name)
	if err != nil {
		return err
	}
	defer rows2.Close()

	table.Indexes = nil
	for rows2.Next() {
		var name string
		if err := rows2.Scan(&name); err != nil {
			return err
		}
		table.Indexes = append(table.Indexes, name)
	}
	if err := rows2.Err(); err != nil {
		return err
	}

	// Constraints
	rows3, err := pool.Query(ctx,
		`SELECT constraint_name || ' (' || constraint_type || ')'
		 FROM information_schema.table_constraints
		 WHERE table_schema = $1 AND table_name = $2
		 ORDER BY constraint_name`, schemaName, table.Name)
	if err != nil {
		return err
	}
	defer rows3.Close()

	table.Constraints = nil
	for rows3.Next() {
		var name string
		if err := rows3.Scan(&name); err != nil {
			return err
		}
		table.Constraints = append(table.Constraints, name)
	}
	if err := rows3.Err(); err != nil {
		return err
	}

	// Triggers
	rows4, err := pool.Query(ctx,
		`SELECT trigger_name FROM information_schema.triggers
		 WHERE trigger_schema = $1 AND event_object_table = $2
		 ORDER BY trigger_name`, schemaName, table.Name)
	if err != nil {
		return err
	}
	defer rows4.Close()

	table.Triggers = nil
	for rows4.Next() {
		var name string
		if err := rows4.Scan(&name); err != nil {
			return err
		}
		table.Triggers = append(table.Triggers, name)
	}
	if err := rows4.Err(); err != nil {
		return err
	}

	table.Loaded = true
	return nil
}

// FetchFunctionDef returns the full CREATE OR REPLACE FUNCTION definition.
// funcSignature is "name(args)" as displayed in the tree.
func FetchFunctionDef(ctx context.Context, pool *pgxpool.Pool, schemaName, funcSignature string) (string, error) {
	// Extract bare name: everything before the first '('.
	funcName := funcSignature
	if idx := strings.Index(funcSignature, "("); idx >= 0 {
		funcName = funcSignature[:idx]
	}

	// Match the exact overload by comparing the full signature.
	var def string
	err := pool.QueryRow(ctx,
		`SELECT pg_get_functiondef(p.oid)
		 FROM pg_proc p
		 JOIN pg_namespace n ON n.oid = p.pronamespace
		 WHERE n.nspname = $1
		   AND p.proname || '(' || COALESCE(pg_get_function_arguments(p.oid), '') || ')' = $2
		 LIMIT 1`, schemaName, funcSignature).Scan(&def)
	if err != nil {
		// Fallback: match by name only (first overload).
		err = pool.QueryRow(ctx,
			`SELECT pg_get_functiondef(p.oid)
			 FROM pg_proc p
			 JOIN pg_namespace n ON n.oid = p.pronamespace
			 WHERE n.nspname = $1 AND p.proname = $2
			 LIMIT 1`, schemaName, funcName).Scan(&def)
	}
	return def, err
}
