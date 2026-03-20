# Helios UX Design Document

## Product Summary

Helios is a desktop PostgreSQL database GUI client built with Go and Fyne v2. It provides a sidebar-plus-tabbed-terminal layout inspired by TablePlus and DuckDB UI. Each terminal tab maintains independent SQL editing, query execution, and transaction state against a configured PostgreSQL connection.

---

## User Stories and Acceptance Criteria

### US-1: Add a New Connection

**As a** database user, **I want to** save a PostgreSQL connection configuration **so that** I can reuse it across sessions without re-entering credentials.

**Acceptance Criteria:**
- A connection form dialog is accessible from File > New Connection and from the sidebar context.
- The form includes fields: Name, Host, Port (default 5432), User, Password (masked), Database, SSL Mode.
- Name, Host, User, and Database are required. Submitting with any blank shows inline validation.
- On save, the connection appears in the sidebar as "disconnected."
- The configuration persists to `~/.config/helios/connections.json` and survives app restart.

### US-2: Connect and Run a Query

**As a** database user, **I want to** connect to a database and run SQL **so that** I can view query results inline.

**Acceptance Criteria:**
- Clicking a sidebar connection opens a new terminal tab and initiates connection.
- The SQL editor accepts multi-line input in a monospace font.
- Pressing Ctrl+Enter or clicking the Run toolbar button executes the editor contents.
- During execution, the status label reads "Running..."
- On success, the results grid shows column headers (bold) and rows. The status label shows row count and duration.
- On error, the status label and results area display the error message.

### US-3: Manage Transactions

**As a** database user, **I want to** control transaction boundaries per terminal **so that** I can test DML safely before committing.

**Acceptance Criteria:**
- Begin TX starts a transaction. The tab label appends "[TX]".
- Commit and Rollback resolve the transaction. The tab label reverts.
- Commit/Rollback buttons are inactive when no transaction is open.
- Closing a tab with an active transaction triggers an automatic rollback.

### US-4: Work Across Multiple Tabs

**As a** database user, **I want to** have multiple terminal tabs open **so that** I can query different databases or run parallel investigations.

**Acceptance Criteria:**
- Each sidebar click opens a new tab for that connection.
- New Terminal toolbar button or Ctrl+T opens a tab for the current connection.
- Ctrl+W closes the active tab.
- Each tab maintains independent editor content, results, and transaction state.
- Switching tabs restores the last visible state of that tab.

### US-5: Edit and Delete Connections

**As a** database user, **I want to** modify or remove saved connections **so that** I can keep my connection list current.

**Acceptance Criteria:**
- Right-clicking a sidebar connection (or selecting it and using a toolbar/menu action) shows options: Connect, Disconnect, Edit, Delete.
- Edit opens the connection form pre-populated with existing values.
- Delete prompts a confirmation dialog before removing.
- Disconnect closes the pool but retains the configuration.
- Changes persist to the config file immediately.

---

## Layout and Component Map

### Primary Layout

```
+------------------------------------------------------------------+
|  Menu Bar: File | Edit                                            |
+------------------------------------------------------------------+
|  Toolbar: [Run] | [Begin TX] [Commit] [Rollback] | [New Terminal]|
+------------------------------------------------------------------+
|                  |                                                 |
|  Sidebar (20%)   |  Terminal Tabs (80%)                           |
|  (widget.List)   |  (container.DocTabs)                           |
|                  |                                                 |
|  [Connection A]  |  +--Tab: mydb--------+--Tab: other [TX]--+    |
|   * connected    |  |                                        |    |
|  [Connection B]  |  |  SQL Editor (top, VSplit)              |    |
|   * disconnected |  |  +----------------------------------+  |    |
|                  |  |  | SELECT * FROM users;              |  |    |
|                  |  |  |                                   |  |    |
|                  |  |  +----------------------------------+  |    |
|                  |  |                                        |    |
|                  |  |  Results Panel (bottom, VSplit)        |    |
|                  |  |  +----------------------------------+  |    |
|                  |  |  | Status: 42 rows returned (12ms)  |  |    |
|                  |  |  +----------------------------------+  |    |
|                  |  |  | id | name  | email    | created  |  |    |
|                  |  |  | 1  | alice | a@b.com  | 2026-01  |  |    |
|                  |  |  | 2  | bob   | b@b.com  | 2026-01  |  |    |
|                  |  |  +----------------------------------+  |    |
|                  |  +----------------------------------------+    |
+------------------------------------------------------------------+
```

### Container Hierarchy

```
Border
  top:    VBox [ MenuBar, Toolbar ]
  center: HSplit (offset 0.2)
            left:  Sidebar (widget.List)
            right: DocTabs
                     each tab -> VSplit
                                   top:    widget.Entry (MultiLine, monospace)
                                   bottom: VBox [ StatusLabel, widget.Table ]
```

### Fyne Widget Mapping

| UI Element         | Fyne Widget / Container         | Notes                                      |
|--------------------|---------------------------------|--------------------------------------------|
| Overall layout     | container.NewBorder             | Menu + toolbar at top, HSplit in center     |
| Sidebar            | widget.List                     | Flat list; widget.Tree lacks right-click    |
| Terminal tabs      | container.DocTabs               | Closable tabs via CloseIntercept            |
| SQL editor         | widget.Entry (MultiLine)        | Monospace font; no syntax highlighting      |
| Results grid       | widget.Table                    | Virtualized rendering; row 0 = bold headers |
| Status label       | widget.Label                    | Below editor, above grid                    |
| Toolbar            | widget.Toolbar                  | Icon buttons with separators                |
| Connection dialog  | dialog.ShowForm                 | Modal; FormItems for each field             |
| Context menu       | widget.PopUpMenu                | Triggered on sidebar right-click            |
| Confirmation       | dialog.ShowConfirm              | Used for delete confirmation                |

---

## Flow Descriptions

### Flow 1: First Launch / Add Connection

**Precondition:** No config file exists or config file has zero connections.

```
[App Opens]
    |
    v
+--Empty Sidebar--+--Empty Tab Area--------------------------+
|                  |                                           |
|  "No connections"|  "Open a connection to get started."      |
|                  |                                           |
+------------------+-------------------------------------------+
    |
    User: File > New Connection (or right-clicks sidebar area)
    |
    v
+--Connection Form Dialog--+
|  Name:     [___________] |
|  Host:     [___________] |
|  Port:     [5432_______] |
|  User:     [___________] |
|  Password: [***________] |
|  Database: [___________] |
|  SSL Mode: [disable  v ] |
|                          |
|  [Cancel]       [Save]   |
+--------------------------+
    |
    User fills fields, clicks Save
    |
    v
Sidebar shows: "MyDB (disconnected)"
Config saved to ~/.config/helios/connections.json
```

**States:**
- **Empty state:** Sidebar shows placeholder text. Tab area shows onboarding hint.
- **Validation error:** Required field labels turn red. Save button remains enabled but submission shows inline error.
- **Save success:** Dialog closes. Sidebar refreshes with new entry.

### Flow 2: Connect and Query

**Precondition:** At least one connection exists in config.

```
User clicks "MyDB" in sidebar
    |
    v
New tab opens: "MyDB"
Status: "Connecting..."
    |
    Connection succeeds
    |
    v
Sidebar: "MyDB (connected)"
Status: "Connected."
Editor: focused, empty, cursor blinking
    |
    User types: SELECT * FROM users;
    User presses Ctrl+Enter or clicks [Run]
    |
    v
Status: "Running..."
Editor: read-only during execution (prevents edits to in-flight query)
    |
    Query returns
    |
    v
Status: "42 rows returned (12ms)"
Grid: populated with columns and rows
Editor: editable again, text preserved
```

**States:**
- **Connecting:** Status reads "Connecting..." Tab is open but editor is disabled.
- **Connection failure:** Status reads "Connection failed: <error>". Editor remains disabled. Sidebar stays "disconnected." User can retry by clicking the connection again or closing the tab.
- **Running:** Status reads "Running..." Editor is temporarily read-only.
- **Success (SELECT):** Status shows row count and duration. Grid shows data.
- **Success (DML/DDL):** Status shows command tag (e.g., "INSERT 0 1"). Grid is empty or hidden.
- **Query error:** Status shows error in red-tinted text. Grid shows single-cell error detail.
- **Empty result:** Status shows "0 rows returned (Xms)". Grid shows column headers only.
- **NULL values:** Rendered as `<NULL>` in grid cells.

### Flow 3: Transaction Management

**Precondition:** Terminal tab is open and connected.

```
User clicks [Begin TX]
    |
    v
Tab label: "MyDB [TX]"
Status: "Transaction started."
[Commit] and [Rollback] become active
[Begin TX] becomes inactive
    |
    User runs DML: INSERT INTO users (name) VALUES ('carol');
    |
    v
Status: "INSERT 0 1"
    |
    User clicks [Commit]
    |
    v
Tab label: "MyDB"
Status: "Transaction committed."
[Begin TX] becomes active
[Commit] and [Rollback] become inactive
```

**Alternate path -- Rollback:**
```
    User clicks [Rollback]
    |
    v
Tab label: "MyDB"
Status: "Transaction rolled back."
```

**Alternate path -- Close tab with active TX:**
```
    User closes tab (Ctrl+W or click X)
    |
    v
Transaction auto-rolled back silently.
Connection pool remains open (shared resource).
Tab removed.
```

**States:**
- **No TX:** Begin TX active. Commit/Rollback inactive.
- **TX active:** Begin TX inactive. Commit/Rollback active. Tab label shows "[TX]".
- **Invalid operation:** Clicking Commit/Rollback with no TX shows inline status error: "No active transaction." Clicking Begin TX while TX is active shows: "Transaction already in progress."

### Flow 4: Multi-Tab Workflow

```
User clicks "ConnA" in sidebar  -->  Tab 1: "ConnA" opens
User clicks "ConnB" in sidebar  -->  Tab 2: "ConnB" opens
User clicks Tab 1               -->  Tab 1 editor and results restore
User presses Ctrl+T             -->  Tab 3: new tab for current connection
User presses Ctrl+W             -->  Active tab closes (auto-rollback if TX)
```

**States:**
- **Single tab:** Close removes tab. Tab area shows empty/onboarding state.
- **Multiple tabs:** Toolbar actions (Run, Begin TX, etc.) always target the active tab.
- **Tab switching:** No data loss. Each tab retains its editor text, results, scroll position, and transaction state independently.

### Flow 5: Connection Management

```
User right-clicks "ConnB" in sidebar
    |
    v
+--Context Menu-------+
|  Connect             |
|  Disconnect          |
|  Edit                |
|  Delete              |
+----------------------+
```

**Connect:** Opens a new terminal tab and connects (same as single-click).
**Disconnect:** Closes the connection pool. All tabs using this connection show status: "Disconnected." Tabs remain open but editor becomes disabled. Sidebar shows "disconnected."
**Edit:** Opens connection form pre-filled with current values. On save, updates config. If connected, disconnects and reconnects with new settings.
**Delete:** Shows confirmation dialog: "Delete connection ConnB? This cannot be undone." On confirm, disconnects, closes associated tabs, removes from config and sidebar.

---

## Error States

| Scenario                    | Display Location        | Message Pattern                                      | Recovery Action                    |
|-----------------------------|-------------------------|------------------------------------------------------|------------------------------------|
| Connection failure          | Terminal status label    | "Connection failed: <pg error>"                      | Fix config or retry                |
| Query syntax error          | Terminal status label    | "ERROR: <pg error message>"                          | Edit SQL, re-run                   |
| Query timeout               | Terminal status label    | "Query cancelled: context deadline exceeded"         | Optimize query, re-run             |
| Commit with no TX           | Terminal status label    | "No active transaction."                             | Begin TX first                     |
| Begin TX while TX active    | Terminal status label    | "Transaction already in progress."                   | Commit or rollback first           |
| Config file missing         | Silent                  | App starts with empty config, creates file on save   | Normal first-launch behavior       |
| Config file corrupt         | Warning dialog on start | "Configuration file could not be read. Starting fresh." | App creates new empty config    |
| Disconnect while TX active  | Auto-rollback           | Status: "Transaction rolled back. Disconnected."     | Reconnect to continue              |

---

## Keyboard Shortcuts

| Shortcut    | Action                         | Scope          |
|-------------|--------------------------------|----------------|
| Ctrl+Enter  | Run query in active terminal   | Active tab     |
| Ctrl+T      | Open new terminal tab          | Global         |
| Ctrl+W      | Close active terminal tab      | Active tab     |

---

## Accessibility Notes

### Keyboard Navigation

- All toolbar actions are reachable via keyboard tab order.
- Sidebar list supports arrow-key navigation and Enter to select.
- DocTabs support standard tab switching (implementation depends on Fyne defaults).
- Ctrl+Enter as the primary query execution shortcut avoids conflict with Enter for newlines in the multi-line editor.

### Visual Design

- **Monospace font** in the SQL editor for character alignment and code readability.
- **Bold column headers** in the results grid (row 0) to distinguish headers from data rows.
- **Status indicators** for connected (visible text label) and disconnected states in the sidebar. Text-based, not color-only, to support colorblind users.
- **Masked password field** in the connection form using Fyne's password entry widget.
- **Sufficient contrast** for status messages. Error messages use text labels rather than relying solely on color.

### Screen Reader Considerations

- Fyne v2 has limited screen reader support. Where possible, widgets use descriptive labels.
- Toolbar buttons include tooltip text describing their action.
- Dialog form fields have associated label text.

### Constraints (Fyne v2)

- No syntax highlighting in the SQL editor. This is a widget.Entry limitation with no workaround in the current Fyne version.
- Context menus on sidebar require `widget.PopUpMenu` manually positioned, since `widget.List` does not natively emit right-click events with item context. Implementation must track the tapped item index.
- `widget.Table` virtualizes cell rendering, making large result sets performant for display, but all row data is held in memory.

---

## Empty and Loading States

### First Launch (No Connections)

```
+--Sidebar---------+--Tab Area-----------------------------+
|                   |                                        |
|  No connections.  |  Add a connection to get started.      |
|                   |  File > New Connection                 |
|                   |                                        |
+-------------------+----------------------------------------+
```

The sidebar shows a centered placeholder label. The tab area shows a centered onboarding message with a reference to the menu action.

### No Tabs Open (Connections Exist)

```
+--Sidebar---------+--Tab Area-----------------------------+
|                   |                                        |
|  [ConnA]          |  Click a connection to open a          |
|   disconnected    |  terminal.                             |
|  [ConnB]          |                                        |
|   disconnected    |                                        |
+-------------------+----------------------------------------+
```

### Query In Progress

The status label shows "Running..." and the editor becomes temporarily read-only to prevent editing the in-flight query text. No spinner -- Fyne lacks a native indeterminate progress indicator that fits inline.

---

## Connection Form Dialog Detail

```
+--New Connection (or Edit Connection)---+
|                                         |
|  Name *        [______________________] |
|  Host *        [______________________] |
|  Port          [5432__________________] |
|  User *        [______________________] |
|  Password      [**********************] |
|  Database *    [______________________] |
|  SSL Mode      [disable           v   ] |
|                                         |
|  Note: Password is stored in plaintext. |
|                                         |
|            [Cancel]         [Save]      |
+-----------------------------------------+
```

**SSL Mode options:** disable, require, verify-ca, verify-full.

**Validation rules:**
- Name, Host, User, Database are required (marked with *).
- Port defaults to 5432 if left empty.
- SSL Mode defaults to "disable."
- On validation failure, the first invalid field receives focus and its label is highlighted.

---

## Tab Label Convention

| State                        | Label Format        |
|------------------------------|---------------------|
| Connected, no transaction    | `ConnName`          |
| Connected, transaction open  | `ConnName [TX]`     |
| Disconnected                 | `ConnName`          |

Tab labels use the connection Name field from the config, not the database name, to allow users to distinguish multiple connections to the same database.
