package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querier is the common interface satisfied by both *pgxpool.Pool and pgx.Tx.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// QueryResult holds the outcome of a SQL execution.
type QueryResult struct {
	Columns  []string
	Rows     [][]string
	RowCount int
	Message  string
	Duration time.Duration
	Error    error
}

// isQuerySQL returns true if the SQL should be routed to the Query path
// (i.e. it returns rows) rather than the Exec path.
// Covers SELECT, WITH, TABLE, VALUES, SHOW, EXPLAIN, and RETURNING clauses.
func isQuerySQL(sql string) bool {
	trimmed := strings.ToUpper(strings.TrimSpace(sql))
	for _, prefix := range []string{"SELECT", "WITH", "TABLE", "VALUES", "SHOW", "EXPLAIN"} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	// DML with RETURNING clause returns rows.
	if strings.Contains(trimmed, "RETURNING") {
		return true
	}
	return false
}

// ExecuteQuery runs a SQL statement against the given Querier and returns a QueryResult.
func ExecuteQuery(ctx context.Context, q Querier, sql string) QueryResult {
	start := time.Now()

	if isQuerySQL(sql) {
		return executeSelect(ctx, q, sql, start)
	}
	return executeExec(ctx, q, sql, start)
}

func executeSelect(ctx context.Context, q Querier, sql string, start time.Time) QueryResult {
	rows, err := q.Query(ctx, sql)
	if err != nil {
		return QueryResult{Duration: time.Since(start), Error: err}
	}
	defer rows.Close()

	// Column names from field descriptions.
	fds := rows.FieldDescriptions()
	columns := make([]string, len(fds))
	for i, fd := range fds {
		columns[i] = fd.Name
	}

	var resultRows [][]string
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return QueryResult{Duration: time.Since(start), Error: err}
		}
		row := make([]string, len(vals))
		for i, v := range vals {
			if v == nil {
				row[i] = "<NULL>"
			} else {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		resultRows = append(resultRows, row)
	}

	if err := rows.Err(); err != nil {
		return QueryResult{Duration: time.Since(start), Error: err}
	}

	return QueryResult{
		Columns:  columns,
		Rows:     resultRows,
		RowCount: len(resultRows),
		Duration: time.Since(start),
	}
}

func executeExec(ctx context.Context, q Querier, sql string, start time.Time) QueryResult {
	tag, err := q.Exec(ctx, sql)
	if err != nil {
		return QueryResult{Duration: time.Since(start), Error: err}
	}

	return QueryResult{
		RowCount: int(tag.RowsAffected()),
		Message:  tag.String(),
		Duration: time.Since(start),
	}
}
