# Helios -- M3 Design Overhaul UX Brief

**Product**: Helios -- Fyne-based desktop PostgreSQL IDE
**Framework**: Fyne v2.7.3 (programmatic theme system, no CSS)
**Window**: 1200 x 800 default
**Date**: 2026-03-20

---

## 1. User Stories and Acceptance Criteria

### US-1: M3 Visual Baseline

> As a user, I want the application to feel modern and polished on first launch so that Helios looks like a professional-grade tool.

**Acceptance criteria**

- The window renders with the M3 dark surface color (`#1B1B1F`) by default when the OS is in dark mode, and the light surface (`#FEFBFF`) in light mode.
- All M3 size tokens are applied: 6dp padding, 12dp inner padding, 12dp input radius, 8dp selection radius, 6dp scrollbar radius, 24dp inline icons, 20dp subheading text, 12dp caption text, 2dp input border.
- Font usage: JetBrains Mono for the SQL editor and results grid; system sans-serif for all other text (labels, buttons, menus).

### US-2: Toolbar Importance Hierarchy

> As a user, I want toolbar buttons to communicate their relative importance so that I can find the primary action (Run Query) at a glance.

**Acceptance criteria**

- "Run Query" renders as a `HighImportance` filled primary button.
- "Begin", "Commit", and "Rollback" render as `MediumImportance` tonal buttons.
- "New Terminal" renders as a `LowImportance` text button.
- Separators appear between each logical group (run | transaction | utility).
- The toolbar is built with `container.NewHBox` and `widget.Button`, not `widget.Toolbar`.

### US-3: Sidebar Connection Status

> As a user, I want to see which databases are connected without reading text so that I can manage many connections quickly.

**Acceptance criteria**

- Each connection row displays a 10dp-diameter circle to the left of the connection name.
- Connected: circle filled with the success color (`#A8DAB5` dark / `#1B7F37` light).
- Disconnected: circle rendered as an outline stroke only, using the outline color (`#938F99` dark / `#79747E` light).
- Status is conveyed by both color AND shape (filled vs outline) -- never color alone.
- A bold "Connections" header label sits above the list.
- The sidebar background uses `surfaceContainerLow` (`#1E1B20` dark / `#F7F2FA` light) to create tonal distinction from the main content area.

### US-4: Results Grid Scannability

> As a user, I want alternating row backgrounds in query results so that I can scan wide tables without losing my place.

**Acceptance criteria**

- Header row: bold text on `surfaceContainerHigh` background.
- Even data rows: `surface` background.
- Odd data rows: `surfaceContainer` background.
- Column width calculation uses a multiplier of 11 (previously 10), minimum 120dp (previously 100), maximum 350dp (previously 300).
- Alternating backgrounds are maintained during horizontal scroll.

### US-5: Autocomplete Dropdown Polish

> As a user, I want the autocomplete dropdown to look like a proper M3 menu surface so that it feels integrated with the rest of the theme.

**Acceptance criteria**

- Background color uses `surfaceContainerHighest` (`#36343B` dark / `#E6E0E9` light) via `theme.Color(theme.ColorNameMenuBackground)`.
- A 1dp border stroke rectangle surrounds the dropdown, using the outline color.

### US-6: Status Label Emphasis

> As a user, I want the terminal status text to be visually prominent so that I notice query feedback without searching for it.

**Acceptance criteria**

- Status label text is rendered with `TextStyle.Bold = true`.
- Error messages display in the error color (`#F2B8B5` dark / `#B3261E` light).
- Normal states ("Ready", "Running...", "N rows (Xms)") display in the standard foreground color.

### US-7: Connection Form Dialog Sizing

> As a user, I want the connection form to have comfortable spacing so that fields are not cramped.

**Acceptance criteria**

- Dialog size is 480 x 450 (previously 450 x 400).
- Internal spacing follows M3 inner padding (12dp).

### US-8: Seamless Theme Switching

> As a user, I want the theme to follow my OS dark/light setting automatically so that I never have to configure it inside Helios.

**Acceptance criteria**

- Fyne detects OS-level theme changes at runtime.
- All M3 color roles switch to the correct variant (dark or light) without restart.
- No in-app theme toggle is required.

---

## 2. Application Layout

```
+---------------------------------------------------+
| Menu Bar (File)                                    |
+---------------------------------------------------+
| Toolbar                                            |
| [>> Run Query]  |  [~ Begin] [~ Commit] [~ Roll-  |
|  (filled)       |   back] (tonal)   [New Terminal] |
|                 |                    (text)         |
+------------+--------------------------------------+
| Sidebar    | Terminal Tabs                         |
| (20%)      | (80%)                                 |
|            |                                       |
| CONNECTIONS| +------------------------------------+ |
|            | | SQL Editor (30%)                   | |
| [*] Conn1  | | monospace, inputBackground fill    | |
| [o] Conn2  | | [autocomplete dropdown below]      | |
|            | +------------------------------------+ |
| [+][ed][x] | | Status: "Ready" (bold)             | |
|            | +------------------------------------+ |
| background:| | Results Grid (70%)                  | |
| surfaceCon-| | HEADER (bold, surfContainerHigh)    | |
| tainerLow  | | even row (surface)                  | |
|            | | odd row  (surfaceContainer)          | |
|            | | even row (surface)                  | |
|            | +------------------------------------+ |
+------------+--------------------------------------+
```

Key:
- `[>> ...]` = HighImportance (filled primary)
- `[~ ...]` = MediumImportance (tonal)
- `[...]` plain = LowImportance (text)
- `[*]` = filled circle (connected)
- `[o]` = outline circle (disconnected)

---

## 3. M3 Color System Reference

| Fyne Color Name      | Dark Mode  | Light Mode | M3 Role                  |
|-----------------------|------------|------------|--------------------------|
| Primary              | `#A8C7FA`  | `#0B57D0`  | primary                  |
| ForegroundOnPrimary  | `#062E6F`  | `#FFFFFF`  | onPrimary                |
| Button               | `#004A77`  | `#D3E3FD`  | primaryContainer         |
| Background           | `#1B1B1F`  | `#FEFBFF`  | surface                  |
| InputBackground      | `#211F26`  | `#F3EDF7`  | surfaceContainer         |
| HeaderBackground     | `#2B2930`  | `#ECE6F0`  | surfaceContainerHigh     |
| MenuBackground       | `#36343B`  | `#E6E0E9`  | surfaceContainerHighest  |
| Foreground           | `#E6E1E5`  | `#1C1B1F`  | onSurface                |
| Placeholder          | `#CAC4D0`  | `#49454F`  | onSurfaceVariant         |
| InputBorder          | `#938F99`  | `#79747E`  | outline                  |
| Separator            | `#49454F`  | `#CAC4D0`  | outlineVariant           |
| Error                | `#F2B8B5`  | `#B3261E`  | error                    |
| OverlayBackground    | `#1E1B20`  | `#F7F2FA`  | surfaceContainerLow      |
| Success              | `#A8DAB5`  | `#1B7F37`  | custom                   |
| Warning              | `#FFD599`  | `#E07800`  | custom                   |

---

## 4. Size Token Changes

| Token            | Default | M3   | Rationale                       |
|------------------|---------|------|---------------------------------|
| Padding          | 4       | 6    | M3 generous spacing             |
| InnerPadding     | 8       | 12   | M3 component internal padding   |
| InputRadius      | 5       | 12   | M3 medium shape                 |
| SelectionRadius  | 3       | 8    | Rounder selections              |
| ScrollBarRadius  | 3       | 6    | Rounder scrollbars              |
| InlineIcon       | 20      | 24   | M3 standard icon size           |
| SubHeadingText   | 18      | 20   | M3 title-large                  |
| CaptionText      | 11      | 12   | M3 label-medium                 |
| InputBorder      | 1       | 2    | M3 outlined field border        |

---

## 5. User Flows

### Flow 1: First Launch (Dark Mode)

1. User opens Helios. OS is in dark mode.
2. Window appears at 1200 x 800 with `#1B1B1F` surface background.
3. Sidebar renders with `surfaceContainerLow` (`#1E1B20`) tinted background and a bold "Connections" header. The sidebar is empty except for the header and the add/edit/delete toolbar.
4. Toolbar displays with clear button hierarchy: filled "Run Query" stands out against tonal transaction buttons and the subtle "New Terminal" text button.
5. The SQL editor area shows an empty monospace input with `surfaceContainer` background.
6. Status label reads "Ready" in bold foreground text.
7. Results grid area is empty -- no visual artifacts, just the surface background.

**States**:
- **Empty**: No connections, no tabs. Sidebar shows header and toolbar only. Main area shows a single default tab with empty editor and empty grid.
- **Loading**: Not applicable at launch; Fyne renders synchronously.
- **Success**: Window fully rendered with all M3 tokens applied.
- **Error**: If theme fails to load, Fyne falls back to its built-in default theme. No user-facing error is shown.

### Flow 2: Running a Query and Reading Results

1. User has an active connection (green filled circle in sidebar).
2. User types SQL in the editor. Autocomplete dropdown appears below the cursor with `surfaceContainerHighest` background and outline border stroke.
3. User clicks "Run Query" (filled primary button) or presses the keyboard shortcut.
4. Status label updates to "Running..." in bold foreground text.
5. Results populate the grid:
   - Header row renders bold on `surfaceContainerHigh`.
   - Data rows alternate between `surface` (even) and `surfaceContainer` (odd).
6. Status label updates to "42 rows (12ms)" in bold foreground text.

**States**:
- **Empty**: Grid shows no rows. Status reads "Ready".
- **Loading**: Status reads "Running..." in bold. Grid is unchanged (shows previous results or empty).
- **Success**: Grid populates with alternating row backgrounds. Status shows row count and timing.
- **Error**: Status label switches to error color (`#F2B8B5` dark / `#B3261E` light) with the error message text. Grid may be empty or show previous results.

### Flow 3: Managing Connections

1. User clicks the add button in the sidebar toolbar.
2. Connection form dialog appears at 480 x 450 with 12dp internal padding.
3. User fills in connection details and submits.
4. New connection appears in the sidebar list with an outline circle (disconnected).
5. User activates the connection. Circle transitions to filled green (connected).
6. When disconnected (manually or due to error), circle returns to outline-only.

**States**:
- **Empty sidebar**: Only the "Connections" header and the add/edit/delete toolbar are visible.
- **Disconnected**: Outline-only circle next to connection name.
- **Connected**: Green filled circle next to connection name.
- **Error on connect**: Connection remains in disconnected state. Error feedback appears in the status label or dialog.

### Flow 4: Transaction Lifecycle

1. User clicks "Begin" (tonal button). A transaction starts on the active connection.
2. User executes queries within the transaction.
3. User clicks "Commit" (tonal) to persist, or "Rollback" (tonal) to discard.
4. No visual theme change occurs for transaction state -- this is behavioral only. The status label may reflect transaction state textually.

**States**:
- **No transaction**: All three transaction buttons are available. No visual indicator beyond default state.
- **Transaction active**: Behavioral state only. Buttons remain visually the same (Fyne does not support animated state layers).

### Flow 5: OS Theme Switch

1. User changes OS appearance from dark to light (or vice versa).
2. Fyne detects the change at runtime.
3. All color roles switch to their corresponding variant from the M3 table.
4. Size tokens remain the same across both modes.
5. No restart or user action inside Helios is required.

**States**:
- **Dark mode**: All "Dark Mode" column values from the color table are active.
- **Light mode**: All "Light Mode" column values from the color table are active.
- **Transition**: Instantaneous re-render by Fyne; no intermediate state is visible to the user.

---

## 6. Component Specifications

### 6.1 Toolbar

| Property        | Value                                         |
|-----------------|-----------------------------------------------|
| Container       | `container.NewHBox`                           |
| Run Query       | `widget.Button`, `HighImportance` (filled)    |
| Begin           | `widget.Button`, `MediumImportance` (tonal)   |
| Commit          | `widget.Button`, `MediumImportance` (tonal)   |
| Rollback        | `widget.Button`, `MediumImportance` (tonal)   |
| New Terminal    | `widget.Button`, `LowImportance` (text)       |
| Group separators| `widget.Separator` between logical groups     |

### 6.2 Sidebar

| Property          | Value                                                      |
|-------------------|------------------------------------------------------------|
| Background        | `surfaceContainerLow` (`#1E1B20` / `#F7F2FA`)             |
| Width             | 20% of window                                              |
| Header            | Bold label "Connections"                                   |
| List item layout  | `container.NewHBox(circle, label)`                         |
| Circle size       | 10dp diameter, vertically centered                         |
| Connected fill    | Success color (`#A8DAB5` / `#1B7F37`)                     |
| Disconnected fill | None (outline stroke only, outline color)                  |
| Bottom toolbar    | Add, Edit, Delete icon buttons                             |

### 6.3 Results Grid

| Property                | Value                                         |
|-------------------------|-----------------------------------------------|
| Header background       | `surfaceContainerHigh` (`#2B2930` / `#ECE6F0`)|
| Header text             | Bold                                          |
| Even row background     | `surface` (`#1B1B1F` / `#FEFBFF`)            |
| Odd row background      | `surfaceContainer` (`#211F26` / `#F3EDF7`)    |
| Column width multiplier | 11 (character count x 11)                     |
| Column width minimum    | 120dp                                         |
| Column width maximum    | 350dp                                         |
| Font                    | JetBrains Mono (monospace)                    |

### 6.4 Autocomplete Dropdown

| Property    | Value                                                    |
|-------------|----------------------------------------------------------|
| Background  | `surfaceContainerHighest` (`#36343B` / `#E6E0E9`)       |
| Border      | 1dp stroke rectangle, outline color                      |
| Position    | Inline below editor cursor                               |

### 6.5 Status Label

| Property     | Value                                                   |
|--------------|---------------------------------------------------------|
| Text style   | Bold (`TextStyle.Bold = true`)                          |
| Normal color | Foreground (`#E6E1E5` / `#1C1B1F`)                     |
| Error color  | Error (`#F2B8B5` / `#B3261E`)                          |

### 6.6 Connection Form Dialog

| Property | Value          |
|----------|----------------|
| Size     | 480 x 450      |
| Padding  | 12dp (M3 inner)|

---

## 7. Accessibility

### Keyboard Navigation

- All toolbar buttons are focusable and activatable via keyboard (Enter/Space). Fyne provides this by default for `widget.Button`.
- Sidebar list items are navigable with arrow keys and selectable with Enter.
- Tab key moves focus between major regions: toolbar, sidebar, editor, results.
- The autocomplete dropdown is dismissible with Escape.

### Labels and Semantics

- Toolbar buttons include text labels (not icon-only), providing built-in accessible names.
- The "Connections" header provides context for the sidebar list.
- Status label text is programmatically readable (standard Fyne label).

### Color and Contrast

- All foreground-on-background pairings meet WCAG AA 4.5:1 contrast ratio:
  - `onSurface` on `surface`: light mode `#1C1B1F` on `#FEFBFF` = 16.9:1. Dark mode `#E6E1E5` on `#1B1B1F` = 13.2:1.
  - `onSurfaceVariant` on `surfaceContainer`: light mode `#49454F` on `#F3EDF7` = 8.0:1. Dark mode `#CAC4D0` on `#211F26` = 9.5:1.
- Alternating row backgrounds provide a subtle but visible distinction without relying on color alone for data structure (rows are inherently positional).

### Non-Color Indicators

- Connection status uses both fill (color) and shape (filled circle vs outline-only circle). Users who cannot distinguish green from the background color can still differentiate filled from hollow.
- Error states use the error color AND appear in the status label as text -- the message content itself communicates the error, not just the color.
- Bold text for headers and status provides structural hierarchy independent of color.

### Spacing and Touch Targets

- M3 padding increase (4 to 6dp) and inner padding increase (8 to 12dp) result in larger interactive areas.
- Toolbar buttons with text labels and 12dp inner padding exceed the 44dp minimum touch target recommendation on touch-capable displays.
- Sidebar list items with 6dp vertical padding between rows provide adequate separation for selection.

---

## 8. Edge Cases

| Scenario                      | Behavior                                                        |
|-------------------------------|-----------------------------------------------------------------|
| Very long connection name     | Label truncates naturally via Fyne HBox layout. No tooltip.     |
| Many columns in results       | Horizontal scroll activates. Alternating row backgrounds extend across all columns. |
| Single row result             | Renders on `surface` background (even row). No alternating.     |
| Empty sidebar                 | "Connections" header and add/edit/delete toolbar visible. No list items. |
| Empty results grid            | Clean surface background. No placeholder text. Status reads "Ready". |
| Very large result set         | Fyne list virtualization handles rendering. Alternating colors are index-based, not pre-rendered. |
| Dialog overflow               | 480 x 450 accommodates all connection form fields with M3 spacing. Scroll if needed. |
| Rapid theme switching         | Fyne re-renders synchronously on OS theme change. No debounce needed. |

---

## 9. Framework Constraints

These visual treatments are explicitly **not possible** in Fyne v2.7.3 and are excluded from scope:

- **Elevation shadows**: Fyne has no box-shadow or elevation system. Surface hierarchy is communicated through tonal color differences only.
- **Ripple effects**: No touch/click ripple animation. Button press feedback is limited to Fyne's built-in pressed state color shift.
- **Animated state layers**: No hover/focus/pressed opacity overlays. Fyne provides basic hover highlighting only.
- **Custom widget renderers**: All components use standard Fyne widgets. No custom `WidgetRenderer` implementations are required for this overhaul.
- **CSS or stylesheet**: Fyne themes are programmatic (`fyne.Theme` interface). All styling is via Go code, not markup.

---

## 10. Summary of Visual Changes

| Component          | Before                          | After                                        |
|--------------------|---------------------------------|----------------------------------------------|
| Toolbar            | Icon-only `widget.Toolbar`      | `HBox` with importance-tiered `Button`s      |
| Sidebar background | Default surface                 | `surfaceContainerLow` tinted background      |
| Sidebar indicators | `bullet` / `circle` as text     | Canvas circles: filled (connected) / outline (disconnected) |
| Sidebar header     | None                            | Bold "Connections" label                     |
| Results header     | Bold text, default background   | Bold text, `surfaceContainerHigh` background |
| Results rows       | Uniform background              | Alternating `surface` / `surfaceContainer`   |
| Results columns    | width x10, min 100, max 300     | width x11, min 120, max 350                  |
| Autocomplete       | `OverlayBackground`, no border  | `MenuBackground` with outline border stroke  |
| Status label       | Plain text                      | Bold text, error color for errors            |
| Connection dialog  | 450 x 400                       | 480 x 450                                   |
| All sizing         | Fyne defaults                   | M3 token overrides (see size table)          |
