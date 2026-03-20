package db

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"helios/internal/config"
)

func configFromFields(user, pass, host string, port int, dbname, sslmode string) config.ConnectionConfig {
	return config.ConnectionConfig{
		ID:       "test",
		Name:     "test",
		Host:     host,
		Port:     port,
		User:     user,
		Password: pass,
		DBName:   dbname,
		SSLMode:  sslmode,
	}
}

// --- mock Querier ----------------------------------------------------------

type mockQuerier struct {
	execFn  func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	queryFn func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (m *mockQuerier) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return m.execFn(ctx, sql, args...)
}

func (m *mockQuerier) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return m.queryFn(ctx, sql, args...)
}

// --- IsQuerySQL tests ------------------------------------------------------

func TestIsQuerySQL(t *testing.T) {
	tests := []struct {
		sql  string
		want bool
	}{
		{"SELECT 1", true},
		{"select * from t", true},
		{"  SELECT 1", true},
		{"\n\tSELECT 1", true},
		{"WITH cte AS (SELECT 1) SELECT * FROM cte", true},
		{"TABLE my_table", true},
		{"VALUES (1,2)", true},
		{"SHOW server_version", true},
		{"show all", true},
		{"INSERT INTO t VALUES (1)", false},
		{"UPDATE t SET x=1", false},
		{"DELETE FROM t", false},
		{"CREATE TABLE t (id int)", false},
		{"DROP TABLE t", false},
		{"ALTER TABLE t ADD COLUMN x int", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := IsQuerySQL(tt.sql); got != tt.want {
			t.Errorf("IsQuerySQL(%q) = %v, want %v", tt.sql, got, tt.want)
		}
	}
}

// --- ExecuteQuery routing tests -------------------------------------------

func TestExecuteQuery_ExecPath(t *testing.T) {
	called := false
	mq := &mockQuerier{
		execFn: func(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
			called = true
			return pgconn.NewCommandTag("INSERT 0 1"), nil
		},
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			t.Fatal("Query should not be called for INSERT")
			return nil, nil
		},
	}

	result := ExecuteQuery(context.Background(), mq, "INSERT INTO t VALUES (1)")
	if !called {
		t.Fatal("Exec was not called")
	}
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.Message != "INSERT 0 1" {
		t.Errorf("Message = %q, want %q", result.Message, "INSERT 0 1")
	}
	if result.RowCount != 1 {
		t.Errorf("RowCount = %d, want 1", result.RowCount)
	}
	if result.Duration <= 0 {
		t.Error("Duration should be positive")
	}
}

func TestExecuteQuery_ExecError(t *testing.T) {
	errBoom := errors.New("boom")
	mq := &mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, errBoom
		},
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			t.Fatal("Query should not be called")
			return nil, nil
		},
	}

	result := ExecuteQuery(context.Background(), mq, "DROP TABLE boom")
	if !errors.Is(result.Error, errBoom) {
		t.Errorf("Error = %v, want %v", result.Error, errBoom)
	}
	if result.Duration <= 0 {
		t.Error("Duration should be positive even on error")
	}
}

func TestExecuteQuery_QueryError(t *testing.T) {
	errBoom := errors.New("query boom")
	mq := &mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
			t.Fatal("Exec should not be called for SELECT")
			return pgconn.CommandTag{}, nil
		},
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return nil, errBoom
		},
	}

	result := ExecuteQuery(context.Background(), mq, "SELECT 1")
	if !errors.Is(result.Error, errBoom) {
		t.Errorf("Error = %v, want %v", result.Error, errBoom)
	}
}

func TestExecuteQuery_SelectRoutesToQuery(t *testing.T) {
	queryCalled := false
	mq := &mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
			t.Fatal("Exec should not be called for SELECT")
			return pgconn.CommandTag{}, nil
		},
		queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			queryCalled = true
			// Return an error to keep mock simple (we already test routing).
			return nil, errors.New("not a real db")
		},
	}

	_ = ExecuteQuery(context.Background(), mq, "SELECT * FROM t")
	if !queryCalled {
		t.Fatal("Query was not called for SELECT statement")
	}
}

// --- ConnectionManager unit tests -----------------------------------------

func TestConnectionManager_PoolAndIsConnected(t *testing.T) {
	cm := NewConnectionManager()

	if cm.IsConnected("missing") {
		t.Error("IsConnected should be false for unknown ID")
	}
	if cm.Pool("missing") != nil {
		t.Error("Pool should be nil for unknown ID")
	}
}

func TestConnectionManager_DisconnectNoop(t *testing.T) {
	cm := NewConnectionManager()
	if err := cm.Disconnect("nope"); err != nil {
		t.Errorf("Disconnect on missing ID should not error, got: %v", err)
	}
}

func TestConnectionManager_CloseAllEmpty(t *testing.T) {
	cm := NewConnectionManager()
	cm.CloseAll() // should not panic
}

// --- buildDSN tests -------------------------------------------------------

func TestBuildDSN(t *testing.T) {
	tests := []struct {
		name string
		cfg  struct {
			user, pass, host string
			port             int
			dbname, sslmode  string
		}
		want string
	}{
		{
			name: "basic",
			cfg: struct {
				user, pass, host string
				port             int
				dbname, sslmode  string
			}{"admin", "secret", "localhost", 5432, "mydb", "disable"},
			want: "postgres://admin:secret@localhost:5432/mydb?sslmode=disable",
		},
		{
			name: "special chars in password",
			cfg: struct {
				user, pass, host string
				port             int
				dbname, sslmode  string
			}{"user", "p@ss:word/here", "db.example.com", 5433, "prod", "require"},
			want: "postgres://user:p%40ss%3Aword%2Fhere@db.example.com:5433/prod?sslmode=require",
		},
		{
			name: "no sslmode",
			cfg: struct {
				user, pass, host string
				port             int
				dbname, sslmode  string
			}{"u", "p", "h", 5432, "d", ""},
			want: "postgres://u:p@h:5432/d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We need a config.ConnectionConfig but importing it would create a
			// circular-ish concern in tests. We call buildDSN directly which
			// takes config.ConnectionConfig, so we do import it.
			cfg := configFromFields(tt.cfg.user, tt.cfg.pass, tt.cfg.host, tt.cfg.port, tt.cfg.dbname, tt.cfg.sslmode)
			got := buildDSN(cfg)
			if got != tt.want {
				t.Errorf("buildDSN() = %q, want %q", got, tt.want)
			}
		})
	}
}
