# Product Requirements Document: Helios

## Context and Why Now

Helios is a desktop PostgreSQL database GUI client built with Go and the Fyne v2 toolkit. It provides SQL terminals with transaction support, multi-database connectivity, and persisted connection configuration. The UI draws inspiration from TablePlus (sidebar + tabbed terminals) and DuckDB UI (inline results below the editor).

This is a **greenfield learning/portfolio project**, not a product venture. The research phase returned a CONDITIONAL GO recommendation: product viability is LOW (saturated market with pgAdmin, DBeaver, DataGrip, TablePlus, Postico, and psql all serving the space), but technical feasibility is HIGH and the project scope is appropriate for demonstrating Go concurrency, GUI development, database pooling, and state machine patterns.

The project is time-boxed and scope-disciplined. Now is the right time because the research phase has validated all critical Fyne APIs (`DocTabs.CloseIntercept`, goroutine-safe `Refresh()`, implicit Querier interface compatibility between `pgxpool.Pool` and `pgx.Tx`) and identified one required design change (use `widget.List` instead of `widget.Tree` for the sidebar, since `widget.Tree` does not support `SecondaryTappable`).

---

## Users and Jobs to Be Done

**Primary user**: The developer building this project (learning context).

**Secondary user persona**: A developer who needs a lightweight, single-binary PostgreSQL client for environments where Electron/Java-based tools are too heavy (e.g., ARM devices, constrained VMs).

| Job to Be Done | Current Alternative | Gap |
|---|---|---|
| Connect to PostgreSQL databases and save credentials for reuse | pgAdmin, TablePlus, DBeaver | No gap (learning exercise) |
| Write and execute SQL with inline results | Any existing GUI client | No gap (learning exercise) |
| Manage transactions per terminal tab | DataGrip, TablePlus | No gap (learning exercise) |
| Run a lightweight native GUI client on resource-constrained hardware | psql (CLI only) | Potential niche |

---

## Success Metrics

### Leading Indicators

- Application compiles cleanly: `go build ./...` produces a working binary.
- All 9 verification steps from REQUEST.md pass end-to-end.
- Code organized into `internal/config`, `internal/db`, and `internal/ui` with proper separation of concerns.
- Estimated scale met: ~12 source files, ~1,100-1,200 production LOC.

### Lagging Indicators

- Skills demonstrated: Go concurrency (goroutine-per-query with context cancellation), GUI development (Fyne v2), database connection pooling (pgxpool), transaction state machines.
- Clean architecture suitable for portfolio presentation.
- Project completed within time-box.

---

## Functional Requirements

### FR-1: Configuration Persistence

Save and load PostgreSQL connection configurations across application restarts.

**Acceptance Criteria:**
- Connections stored as JSON at `~/.config/helios/connections.json` (path resolved via `os.UserConfigDir()`).
- `ConnectionConfig` struct includes: ID (UUID), Name, Host, Port, User, Password, DBName, SSLMode.
- `AppConfig` supports `Add()`, `Update()`, `Remove()`, `FindByID()` operations.
- `Save()` is called after every mutation. `Load()` reads on startup.
- Config directory is created automatically if it does not exist.
- Application starts cleanly with no config file (first launch).

### FR-2: Connection Management

Manage PostgreSQL connection pools with one pool per saved configuration.

**Acceptance Criteria:**
- `ConnectionManager` maintains a `map[string]*pgxpool.Pool` keyed by config ID.
- Map access is mutex-protected.
- `Connect(ctx, cfg)` builds a DSN and creates a pool; idempotent (returns existing pool if already connected).
- `Disconnect(id)` closes the pool and removes it from the map.
- `Pool(id)` and `IsConnected(id)` accessors are available.
- `CloseAll()` shuts down every pool.

### FR-3: SQL Execution

Execute arbitrary SQL statements and return structured results.

**Acceptance Criteria:**
- `ExecuteQuery(ctx, querier, sql)` accepts a `Querier` interface (satisfied by both `pgxpool.Pool` and `pgx.Tx`).
- Returns `QueryResult` with: Columns (`[]string`), Rows (`[][]string`), RowCount, Message, Duration, Error.
- SELECT statements iterate rows and collect all values as strings; NULL values render as `<NULL>`.
- DML/DDL statements return the command tag as Message.
- Errors populate `QueryResult.Error` rather than panicking.
- Queries execute in a goroutine with `context.WithCancel` to avoid blocking the UI thread.

### FR-4: Transaction Support

Provide per-terminal transaction lifecycle control.

**Acceptance Criteria:**
- Each terminal maintains its own transaction state: None, Active.
- `BeginTx()` acquires a `pgx.Tx` from the connection pool and sets state to Active.
- `CommitTx()` commits the transaction and clears state.
- `RollbackTx()` rolls back the transaction and clears state.
- When a transaction is active, all queries in that terminal execute against the `pgx.Tx`.
- When no transaction is active, queries execute against the `pgxpool.Pool`.
- Closing a terminal with an active transaction triggers automatic rollback.
- Tab label displays `[TX]` suffix when a transaction is active.

### FR-5: Tabbed Terminal Interface

Support multiple independent terminal tabs, each with its own connection and transaction.

**Acceptance Criteria:**
- `DocTabs` container manages terminal tabs; tabs are closable.
- `NewTerminal(configID)` auto-connects if needed, creates a Terminal, adds a tab, and selects it.
- `ActiveTerminal()` returns the terminal for the currently selected tab.
- `CloseIntercept` calls `terminal.Close()` (which rolls back any open transaction) before removing the tab.
- Each terminal contains: a multi-line monospace SQL editor (top), a status label, and a results grid (bottom), arranged in a vertical split.

### FR-6: Results Grid

Display query results in a scrollable table.

**Acceptance Criteria:**
- Uses `widget.Table` with Length, CreateCell, and UpdateCell callbacks.
- Row 0 displays bold column headers from `QueryResult.Columns`.
- `SetData(result)` stores the result and refreshes the table.
- Column widths estimated from header name length (minimum 100, maximum 300).
- Status label above the grid shows row count and duration, or error message on failure.

### FR-7: Sidebar

Display saved connections with status indicators.

**Acceptance Criteria:**
- Uses `widget.List` (not `widget.Tree`) for the flat connection list.
- Each item shows the connection name and a connected/disconnected status indicator.
- Single-click opens a new terminal tab for that connection.
- Action buttons or context mechanism for: Connect, Disconnect, Edit, Delete.
- `Refresh()` rebuilds list data after config changes.

### FR-8: Connection Form

Provide Add and Edit dialogs for connection configurations.

**Acceptance Criteria:**
- `ShowConnectionDialog(window, existing, onSave)` using `dialog.ShowForm()`.
- Fields: Name, Host, Port (default 5432), User, Password (password entry), Database, SSL Mode (select).
- Edit mode pre-populates fields from the existing config.
- Required fields are validated before calling `onSave`.

### FR-9: Toolbar

Provide quick-access buttons for common terminal actions.

**Acceptance Criteria:**
- Buttons: Run (play icon), separator, Begin TX, Commit, Rollback, separator, New Terminal.
- Each action delegates to the `ActiveTerminal()` methods.
- Shows an error dialog if no active terminal exists or the transaction state is invalid for the action.

### FR-10: Menu and Keyboard Shortcuts

Provide standard application menus and keyboard shortcuts.

**Acceptance Criteria:**
- Menu: File > New Connection, New Terminal, Quit.
- Keyboard shortcuts: Ctrl+Enter (run query), Ctrl+T (new terminal), Ctrl+W (close tab).
- Shortcuts function regardless of focus position within the terminal.

### FR-11: Graceful Shutdown

Clean up all resources on application exit.

**Acceptance Criteria:**
- `window.SetOnClosed` triggers cleanup.
- All open transactions are rolled back.
- All connection pools are closed via `connMgr.CloseAll()`.
- No leaked goroutines or database connections after exit.

---

## Non-Functional Requirements

### Performance

- Query execution must not block the UI thread (goroutine with context cancellation).
- Results grid rendering must handle at least 1,000 rows without visible lag (Fyne `widget.Table` virtualizes rendering; data is held in memory).

### Concurrency and Safety

- Connection pool map must be mutex-protected for concurrent access.
- One goroutine per query with `context.WithCancel`; no channels or worker pools needed.
- Fyne `Refresh()` calls are safe from any goroutine (framework guarantee).

### Cross-Platform

- Must compile and run on Linux, macOS, and Windows via Fyne's cross-platform support.
- CGo dependency (Fyne requirement) must be verified for cross-compilation via `fyne-cross`.

### Security

- Passwords are stored in plaintext in the JSON config file (MVP limitation).
- A note should appear in the UI acknowledging this limitation.
- The config file should not be committed to version control (`.gitignore`).

### Observability

- Status label per terminal shows: row count and query duration on success, error message on failure.
- Tab labels reflect transaction state (`[TX]` suffix).
- Sidebar reflects connection status (connected/disconnected).

### Scale

- ~12 source files across 4 packages (`main`, `internal/config`, `internal/db`, `internal/ui`).
- ~1,100-1,200 production LOC, ~400 test LOC.
- 3 direct dependencies: `fyne.io/fyne/v2`, `github.com/jackc/pgx/v5`, `github.com/google/uuid`.

---

## Scope

### In Scope (MVP)

- PostgreSQL connection config CRUD with JSON persistence.
- SQL execution with inline results grid.
- Per-terminal transaction lifecycle (Begin/Commit/Rollback).
- Multiple independent terminal tabs.
- Sidebar with connection list and status.
- Toolbar, menus, and keyboard shortcuts.
- Graceful shutdown with resource cleanup.

### Out of Scope (MVP)

- Syntax highlighting in SQL editor (Fyne Entry limitation).
- Schema/table browsing in sidebar.
- Data export (CSV, JSON).
- OS keyring integration for password storage.
- Multi-statement result sets (only last result returned).
- Result set pagination or streaming.
- Query history or saved queries.
- Auto-complete or IntelliSense.
- Dark/light theme switching (use Fyne defaults).
- Support for databases other than PostgreSQL.

---

## Rollout Plan

### Phase 1: Foundation

Build config persistence, database connection layer, and query execution.

- Files: `go.mod`, `main.go`, `internal/config/config.go`, `internal/db/connection.go`, `internal/db/query.go`.
- Verification: Config saves/loads; pool connects/disconnects; queries return structured results.

### Phase 2: UI Shell

Build the results grid, terminal component, tab container, and top-level layout.

- Files: `internal/ui/resultsgrid.go`, `internal/ui/terminal.go`, `internal/ui/terminal_tabs.go`, `internal/ui/app.go`.
- Verification: Can open a terminal, execute SQL, and see results in the grid.

### Phase 3: Connection Management

Build the sidebar and connection form dialogs.

- Files: `internal/ui/sidebar.go`, `internal/ui/connform.go`.
- Verification: Can add/edit/remove connections from the sidebar; changes persist.

### Phase 4: Toolbar and Polish

Build the toolbar, wire up keyboard shortcuts, implement graceful shutdown, and handle edge cases.

- Files: `internal/ui/toolbar.go`, polish across all files.
- Verification: All 9 verification steps from REQUEST.md pass.

---

## Risks and Open Questions

### Risks

| Risk | Severity | Likelihood | Mitigation |
|---|---|---|---|
| Scope creep beyond MVP | High | High | Written scope boundaries; time-box enforcement |
| Fyne UX ceiling limits polish | Medium | Medium | Accept framework constraints; this is a learning project |
| CGo cross-compilation issues | Medium | High | Verify with `fyne-cross` before writing application code |
| Goroutine-UI race conditions | Medium | Medium | Use Fyne's thread-safe `Refresh()`; context cancellation on close |
| Plaintext passwords in public repo | High | Medium | `.gitignore` the config directory; document the limitation |
| Project abandoned incomplete | Medium | High | Phased delivery; each phase produces a usable increment |

### Open Questions

| # | Question | Impact |
|---|---|---|
| 1 | Platform priority: Linux-first or equal weight across all three? | Affects testing and CI setup |
| 2 | Connection failure behavior: show error inline in terminal vs. block tab creation? | Affects FR-5 implementation |
| 3 | Concurrent query execution policy: allow multiple in-flight queries per terminal? | Affects FR-3 goroutine management |
| 4 | Empty state on first launch: show a welcome message or just an empty sidebar? | UX detail |
| 5 | Config file corruption handling: fail silently with empty config or show error? | Affects FR-1 edge cases |
| 6 | Tab count limit: cap the number of open terminals? | Resource management |
