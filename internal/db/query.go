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
	Truncated bool // true if results were capped at MaxRows
}

// MaxRows is the maximum number of rows to load into memory.
const MaxRows = 50000

// IsQuerySQL returns true if the SQL should be routed to the Query path
// (i.e. it returns rows) rather than the Exec path.
func IsQuerySQL(sql string) bool {
	trimmed := strings.ToUpper(strings.TrimSpace(sql))
	for _, prefix := range []string{"SELECT", "WITH", "TABLE", "VALUES", "SHOW", "EXPLAIN"} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	if strings.Contains(trimmed, "RETURNING") {
		return true
	}
	return false
}

// ExecuteQuery runs a SQL statement against the given Querier and returns a QueryResult.
func ExecuteQuery(ctx context.Context, q Querier, sql string) QueryResult {
	start := time.Now()

	if IsQuerySQL(sql) {
		return executeSelect(ctx, q, sql, start)
	}
	return executeExec(ctx, q, sql, start)
}

// StreamCallback is called with batches of rows as they're read.
// columns is sent once on the first call. Return false to stop reading.
type StreamCallback func(columns []string, batch [][]string, totalSoFar int) bool

// ExecuteQueryStreaming runs a SELECT and calls cb with batches of rows.
// This allows the UI to display partial results while reading continues.
func ExecuteQueryStreaming(ctx context.Context, q Querier, sql string, batchSize int, cb StreamCallback) QueryResult {
	start := time.Now()

	rows, err := q.Query(ctx, sql)
	if err != nil {
		return QueryResult{Duration: time.Since(start), Error: err}
	}
	defer rows.Close()

	fds := rows.FieldDescriptions()
	columns := make([]string, len(fds))
	for i, fd := range fds {
		columns[i] = fd.Name
	}

	total := 0
	truncated := false
	batch := make([][]string, 0, batchSize)

	for rows.Next() {
		if total >= MaxRows {
			truncated = true
			break
		}

		vals, err := rows.Values()
		if err != nil {
			return QueryResult{Duration: time.Since(start), Error: err}
		}
		row := convertRow(vals)
		batch = append(batch, row)
		total++

		if len(batch) >= batchSize {
			if !cb(columns, batch, total) {
				break
			}
			batch = make([][]string, 0, batchSize)
		}
	}

	// Flush remaining rows.
	if len(batch) > 0 {
		cb(columns, batch, total)
	}

	if err := rows.Err(); err != nil {
		return QueryResult{Duration: time.Since(start), Error: err}
	}

	return QueryResult{
		Columns:   columns,
		RowCount:  total,
		Duration:  time.Since(start),
		Truncated: truncated,
	}
}

func executeSelect(ctx context.Context, q Querier, sql string, start time.Time) QueryResult {
	rows, err := q.Query(ctx, sql)
	if err != nil {
		return QueryResult{Duration: time.Since(start), Error: err}
	}
	defer rows.Close()

	fds := rows.FieldDescriptions()
	columns := make([]string, len(fds))
	for i, fd := range fds {
		columns[i] = fd.Name
	}

	resultRows := make([][]string, 0, 256)
	truncated := false
	for rows.Next() {
		if len(resultRows) >= MaxRows {
			truncated = true
			break
		}
		vals, err := rows.Values()
		if err != nil {
			return QueryResult{Duration: time.Since(start), Error: err}
		}
		resultRows = append(resultRows, convertRow(vals))
	}

	if err := rows.Err(); err != nil {
		return QueryResult{Duration: time.Since(start), Error: err}
	}

	return QueryResult{
		Columns:   columns,
		Rows:      resultRows,
		RowCount:  len(resultRows),
		Duration:  time.Since(start),
		Truncated: truncated,
	}
}

func convertRow(vals []any) []string {
	row := make([]string, len(vals))
	for i, v := range vals {
		if v == nil {
			row[i] = "<NULL>"
		} else {
			switch tv := v.(type) {
			case string:
				row[i] = tv
			case int64:
				row[i] = fmt.Sprintf("%d", tv)
			case float64:
				row[i] = fmt.Sprintf("%g", tv)
			case bool:
				if tv {
					row[i] = "true"
				} else {
					row[i] = "false"
				}
			case int32:
				row[i] = fmt.Sprintf("%d", tv)
			case []byte:
				row[i] = string(tv)
			default:
				row[i] = fmt.Sprintf("%v", v)
			}
		}
	}
	return row
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
