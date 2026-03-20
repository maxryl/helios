package db

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SchemaCache holds database metadata for autocomplete suggestions.
// It is safe for concurrent use; RefreshSchema may run in a background
// goroutine while Suggest/SuggestColumns are called from the UI thread.
type SchemaCache struct {
	mu        sync.RWMutex
	Tables    []string            // table names
	Columns   map[string][]string // table_name -> column names
	Functions []string            // function names
}

// NewSchemaCache returns an empty cache. Suggestions are limited to database
// objects (tables, columns, functions) — SQL keywords are intentionally
// excluded so the autocomplete never interrupts the user with obvious
// language constructs like SELECT, FROM, WHERE, etc.
func NewSchemaCache() *SchemaCache {
	return &SchemaCache{
		Columns: make(map[string][]string),
	}
}

// RefreshSchema queries the database to populate Tables, Columns, and Functions.
// Keywords are preserved; all other fields are replaced.
func (sc *SchemaCache) RefreshSchema(ctx context.Context, pool *pgxpool.Pool) error {
	tables, err := sc.queryTables(ctx, pool)
	if err != nil {
		return fmt.Errorf("db: refresh tables: %w", err)
	}

	columns, err := sc.queryColumns(ctx, pool)
	if err != nil {
		return fmt.Errorf("db: refresh columns: %w", err)
	}

	functions, err := sc.queryFunctions(ctx, pool)
	if err != nil {
		return fmt.Errorf("db: refresh functions: %w", err)
	}

	sc.mu.Lock()
	sc.Tables = tables
	sc.Columns = columns
	sc.Functions = functions
	sc.mu.Unlock()
	return nil
}

func (sc *SchemaCache) queryTables(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	rows, err := pool.Query(ctx,
		`SELECT table_name FROM information_schema.tables
		 WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		 ORDER BY table_name`)
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

func (sc *SchemaCache) queryColumns(ctx context.Context, pool *pgxpool.Pool) (map[string][]string, error) {
	rows, err := pool.Query(ctx,
		`SELECT table_name, column_name FROM information_schema.columns
		 WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		 ORDER BY table_name, ordinal_position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make(map[string][]string)
	for rows.Next() {
		var table, col string
		if err := rows.Scan(&table, &col); err != nil {
			return nil, err
		}
		columns[table] = append(columns[table], col)
	}
	return columns, rows.Err()
}

func (sc *SchemaCache) queryFunctions(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	rows, err := pool.Query(ctx,
		`SELECT routine_name FROM information_schema.routines
		 WHERE routine_schema NOT IN ('pg_catalog', 'information_schema')
		 ORDER BY routine_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var funcs []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		funcs = append(funcs, name)
	}
	return funcs, rows.Err()
}

// Suggest returns matching completions for the given prefix.
// Matching is case-insensitive. Results are sorted with exact-case matches first,
// then alphabetically. At most 20 results are returned.
func (sc *SchemaCache) Suggest(prefix string) []string {
	if prefix == "" {
		return nil
	}

	sc.mu.RLock()
	defer sc.mu.RUnlock()

	upper := strings.ToUpper(prefix)
	var matches []string

	for _, t := range sc.Tables {
		if strings.HasPrefix(strings.ToUpper(t), upper) {
			matches = append(matches, t)
		}
	}
	for _, f := range sc.Functions {
		if strings.HasPrefix(strings.ToUpper(f), upper) {
			matches = append(matches, f)
		}
	}
	for _, cols := range sc.Columns {
		for _, c := range cols {
			if strings.HasPrefix(strings.ToUpper(c), upper) {
				matches = append(matches, c)
			}
		}
	}

	// Deduplicate.
	seen := make(map[string]bool, len(matches))
	deduped := matches[:0]
	for _, m := range matches {
		if !seen[m] {
			seen[m] = true
			deduped = append(deduped, m)
		}
	}
	matches = deduped

	// Sort: exact-case prefix first, then alphabetical.
	sort.Slice(matches, func(i, j int) bool {
		iExact := strings.HasPrefix(matches[i], prefix)
		jExact := strings.HasPrefix(matches[j], prefix)
		if iExact != jExact {
			return iExact
		}
		return strings.ToLower(matches[i]) < strings.ToLower(matches[j])
	})

	if len(matches) > 20 {
		matches = matches[:20]
	}
	return matches
}

// SuggestTables returns table names matching the given prefix (case-insensitive).
func (sc *SchemaCache) SuggestTables(prefix string) []string {
	if prefix == "" {
		return nil
	}

	sc.mu.RLock()
	defer sc.mu.RUnlock()

	upper := strings.ToUpper(prefix)
	var matches []string
	for _, t := range sc.Tables {
		if strings.HasPrefix(strings.ToUpper(t), upper) {
			matches = append(matches, t)
		}
	}

	sort.Strings(matches)
	if len(matches) > 20 {
		matches = matches[:20]
	}
	return matches
}

// SuggestAllColumns returns column names from all tables matching the given
// prefix (case-insensitive). Duplicates are removed.
func (sc *SchemaCache) SuggestAllColumns(prefix string) []string {
	if prefix == "" {
		return nil
	}

	sc.mu.RLock()
	defer sc.mu.RUnlock()

	upper := strings.ToUpper(prefix)
	seen := make(map[string]bool)
	var matches []string
	for _, cols := range sc.Columns {
		for _, c := range cols {
			if strings.HasPrefix(strings.ToUpper(c), upper) && !seen[c] {
				seen[c] = true
				matches = append(matches, c)
			}
		}
	}

	sort.Strings(matches)
	if len(matches) > 20 {
		matches = matches[:20]
	}
	return matches
}

// SuggestColumns returns column names for a specific table.
// Table name lookup is case-insensitive. Returns nil if the table is unknown.
func (sc *SchemaCache) SuggestColumns(tableName string) []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	lower := strings.ToLower(tableName)
	for t, cols := range sc.Columns {
		if strings.ToLower(t) == lower {
			result := make([]string, len(cols))
			copy(result, cols)
			return result
		}
	}
	return nil
}
