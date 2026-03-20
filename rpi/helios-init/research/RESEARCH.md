# Research Report: Helios — PostgreSQL Database GUI Client

## Overview

| Field | Value |
|---|---|
| Feature Name | Helios — PostgreSQL Database GUI Client |
| Feature Type | Desktop Application (GUI) — Greenfield New Project |
| Target Component | Entire application — 12 source files across 4 packages (`main`, `internal/config`, `internal/db`, `internal/ui`) |
| Complexity | Complex |
| Decision | CONDITIONAL GO |
| Confidence | High |
| Framing | Learning/portfolio project only, not a product venture |

---

## Phase 1: Requirements Parsing

### Goals

1. Cross-platform desktop GUI for connecting to and querying PostgreSQL databases.
2. Multiple simultaneous database connections.
3. SQL authoring and execution with inline results.
4. Transaction lifecycle control (Begin/Commit/Rollback) per terminal.
5. Persistent connection configurations via JSON.
6. UI inspired by TablePlus (sidebar + tabs) and DuckDB UI (inline results).

### Functional Requirements (Must Have)

- **Config persistence** to `~/.config/helios/connections.json`.
- **Connection management** with `pgxpool.Pool` per config, mutex-protected.
- **SQL execution** via `ExecuteQuery` returning `QueryResult` with columns, rows (as strings), count, duration, and error.
- **NULL rendering** as `<NULL>`; DML/DDL returns command tag.
- **Per-terminal transaction control**: Begin/Commit/Rollback with automatic rollback on close.
- **Tabbed terminal interface** (`DocTabs`), each with its own connection and transaction.
- **Multi-line monospace SQL editor**.
- **Results grid** (`widget.Table`) with bold headers and estimated column widths.
- **Status bar** per terminal.
- **Sidebar** with connection list, status indicators, single-click to open terminal, right-click context menu.
- **Connection form dialog** supporting Add and Edit modes.
- **Toolbar**: Run, Begin TX, Commit, Rollback, New Terminal.
- **Menu**: File > New Connection, New Terminal, Quit.
- **Keyboard shortcuts**: Ctrl+Enter (run), Ctrl+T (new terminal), Ctrl+W (close terminal).
- **Tab labels** show `[TX]` when a transaction is active.
- **Graceful shutdown** of all connections on exit.

### Non-Functional Requirements

- Query execution in goroutines (non-blocking UI).
- Mutex-protected connection pool map.
- Cross-platform support: Linux, macOS, Windows via Fyne.
- Passwords stored in plaintext (MVP caveat).
- `internal/` package convention enforced.

### Open Questions

| # | Question |
|---|---|
| 1 | Target platform priority — Linux-first or equal weight across all three? |
| 2 | Querier interface compatibility between `pgxpool.Pool` and `pgx.Tx`. |
| 3 | `DocTabs.CloseIntercept` API verification. |
| 4 | Connection failure behavior — error inline in terminal vs. block tab creation? |
| 5 | Concurrent query execution policy per terminal. |
| 6 | SSL mode valid values. |
| 7 | Empty state on first launch. |
| 8 | Config file corruption handling. |
| 9 | Password plaintext warning in UI. |
| 10 | Tab count limit. |

---

## Phase 2: Product Analysis

**Product Viability Score: LOW**

### User Value

Low to Medium. Core workflows (connect, query, view results) are table-stakes already delivered by every competitor. Per-terminal transaction management is useful but not a large enough differentiator to justify adoption over established tools.

### Competitive Landscape

| Tool | Price | Key Advantage |
|---|---|---|
| pgAdmin | Free/OSS | Feature-complete, official PostgreSQL tool |
| DBeaver | Free/OSS | Multi-database, huge plugin ecosystem |
| DataGrip | Paid ($100+/yr) | Best-in-class IDE integration |
| TablePlus | Paid ($89) | Fast native UI, excellent UX |
| Postico | Paid ($50) | Mac-native, simple and beautiful |
| psql | Free/built-in | Zero-install, scriptable |

### Key Concerns

- No identified wedge use case or differentiation from existing tools.
- No articulation of a target user segment not already served.
- Fyne introduces a ceiling on UX quality relative to native or web-based competitors.
- No user acquisition strategy.

### Potential Niches

- Single-binary/zero-install deployment for constrained environments.
- ARM/Raspberry Pi optimization (lightweight where Electron/Java tools struggle).
- Opinionated workflow tool (migration testing, query auditing).

### Verdict

Proceed only as a personal or learning project. The market is saturated and there is no clear differentiation path.

---

## Phase 2.5: Technical Discovery

- **Greenfield project**: No existing code in the repository.
- **No conflicts**: Nothing to break.
- **No reusable components**: Everything built from scratch.
- **Repository contents**: Only `.git`, `.claude/`, `constitution.md` (empty), and `rpi/` folder.

---

## Phase 3: Technical Feasibility

**Technical Feasibility Score: HIGH**
**Complexity: MEDIUM**

### API Verifications

| API | Status | Notes |
|---|---|---|
| `DocTabs.CloseIntercept` | Confirmed | Present in Fyne v2. |
| Querier interface | Confirmed | Both `pgxpool.Pool` and `pgx.Tx` satisfy `Query` and `Exec` implicitly — no adapter needed. |
| Goroutine-to-UI safety | Confirmed | Fyne's `Refresh()` is safe from any goroutine. |
| `widget.Tree` right-click | Not supported | `widget.Tree` does not implement `SecondaryTappable`. |

### Technical Recommendations

1. **Use `widget.List` instead of `widget.Tree`** for the sidebar. Simpler for flat data, easier to attach action buttons.
2. **Minimal Querier interface** with just `Query` and `Exec`. Both `pgxpool.Pool` and `pgx.Tx` satisfy it implicitly.
3. **One goroutine per query** with `context.WithCancel`. No channels, no worker pools.
4. **`TerminalSession` struct** owns the pool reference, optional transaction, and a `TxState` enum.
5. **Inline error display** instead of modal dialogs.
6. **Use `fyne-cross`** (Docker-based) for CI cross-compilation.

### Estimated Scale

| Metric | Estimate |
|---|---|
| Source files | ~12 |
| Production LOC | ~1,100–1,200 |
| Test LOC | ~400 |
| Direct dependencies | 3 (Fyne, pgx, uuid) |

### Build vs. Buy

| Component | Decision |
|---|---|
| GUI toolkit | Buy (Fyne v2) |
| DB driver | Buy (pgx/v5) |
| UUID generation | Buy (google/uuid) |
| Config management | Build |
| Connection manager | Build |
| Transaction state machine | Build |
| Results renderer | Build |
| Syntax highlighting | Defer |

---

## Phase 4: Strategic Assessment

**Recommendation: CONDITIONAL GO — as Learning/Portfolio Project ONLY**
**Confidence: HIGH**

### Strategic Rationale

- **As a product**: LOW viability. No differentiation, saturated market, framework ceiling, no acquisition strategy.
- **As a learning project**: MEDIUM viability. Appropriate scope, genuine engineering challenges (concurrency, GUI state management, database pooling), transferable skills.

### Conditions for Proceeding

| # | Condition | Blocking |
|---|---|---|
| 1 | Fyne API Verification — build a ~100-line spike to validate all assumed APIs. | Yes |
| 2 | Time-Box — hard stop at 6 weeks. | Yes |
| 3 | Scope Discipline — written list of 5 features to build and 20 to skip. | Yes |
| 4 | CGo Build Verification — verify cross-compilation works before writing app code. | Yes |
| 5 | Constitution Document — define coding standards, error handling patterns, testing requirements. | No |

### Risk Register

| Risk | Severity | Likelihood |
|---|---|---|
| Fyne API assumptions wrong | High | Medium |
| Scope creep | High | High |
| CGo build complications | Medium | High |
| Goroutine-UI race conditions | Medium | Medium |
| Project abandoned incomplete | Medium | High |
| Plaintext passwords in public repo | High | Medium |
| Fyne UX ceiling | Medium | Medium |

### Alternatives Considered

| Alternative | Tradeoff |
|---|---|
| Wails (Go + web) or Tauri (Rust + web) | Better UI capabilities but different learning goals. |
| Use pgAdmin or DBeaver | Correct if goal is productivity, not learning. |
| Contribute PRs to existing OSS projects | Comparable learning, better portfolio signal. |
| Defer | If Go skills not yet strong, build something simpler first. |

---

## Summary

| Dimension | Score |
|---|---|
| Product Viability | Low |
| Technical Feasibility | High |
| Overall Assessment | Medium |

### Top 3 Risks

1. Scope creep turning a 6-week project into a 6-month project.
2. Fyne framework ceiling limiting future capabilities and UX quality.
3. Market saturation rendering the tool redundant for external users.

### Next Steps (if Proceeding)

1. Build Fyne API spike to validate assumptions.
2. Define scope boundaries and time-box.
3. Verify CGo cross-compilation.
4. Fill in `constitution.md` with project standards.
5. Proceed to planning phase.
