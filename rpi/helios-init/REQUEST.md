# Helios - PostgreSQL Database GUI Client

## Context

Build a desktop PostgreSQL database client in Go using the Fyne GUI toolkit. The tool provides SQL terminals with transaction support, multi-database connectivity, and persisted connection configuration. UI inspired by TablePlus (sidebar + tabbed terminals) and DuckDB UI (inline results below editor).

## Project Structure

```
helios/
├── main.go
├── go.mod
├── internal/
│   ├── config/
│   │   └── config.go        # Connection config types, JSON load/save
│   ├── db/
│   │   ├── connection.go    # ConnectionManager: pool-per-config via pgxpool
│   │   └── query.go         # ExecuteQuery, QueryResult, Querier interface
│   └── ui/
│       ├── app.go           # Top-level window layout, menu, lifecycle
│       ├── sidebar.go       # Connection tree (widget.Tree)
│       ├── terminal.go      # SQL editor + results + transaction state
│       ├── terminal_tabs.go # DocTabs managing multiple terminals
│       ├── toolbar.go       # Run, Begin/Commit/Rollback, New Terminal
│       ├── connform.go      # Add/Edit connection dialog
│       └── resultsgrid.go   # widget.Table for query results
```

## UI Layout

```
+------------------------------------------------------------------+
|  Menu: File | Edit                                                |
+------------------------------------------------------------------+
|  Toolbar: [Run] | [Begin TX] [Commit] [Rollback] | [New Terminal]|
+------------------------------------------------------------------+
|  Sidebar (20%)  |  DocTabs                                       |
|                  |  +------------------------------------------+ |
|  Connection A    |  | Tab: mydb    | Tab: other [TX]  |        | |
|    (connected)   |  +------------------------------------------+ |
|  Connection B    |  | SQL Editor (MultiLine Entry)             | |
|    (disconnected)|  |                                          | |
|                  |  |  SELECT * FROM users;                    | |
|                  |  +------------------------------------------+ |
|                  |  | Status: 42 rows returned (12ms)          | |
|                  |  +------------------------------------------+ |
|                  |  | id | name    | email        | created_at | |
|                  |  | 1  | alice   | a@b.com      | 2026-01-01 | |
|                  |  | 2  | bob     | b@b.com      | 2026-01-02 | |
|                  |  +------------------------------------------+ |
+------------------------------------------------------------------+
```

- **HSplit**: sidebar (0.2) | terminal tabs (0.8)
- **Each terminal**: VSplit of editor (top) and results panel (bottom: status label + table)
- **DocTabs**: closable tabs, each an independent terminal with its own connection + transaction

## Dependencies

- `fyne.io/fyne/v2` - GUI toolkit
- `github.com/jackc/pgx/v5` - PostgreSQL driver with `pgxpool` for connection pooling
- `github.com/google/uuid` - Connection config IDs

## Implementation Phases

### Phase 1: Foundation (config, db layer, basic terminal)

**1. `go.mod` + `main.go`**
- Initialize module, install deps
- Thin entry point: create fyne app, load config, create ConnectionManager, build UI

**2. `internal/config/config.go`**
- `ConnectionConfig` struct: ID (uuid), Name, Host, Port, User, Password, DBName, SSLMode
- `AppConfig` struct with `[]ConnectionConfig`
- `Load()` reads from `~/.config/helios/connections.json` (via `os.UserConfigDir()`)
- `Save()` writes JSON. Called after every Add/Update/Remove
- `Add()`, `Update()`, `Remove()`, `FindByID()` methods

**3. `internal/db/connection.go`**
- `ConnectionManager` with mutex-protected `map[string]*pgxpool.Pool` keyed by config ID
- `Connect(ctx, cfg)` builds DSN, creates pool (idempotent - returns existing if present)
- `Disconnect(id)` closes pool and removes from map
- `Pool(id)`, `IsConnected(id)`, `CloseAll()`

**4. `internal/db/query.go`**
- `QueryResult` struct: Columns `[]string`, Rows `[][]string`, RowCount, Message, Duration, Error
- `Querier` interface (satisfied by both `*pgxpool.Pool` and `pgx.Tx`): `Query()` and `Exec()`
- `ExecuteQuery(ctx, querier, sql)` - executes SQL, returns QueryResult
  - SELECT: iterates rows, collects all values as strings, NULL rendered as `<NULL>`
  - DML/DDL: returns command tag as Message
  - Errors: populates QueryResult.Error

### Phase 2: UI shell (results grid, terminal, tabs, app layout)

**5. `internal/ui/resultsgrid.go`**
- Wraps `widget.NewTable()` with Length/CreateCell/UpdateCell callbacks
- Row 0 = bold headers from QueryResult.Columns
- `SetData(result)` stores result and refreshes table
- Column widths estimated from header name length (min 100, max 300)

**6. `internal/ui/terminal.go`**
- `Terminal` struct: editor (`widget.Entry` multiline, monospace), ResultsGrid, status label, txState, pgx.Tx, cancel func
- `Content()` returns `container.NewVSplit(editor, container.NewVBox(statusLabel, resultsGrid))`
- `RunQuery()`: get SQL from editor, execute in goroutine via `db.ExecuteQuery`, update grid + status on completion
- `BeginTx()`: acquire `pgx.Tx` from pool, set txState=Active, update tab label to show `[TX]`
- `CommitTx()` / `RollbackTx()`: commit/rollback tx, clear state
- `Close()`: rollback any open tx, cancel any running query
- `querier()` helper: returns tx if active, else pool

**7. `internal/ui/terminal_tabs.go`**
- `TerminalTabs` wraps `container.NewDocTabs()`
- `NewTerminal(configID)`: auto-connects if needed, creates Terminal, adds tab, selects it
- `ActiveTerminal()` returns terminal for selected tab
- `CloseIntercept` calls `terminal.Close()` before removing

**8. `internal/ui/app.go`**
- Assembles: `container.NewBorder(toolbar, nil, nil, nil, container.NewHSplit(sidebar, tabs))`
- Main menu: File > New Connection, New Terminal, Quit
- `window.SetOnClosed` calls `connMgr.CloseAll()`
- Keyboard shortcuts: Ctrl+Enter (run query), Ctrl+T (new terminal), Ctrl+W (close tab)

### Phase 3: Connection management (sidebar, dialogs)

**9. `internal/ui/connform.go`**
- `ShowConnectionDialog(window, existing, onSave)` using `dialog.ShowForm()`
- Fields: Name, Host, Port (default 5432), User, Password (password entry), Database, SSL Mode (select)
- Edit mode pre-populates from existing config
- Validates required fields before calling onSave

**10. `internal/ui/sidebar.go`**
- `widget.Tree` with flat list of connections (one level)
- Each node shows connection name + status indicator (connected/disconnected)
- Single-click: opens new terminal tab for that connection
- Right-click via `widget.NewPopUpMenu`: Connect, Disconnect, Edit, Delete
- `Refresh()` rebuilds tree data after config changes

### Phase 4: Toolbar + polish

**11. `internal/ui/toolbar.go`**
- Actions: Run (play icon), separator, Begin TX, Commit, Rollback, separator, New Terminal
- Each action delegates to `ActiveTerminal()` methods
- Shows error dialog if no active terminal or wrong tx state

**12. Polish**
- Error dialogs for connection failures
- Graceful shutdown (rollback open txs, close pools)
- Tab labels show `connName` or `connName [TX]` when transaction is active
- Status label shows row count, duration, or error message

## Key Design Decisions

| Decision | Rationale |
|---|---|
| **pgx/v5 over lib/pq** | Actively maintained, native pgxpool, better Postgres type support |
| **One pgxpool.Pool per connection config** | pgxpool is concurrency-safe; multiple terminals share one pool efficiently |
| **Querier interface (pool + tx)** | Terminal switches between pool and transaction transparently |
| **All result values as `[]string`** | Display-only for MVP; keeps UI layer simple |
| **JSON config file (not Fyne prefs)** | Structured data maps better to JSON; human-readable and editable |
| **`internal/` packages** | Standard Go convention for non-exported packages |
| **Passwords in plaintext (MVP)** | Simple for now; note in UI, future: OS keyring integration |

## Known Limitations (MVP)

- No syntax highlighting in SQL editor (Fyne Entry limitation)
- Multi-statement SQL returns only last result set
- No schema/table browsing in sidebar (flat connection list only)
- No data export (CSV, JSON)
- Large result sets loaded fully into memory (widget.Table virtualizes rendering but data is in-memory)

## Verification

1. `go build ./...` compiles without errors
2. Launch app, add a Postgres connection via sidebar right-click > New or File > New Connection
3. Verify connection persists after app restart (check `~/.config/helios/connections.json`)
4. Click connection in sidebar to open terminal tab
5. Type `SELECT version();` and click Run (or Ctrl+Enter) - verify result appears in grid
6. Click Begin TX, run `CREATE TABLE test_tx (id int);`, click Rollback, verify table doesn't exist
7. Open multiple terminals to different databases, verify independent operation
8. Close terminal with active transaction, verify it rolls back cleanly
9. Close app, verify all connections shut down (no leaked pools)
