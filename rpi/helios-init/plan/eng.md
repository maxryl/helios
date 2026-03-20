# Helios -- Technical Specification

## Overview

Helios is a greenfield PostgreSQL database GUI client built with Go, the Fyne v2
GUI toolkit, and the pgx/v5 PostgreSQL driver. The application provides a
multi-tab SQL editor with transaction support, a connection sidebar, and
virtualized result grids.

Target scope: approximately 12 source files, 1,100--1,200 lines of application
code, and roughly 400 lines of tests.

Module path: `helios` (private project, no remote import path required).

---

## Dependencies

```
require (
    fyne.io/fyne/v2   v2.7.x
    github.com/jackc/pgx/v5   v5.8.x
    github.com/google/uuid    v1.6.x
)
```

Fyne requires CGo for platform rendering. Cross-compilation uses `fyne-cross`.

---

## Package Structure

```
helios/
  main.go                       # Entry point
  go.mod
  internal/
    config/
      config.go                 # ConnectionConfig, AppConfig, JSON persistence
    db/
      connection.go             # ConnectionManager with pgxpool pools
      query.go                  # Querier interface, ExecuteQuery, QueryResult
      schema.go                 # Schema introspection, metadata cache
    ui/
      app.go                    # Window layout, menu, lifecycle, shortcuts
      sidebar.go                # widget.List for connections
      terminal.go               # SQL editor, results, transaction state
      terminal_tabs.go          # DocTabs managing terminals
      toolbar.go                # Action toolbar
      connform.go               # Connection add/edit dialog
      completer.go              # Autocomplete popup and word extraction
      resultsgrid.go            # widget.Table for query results
```

---

## Type Definitions and Interfaces

### internal/config/config.go

```go
type ConnectionConfig struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Host     string `json:"host"`
    Port     int    `json:"port"`
    User     string `json:"user"`
    Password string `json:"password"`
    DBName   string `json:"dbname"`
    SSLMode  string `json:"sslmode"`
}

type AppConfig struct {
    Connections []ConnectionConfig `json:"connections"`
}
```

AppConfig methods:

- `Load(path string) error` -- read and unmarshal the JSON config file.
- `Save(path string) error` -- marshal and write the JSON config file.
- `Add(cfg ConnectionConfig)` -- append a connection; generates a UUID if ID is empty.
- `Update(cfg ConnectionConfig) error` -- replace an existing connection by ID.
- `Remove(id string) error` -- delete a connection by ID.
- `FindByID(id string) (*ConnectionConfig, error)` -- look up a connection by ID.

The config file is stored as plain JSON on disk. Passwords are stored in
plaintext. This is a known limitation; keyring integration is deferred to
post-MVP.

### internal/db/query.go

```go
type Querier interface {
    Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type QueryResult struct {
    Columns  []string
    Rows     [][]string
    RowCount int
    Message  string
    Duration time.Duration
    Error    error
}

func ExecuteQuery(ctx context.Context, q Querier, sql string) QueryResult
```

Both `*pgxpool.Pool` and `pgx.Tx` satisfy the `Querier` interface implicitly.
This was verified during research -- no adapter types are needed.

`ExecuteQuery` inspects the SQL string to determine whether to call `Query`
(SELECT-like statements) or `Exec` (DML/DDL). The function measures wall-clock
duration, collects column names and stringified row data, and packages everything
into a `QueryResult`. Errors are returned inline in the `QueryResult.Error`
field rather than as a separate return value.

### internal/db/connection.go

```go
type ConnectionManager struct {
    mu    sync.RWMutex
    pools map[string]*pgxpool.Pool
}
```

ConnectionManager methods:

- `Connect(ctx context.Context, cfg ConnectionConfig) (*pgxpool.Pool, error)` --
  idempotent. If a pool already exists for the given config ID, return it.
  Otherwise build a DSN from the config, create a new `pgxpool.Pool`, store it,
  and return it.
- `Disconnect(id string) error` -- close the pool and remove it from the map.
- `Pool(id string) *pgxpool.Pool` -- return the pool for a config ID, or nil.
- `IsConnected(id string) bool` -- check whether a pool exists for the ID.
- `CloseAll()` -- close every pool and clear the map.

All methods are safe for concurrent use; the RWMutex protects the map.

### internal/ui/terminal.go

```go
type TxState int

const (
    TxNone   TxState = iota
    TxActive
)

type Terminal struct {
    editor     *widget.Entry
    results    *ResultsGrid
    statusLabel *widget.Label
    pool       *pgxpool.Pool
    tx         pgx.Tx
    txState    TxState
    cancel     context.CancelFunc
    configID   string
    configName string
    onTxChange func()
}
```

Terminal methods:

- `Content() fyne.CanvasObject` -- returns a `container.VSplit` with the editor
  in the top pane and the results grid plus status bar in the bottom pane.
- `RunQuery()` -- spawns a goroutine to execute SQL. If text is selected in the
  editor (`editor.SelectedText() != ""`), executes only the selection. Otherwise
  executes the full editor contents. This supports the common workflow of
  selecting SQL with Ctrl+Shift+arrow keys, then pressing Ctrl+Enter.
  Calls `db.ExecuteQuery(ctx, terminal.querier(), sql)`, then updates the
  results grid and status label on the main thread via Fyne's thread-safe
  `Refresh()`.
- `BeginTx(ctx context.Context) error` -- calls `pool.Begin(ctx)`, stores the
  returned `pgx.Tx`, sets state to `TxActive`, fires `onTxChange`.
- `CommitTx(ctx context.Context) error` -- commits the active transaction,
  clears `tx`, sets state to `TxNone`, fires `onTxChange`.
- `RollbackTx(ctx context.Context) error` -- rolls back the active transaction,
  clears `tx`, sets state to `TxNone`, fires `onTxChange`.
- `Close()` -- if a transaction is active, rolls it back. If a query is running,
  cancels its context.
- `querier() Querier` -- returns `tx` if `txState == TxActive`, otherwise
  returns `pool`.

### internal/ui/app.go

Owns the top-level `fyne.Window`. Composes the sidebar, toolbar, and terminal
tabs into a horizontal split layout. Manages application lifecycle: loads config
on startup, saves config on changes, calls `ConnectionManager.CloseAll()` on
window close.

Keyboard shortcuts:

- `Ctrl+Enter` -- run query in active terminal.
- `Ctrl+T` -- new terminal tab.
- `Ctrl+W` -- close active terminal tab.

### internal/ui/sidebar.go

Uses `widget.List` (not `widget.Tree`) to display saved connections. Each list
item shows the connection name. Right-click context menus provide edit, delete,
connect, and disconnect actions.

The decision to use `widget.List` over `widget.Tree` is deliberate:
`widget.Tree` does not support `SecondaryTappable` (right-click), which is
required for the context menu.

### internal/ui/terminal_tabs.go

Wraps `container.DocTabs`. Each tab corresponds to a `Terminal` instance. Tab
labels show the connection name and transaction indicator (e.g., "[TX]" when a
transaction is active). The `onTxChange` callback on each terminal triggers a
tab label refresh.

### internal/ui/toolbar.go

A horizontal row of action buttons: Run, Begin Transaction, Commit, Rollback,
Cancel. Buttons delegate to the active terminal's corresponding methods.

### internal/ui/connform.go

A modal dialog for adding or editing a connection. Contains entry fields for
name, host, port, user, password, database name, and SSL mode. On save, calls
`AppConfig.Add()` or `AppConfig.Update()` and refreshes the sidebar.

### internal/ui/resultsgrid.go

```go
type ResultsGrid struct {
    widget.Table
    columns []string
    rows    [][]string
}
```

- `SetData(columns []string, rows [][]string)` -- replaces the data and calls
  `Refresh()`.
- The `widget.Table` virtualizes rendering, so large result sets do not create
  one widget per cell.

### internal/db/schema.go

```go
type SchemaCache struct {
    Tables    []string            // table names
    Columns   map[string][]string // table_name -> column names
    Functions []string            // function names
    Keywords  []string            // SQL keywords (built-in)
}
```

- `NewSchemaCache() *SchemaCache` -- returns a cache pre-loaded with SQL keywords.
- `RefreshSchema(ctx context.Context, pool *pgxpool.Pool) error` -- queries
  `information_schema.tables`, `information_schema.columns`, and
  `information_schema.routines` to populate Tables, Columns, and Functions.
  Filters out `pg_catalog` and `information_schema` system schemas.
- `Suggest(prefix string, context string) []string` -- returns matching
  completions. If context indicates a column position (after a table alias or
  dot), returns column names for that table. Otherwise returns matching tables,
  functions, and keywords. Case-insensitive prefix matching.

### internal/ui/completer.go

```go
type Completer struct {
    schema  *db.SchemaCache
    editor  *widget.Entry
    popup   *widget.PopUpMenu
    canvas  fyne.Canvas
}
```

- `NewCompleter(editor *widget.Entry, canvas fyne.Canvas, schema *db.SchemaCache) *Completer`
- `OnTextChanged(text string)` -- extracts the word at the cursor position,
  queries `schema.Suggest()`, shows or hides the popup. Debounced to avoid
  excessive filtering on fast typing.
- `InsertCompletion(selected string)` -- replaces the current word fragment in
  the editor with the selected completion text.
- `Dismiss()` -- hides the popup.

The popup is a `widget.PopUpMenu` positioned relative to the editor widget.
Fyne does not expose exact cursor pixel coordinates in a multiline Entry, so
the popup is anchored to a fixed position near the top of the editor. Arrow
keys navigate within the popup; Enter/Tab inserts the selection; Esc dismisses.

---

## Data Flow

### Query Execution

1. User types SQL in the terminal editor.
2. User clicks Run or presses `Ctrl+Enter`.
3. `terminal.RunQuery()` creates a cancellable context and spawns a goroutine.
4. The goroutine calls `db.ExecuteQuery(ctx, terminal.querier(), sql)`.
5. `ExecuteQuery` determines whether the statement is a SELECT or DML/DDL.
6. For SELECT: calls `q.Query()`, iterates rows, stringifies values, collects
   column names. For DML/DDL: calls `q.Exec()`, reads the command tag.
7. Returns a `QueryResult` containing columns, rows, duration, message, and any
   error.
8. The goroutine calls `ResultsGrid.SetData()` and updates the status label
   with duration and row count (or error text).
9. `widget.Table` virtualizes rendering of the result set.

### Transaction Flow

1. **Begin**: `pool.Begin(ctx)` returns a `pgx.Tx`. Terminal stores it and sets
   `txState` to `TxActive`. Tab label updates to show "[TX]".
2. **Query routing**: `terminal.querier()` returns the `tx` while a transaction
   is active. All queries execute within the transaction.
3. **Commit**: `tx.Commit(ctx)` commits the transaction. Terminal clears `tx`
   and sets `txState` to `TxNone`. Tab label removes "[TX]".
4. **Rollback**: `tx.Rollback(ctx)` rolls back the transaction. Same cleanup as
   commit.
5. **Close with active transaction**: `terminal.Close()` automatically rolls
   back any active transaction before releasing resources.

### Connection Lifecycle

1. Application start: `AppConfig.Load()` reads the JSON config file. The
   sidebar populates from the loaded connections. No pools are created yet.
2. User clicks a connection in the sidebar (or opens a terminal for it):
   `ConnectionManager.Connect(ctx, cfg)` creates a `pgxpool.Pool` if one does
   not already exist.
3. The pool is shared across all terminals associated with that connection.
   Each terminal holds a reference to the pool.
4. Disconnect: `ConnectionManager.Disconnect(id)` closes the pool and removes
   it from the map. Any terminals using that pool should handle the closed state.
5. Application close: `ConnectionManager.CloseAll()` closes every open pool.

---

## Technical Decisions

1. **widget.List over widget.Tree for the sidebar.** Research confirmed that
   `widget.Tree` does not implement `SecondaryTappable`, making right-click
   context menus impossible. `widget.List` supports this interaction.

2. **Minimal Querier interface.** The two-method interface (`Exec` + `Query`) is
   implicitly satisfied by both `*pgxpool.Pool` and `pgx.Tx`. No wrappers or
   adapters are needed. This also makes testing straightforward -- a mock
   `Querier` can be used in unit tests without a real database.

3. **One goroutine per query with context.WithCancel.** Each `RunQuery()` call
   creates a cancellable context. The cancel function is stored on the terminal
   so that running queries can be interrupted via the Cancel toolbar button.

4. **Inline error display.** Query errors are shown in the status area below the
   results grid, not in modal dialogs. This keeps the workflow uninterrupted and
   allows the user to see the error alongside their SQL.

5. **fyne-cross for CI cross-compilation.** Fyne requires CGo. `fyne-cross`
   handles the cross-compilation toolchain in Docker containers. This must be
   validated before writing application code to avoid late surprises.

6. **Plaintext password storage.** Passwords are stored in the JSON config file
   without encryption. This is acceptable for an MVP/personal-use tool. Keyring
   integration (e.g., via `zalando/go-keyring`) is deferred to a future
   iteration.

---

## Testing Strategy

### Unit Tests

**internal/config (config_test.go)**
- Test `Load` with valid JSON, missing file, and malformed JSON.
- Test `Save` writes valid JSON that round-trips through `Load`.
- Test `Add` generates a UUID and appends to the list.
- Test `Update` replaces the correct entry; errors on unknown ID.
- Test `Remove` deletes the correct entry; errors on unknown ID.
- Test `FindByID` returns the correct config or an error.
- All tests use a temporary directory for file I/O.

**internal/db/query.go (query_test.go)**
- Define a mock `Querier` that returns canned `pgx.Rows` or `CommandTag`.
- Test `ExecuteQuery` with a SELECT statement: verify columns, rows, duration.
- Test `ExecuteQuery` with an INSERT/UPDATE/DELETE: verify message and row count.
- Test `ExecuteQuery` with a cancelled context: verify error propagation.
- Test `ExecuteQuery` with a query that returns an error: verify `QueryResult.Error`.

**internal/db/connection.go (connection_test.go)**
- Integration tests requiring a live PostgreSQL instance (optional in CI).
- Test `Connect` creates a pool; second call with same ID returns same pool.
- Test `Disconnect` closes pool; `IsConnected` returns false.
- Test `CloseAll` closes all pools.

### Manual Testing

UI verification follows a 9-step checklist (defined externally in REQUEST.md):
connection creation, connection editing, query execution, transaction
begin/commit/rollback, result grid scrolling, tab management, error display,
keyboard shortcuts, and application close/reopen config persistence.

### Estimated Test Volume

Approximately 400 lines of test code across `config_test.go` and
`query_test.go`.

---

## Risk Mitigations

| Risk | Mitigation |
|------|------------|
| Fyne API assumptions prove incorrect | Build a minimal spike (sidebar + editor + table) before full implementation to validate widget behavior. |
| Goroutine-to-UI-thread race conditions | Use Fyne's thread-safe `Refresh()` method exclusively for UI updates from goroutines. Establish this pattern in the first terminal implementation and reuse it. |
| widget.Tree right-click not supported | Use `widget.List` instead. Decision already validated in research phase. |
| CGo cross-compilation failures | Verify `fyne-cross` produces working binaries for target platforms before writing application code. |
| Plaintext passwords in config file | Document the limitation. Defer keyring integration to post-MVP. File permissions (0600) provide minimal protection. |

---

## Build and Run

```bash
# Development
go run .

# Build
go build -o helios .

# Cross-compile (Linux target from any platform)
fyne-cross linux -arch amd64

# Run tests
go test ./internal/...
```

---

## File Inventory

| File | Responsibility | Estimated LOC |
|------|---------------|---------------|
| main.go | Entry point, app initialization | ~30 |
| internal/config/config.go | Config types, JSON persistence | ~120 |
| internal/db/connection.go | Pool management, connect/disconnect | ~80 |
| internal/db/query.go | Query execution, result packaging | ~90 |
| internal/ui/app.go | Window, layout, lifecycle, shortcuts | ~150 |
| internal/ui/sidebar.go | Connection list, context menus | ~120 |
| internal/ui/terminal.go | Editor, results, transaction state | ~180 |
| internal/ui/terminal_tabs.go | Tab container management | ~80 |
| internal/ui/toolbar.go | Action buttons | ~60 |
| internal/ui/connform.go | Connection form dialog | ~100 |
| internal/ui/resultsgrid.go | Virtualized result table | ~80 |
| internal/db/schema.go | Schema introspection, metadata cache | ~120 |
| internal/ui/completer.go | Autocomplete popup, word extraction, insertion | ~200 |
| **Total** | | **~1,410** |
| internal/config/config_test.go | Config unit tests | ~200 |
| internal/db/query_test.go | Query execution unit tests | ~200 |
| internal/db/schema_test.go | Schema cache unit tests | ~100 |
| **Total with tests** | | **~1,910** |
