# Helios -- Implementation Roadmap

## Overview

| Field | Value |
|---|---|
| Feature | Helios -- PostgreSQL Database GUI Client |
| Type | Greenfield desktop application |
| Stack | Go + Fyne v2 + pgx/v5 |
| Estimated Scale | ~12 files, ~1,100--1,200 LOC + ~400 LOC tests |
| Framing | Learning/portfolio project (CONDITIONAL GO from research) |

This document is the master implementation plan for Helios. It breaks the project
into sequential phases with explicit dependencies, success criteria, and
validation gates. Each phase must pass its gate before the next phase begins.

---

## Phase 0: Prerequisites

Before writing any application code, verify that the development environment and
all assumed APIs are sound.

- [ ] Verify Go 1.21+ installed
- [ ] Verify C compiler available (CGo requirement for Fyne)
- [ ] Verify Fyne system dependencies installed (`libgl1-mesa-dev`, `xorg-dev` on Linux)
- [ ] Build Fyne API validation spike (~100 LOC) to confirm:
  - `DocTabs.CloseIntercept` works
  - `widget.List` renders correctly
  - `widget.Entry` supports monospace
  - Goroutine-to-UI `Refresh()` is safe
- [ ] Verify PostgreSQL test database is accessible
- [ ] Initialize `go.mod` with module path `helios`
- [ ] Install dependencies: `fyne.io/fyne/v2`, `github.com/jackc/pgx/v5`, `github.com/google/uuid`

**Validation Gate**: Spike compiles and runs, all four API behaviors confirmed,
PostgreSQL accepts a test connection.

---

## Phase 1: Foundation (config + db layer)

**Goal**: Core data layer that compiles and can be tested independently of UI.

### Tasks

| # | Task | File | Complexity | Dependencies |
|---|------|------|------------|--------------|
| 1.1 | Create `go.mod` + `main.go` scaffold | `main.go`, `go.mod` | Low | Prerequisites |
| 1.2 | Implement `ConnectionConfig` and `AppConfig` types | `internal/config/config.go` | Low | 1.1 |
| 1.3 | Implement `Load`/`Save` JSON persistence | `internal/config/config.go` | Low | 1.2 |
| 1.4 | Implement `Add`/`Update`/`Remove`/`FindByID` methods | `internal/config/config.go` | Low | 1.3 |
| 1.5 | Write config unit tests | `internal/config/config_test.go` | Medium | 1.4 |
| 1.6 | Define `Querier` interface and `QueryResult` type | `internal/db/query.go` | Low | 1.1 |
| 1.7 | Implement `ExecuteQuery` (SELECT path) | `internal/db/query.go` | Medium | 1.6 |
| 1.8 | Implement `ExecuteQuery` (DML/DDL path) | `internal/db/query.go` | Low | 1.7 |
| 1.9 | Write query unit tests with mock `Querier` | `internal/db/query_test.go` | Medium | 1.8 |
| 1.10 | Implement `ConnectionManager` struct | `internal/db/connection.go` | Medium | 1.6 |
| 1.11 | Implement `Connect`/`Disconnect`/`Pool`/`IsConnected`/`CloseAll` | `internal/db/connection.go` | Medium | 1.10 |

### Success Criteria

- [ ] `go build ./...` compiles
- [ ] Config tests pass: load, save, add, update, remove, find
- [ ] Query tests pass with mock querier
- [ ] `ConnectionManager` can connect to real PostgreSQL (manual test)

**Validation Gate**: All tests pass, `go vet ./...` clean.

---

## Phase 2: UI Shell (results grid, terminal, tabs, app layout)

**Goal**: Working application window with query execution and results display.

### Tasks

| # | Task | File | Complexity | Dependencies |
|---|------|------|------------|--------------|
| 2.1 | Implement `ResultsGrid` wrapping `widget.Table` | `internal/ui/resultsgrid.go` | Medium | Phase 1 |
| 2.2 | Implement bold headers (row 0) and column width estimation | `internal/ui/resultsgrid.go` | Low | 2.1 |
| 2.3 | Implement `SetData(result)` with refresh | `internal/ui/resultsgrid.go` | Low | 2.2 |
| 2.4 | Implement `Terminal` struct with editor, results, status | `internal/ui/terminal.go` | Medium | 2.3 |
| 2.5 | Implement `RunQuery()` with goroutine execution (runs selected text if any, else full editor) | `internal/ui/terminal.go` | High | 2.4 |
| 2.6 | Implement `BeginTx`/`CommitTx`/`RollbackTx` | `internal/ui/terminal.go` | Medium | 2.5 |
| 2.7 | Implement `Terminal.Close()` with auto-rollback | `internal/ui/terminal.go` | Low | 2.6 |
| 2.8 | Implement `querier()` helper (tx if active, else pool) | `internal/ui/terminal.go` | Low | 2.6 |
| 2.9 | Implement `TerminalTabs` wrapping `DocTabs` | `internal/ui/terminal_tabs.go` | Medium | 2.4 |
| 2.10 | Implement `NewTerminal` with auto-connect | `internal/ui/terminal_tabs.go` | Medium | 2.9 |
| 2.11 | Implement `CloseIntercept` callback | `internal/ui/terminal_tabs.go` | Low | 2.10 |
| 2.12 | Implement `ActiveTerminal()` | `internal/ui/terminal_tabs.go` | Low | 2.9 |
| 2.13 | Implement App layout (`Border` + `HSplit`) | `internal/ui/app.go` | Medium | 2.9 |
| 2.14 | Implement main menu (File > New Connection, New Terminal, Quit) | `internal/ui/app.go` | Low | 2.13 |
| 2.15 | Implement keyboard shortcuts (`Ctrl+Enter`, `Ctrl+T`, `Ctrl+W`) | `internal/ui/app.go` | Medium | 2.14 |
| 2.16 | Wire up `main.go` to create app, load config, build UI | `main.go` | Low | 2.15 |

### Success Criteria

- [ ] App window opens with HSplit layout
- [ ] Can create terminal tab (hardcoded connection for testing)
- [ ] Can type SQL and execute with `Ctrl+Enter`
- [ ] Can select a portion of SQL and execute only the selection with `Ctrl+Enter`
- [ ] Results appear in grid below editor
- [ ] Status label shows row count and duration
- [ ] Transaction Begin/Commit/Rollback work
- [ ] Tab label shows `[TX]` during active transaction
- [ ] Closing tab with active TX triggers rollback

**Validation Gate**: Manual verification of query execution and transaction
lifecycle.

---

## Phase 3: Connection Management (sidebar + dialogs)

**Goal**: Full connection CRUD via sidebar and dialogs.

### Tasks

| # | Task | File | Complexity | Dependencies |
|---|------|------|------------|--------------|
| 3.1 | Implement `ShowConnectionDialog` with form fields | `internal/ui/connform.go` | Medium | Phase 2 |
| 3.2 | Implement Edit mode (pre-populate from existing config) | `internal/ui/connform.go` | Low | 3.1 |
| 3.3 | Implement field validation (required fields) | `internal/ui/connform.go` | Low | 3.2 |
| 3.4 | Implement Sidebar with `widget.List` | `internal/ui/sidebar.go` | Medium | Phase 2 |
| 3.5 | Implement connection status indicators | `internal/ui/sidebar.go` | Low | 3.4 |
| 3.6 | Implement single-click to open terminal | `internal/ui/sidebar.go` | Low | 3.5 |
| 3.7 | Implement context actions (Connect, Disconnect, Edit, Delete) | `internal/ui/sidebar.go` | Medium | 3.6 |
| 3.8 | Implement `Sidebar.Refresh()` after config changes | `internal/ui/sidebar.go` | Low | 3.7 |
| 3.9 | Wire sidebar and connection form into `app.go` | `internal/ui/app.go` | Low | 3.8 |

### Success Criteria

- [ ] Can add new connection via dialog
- [ ] Connection appears in sidebar
- [ ] Can edit existing connection
- [ ] Can delete connection
- [ ] Clicking connection opens terminal tab
- [ ] Status indicators show connected/disconnected
- [ ] Connections persist across app restart

**Validation Gate**: Full connection lifecycle works end-to-end.

---

## Phase 4: Toolbar + Polish

**Goal**: Complete UI with toolbar actions and polished error handling.

### Tasks

| # | Task | File | Complexity | Dependencies |
|---|------|------|------------|--------------|
| 4.1 | Implement Toolbar with Run, Begin TX, Commit, Rollback, New Terminal | `internal/ui/toolbar.go` | Medium | Phase 3 |
| 4.2 | Wire toolbar actions to `ActiveTerminal()` methods | `internal/ui/toolbar.go` | Low | 4.1 |
| 4.3 | Add error dialogs for invalid toolbar state | `internal/ui/toolbar.go` | Low | 4.2 |
| 4.4 | Implement graceful shutdown (rollback open txs, close pools) | `internal/ui/app.go` | Medium | 4.1 |
| 4.5 | Add error handling for connection failures | `internal/ui/terminal.go` | Low | 4.1 |
| 4.6 | Polish status label (row count, duration, error messages) | `internal/ui/terminal.go` | Low | 4.5 |
| 4.7 | Final integration testing against 9-step verification checklist | -- | Medium | 4.6 |

### Success Criteria

- [ ] All toolbar buttons work correctly
- [ ] Error states handled gracefully (no panics)
- [ ] App shutdown cleans up all resources
- [ ] All 9 verification steps from REQUEST.md pass

**Validation Gate**: Full 9-step verification checklist passes.

---

## Phase 5: SQL Autocomplete

**Goal**: Context-aware autocomplete for table names, column names, and SQL
functions in the editor. Popup appears as the user types and can be navigated
with arrow keys + Enter/Tab.

### Design Notes

Fyne's `CompletionEntry` (fyne-x) is single-line only. For our multiline SQL
editor we need a custom approach: intercept keystrokes on the `widget.Entry`,
extract the word being typed, show a `widget.PopUpMenu` near the editor with
filtered suggestions, and insert the selected completion into the editor text.

Schema metadata (tables, columns, functions) is fetched from the connected
database via `pg_catalog` queries and cached per connection.

### Tasks

| # | Task | File | Complexity | Dependencies |
|---|------|------|------------|--------------|
| 5.1 | Implement `SchemaCache` struct with tables, columns, functions maps | `internal/db/schema.go` | Medium | Phase 1 |
| 5.2 | Implement `RefreshSchema(ctx, pool)` querying `pg_catalog` | `internal/db/schema.go` | Medium | 5.1 |
| 5.3 | Add built-in SQL keyword list (SELECT, FROM, WHERE, JOIN, etc.) | `internal/db/schema.go` | Low | 5.1 |
| 5.4 | Write schema cache tests | `internal/db/schema_test.go` | Medium | 5.2 |
| 5.5 | Implement `Completer` struct with fuzzy prefix matching | `internal/ui/completer.go` | Medium | 5.2 |
| 5.6 | Extract current word at cursor from editor text | `internal/ui/completer.go` | Medium | 5.5 |
| 5.7 | Implement popup menu showing filtered suggestions | `internal/ui/completer.go` | High | 5.6 |
| 5.8 | Implement completion insertion (replace current word with selection) | `internal/ui/completer.go` | Medium | 5.7 |
| 5.9 | Wire completer into Terminal (trigger on typing, dismiss on Esc/space) | `internal/ui/terminal.go` | Medium | 5.8 |
| 5.10 | Trigger `RefreshSchema` on connection open; cache per config ID | `internal/ui/terminal_tabs.go` | Low | 5.9 |

### Schema Queries

```sql
-- Tables
SELECT table_name FROM information_schema.tables
WHERE table_schema NOT IN ('pg_catalog', 'information_schema');

-- Columns (keyed by table)
SELECT table_name, column_name FROM information_schema.columns
WHERE table_schema NOT IN ('pg_catalog', 'information_schema');

-- Functions
SELECT routine_name FROM information_schema.routines
WHERE routine_schema NOT IN ('pg_catalog', 'information_schema');
```

### Success Criteria

- [ ] Typing a table name prefix shows matching table suggestions
- [ ] After `SELECT ... FROM tablename.`, column suggestions appear
- [ ] SQL keywords (SELECT, WHERE, JOIN, etc.) appear in suggestions
- [ ] Arrow keys navigate the popup, Enter/Tab inserts the selection
- [ ] Esc dismisses the popup
- [ ] Schema cache refreshes when connecting to a new database
- [ ] Autocomplete does not block typing (async filtering)

**Validation Gate**: Autocomplete works for tables, columns, keywords, and
functions against a real PostgreSQL database.

---

## Task Summary

| Phase | Description | Tasks |
|-------|-------------|-------|
| 0 | Prerequisites | 7 items |
| 1 | Foundation (config + db) | 11 tasks |
| 2 | UI Shell (results grid, terminal, tabs, app layout) | 16 tasks |
| 3 | Connection Management (sidebar + dialogs) | 9 tasks |
| 4 | Toolbar + Polish | 7 tasks |
| 5 | SQL Autocomplete | 10 tasks |
| **Total** | | **53 tasks + 7 prerequisites** |

---

## Dependency Chart

```
Phase 0 (Prerequisites)
    └── Phase 1 (Foundation: config + db)
        ├── 1.1-1.5: Config package (independent)
        ├── 1.6-1.9: Query package (independent)
        └── 1.10-1.11: Connection manager (depends on 1.6)
            └── Phase 2 (UI Shell)
                ├── 2.1-2.3: ResultsGrid (independent)
                ├── 2.4-2.8: Terminal (depends on ResultsGrid)
                ├── 2.9-2.12: TerminalTabs (depends on Terminal)
                └── 2.13-2.16: App layout (depends on all above)
                    └── Phase 3 (Connection Management)
                        ├── 3.1-3.3: ConnForm (independent)
                        ├── 3.4-3.8: Sidebar (independent)
                        └── 3.9: Wire into app (depends on both)
                            └── Phase 4 (Toolbar + Polish)
                                └── Phase 5 (SQL Autocomplete)
                                    ├── 5.1-5.4: SchemaCache + tests (independent)
                                    ├── 5.5-5.8: Completer widget (depends on SchemaCache)
                                    └── 5.9-5.10: Wire into terminal (depends on Completer)
```

---

## Parallelization Opportunities

- **Phase 1**: Config (1.1--1.5) and Query (1.6--1.9) can be developed in
  parallel. They share no types or imports.
- **Phase 2**: ResultsGrid (2.1--2.3) is independent until Terminal needs it at
  2.4.
- **Phase 3**: ConnForm (3.1--3.3) and Sidebar (3.4--3.8) can be developed in
  parallel. Both wire into `app.go` at 3.9.
- **Phase 4**: Mostly sequential. Integration work depends on all prior phases
  being complete.
- **Phase 5**: SchemaCache (5.1--5.4) can start once Phase 1 is done. Completer
  (5.5--5.8) depends on SchemaCache. Wiring (5.9--5.10) depends on Phase 4.

---

## Risk Mitigations

| Risk | Mitigation | Phase |
|------|------------|-------|
| Fyne API does not behave as expected | Spike validation in Phase 0 | 0 |
| Goroutine-UI race conditions | Establish `Refresh()` pattern in Phase 2, task 2.5 | 2 |
| `widget.List` context actions awkward | Fall back to toolbar buttons for connection actions | 3 |
| CGo build fails on target platform | Verify in Phase 0 before writing app code | 0 |
| Transaction state machine bugs | Test Begin/Commit/Rollback exhaustively in Phase 2 | 2 |
| Fyne PopUpMenu positioning in multiline editor | Fall back to fixed position below editor if cursor position unavailable | 5 |
| Schema introspection slow on large databases | Cache aggressively, refresh async, limit result count | 5 |

---

## Verification Checklist

These steps come from REQUEST.md and serve as the final acceptance test.

1. [ ] `go build ./...` compiles without errors
2. [ ] Add a Postgres connection via sidebar or File > New Connection
3. [ ] Connection persists after app restart
4. [ ] Click connection in sidebar to open terminal tab
5. [ ] `SELECT version();` returns result in grid
6. [ ] Begin TX, `CREATE TABLE`, Rollback -- table does not exist
7. [ ] Multiple terminals to different databases work independently
8. [ ] Close terminal with active transaction -- rollback occurs
9. [ ] Close app -- all connections shut down cleanly
10. [ ] Select partial SQL, `Ctrl+Enter` -- only selection executes
11. [ ] Type a table name prefix -- autocomplete popup shows matching tables
12. [ ] Select autocomplete suggestion with Enter -- text inserted at cursor
