# Implementation Plan: M3 Design Overhaul

## Overview

| Field | Value |
|---|---|
| Feature | Material Design 3 (M3) Theme Overhaul |
| Target | `internal/ui` package + `main.go` |
| Complexity | Medium |
| Total Phases | 4 |
| Total Tasks | 15 |
| New Files | 1 (`internal/ui/m3theme.go`) + 1 (`internal/ui/bundled.go` via fyne bundle) |
| Modified Files | 8 |

---

## Phase 1: Theme Foundation (5 tasks)

**Goal**: Create the M3 theme implementation and activate it. This is the foundation — the entire app transforms with just this phase.

**Files**: `internal/ui/m3theme.go` (NEW), `internal/ui/bundled.go` (NEW, generated), `main.go`

### Tasks

#### 1.1. Download and bundle JetBrains Mono font (Low complexity)

- Download JetBrains Mono Regular TTF
- Run `fyne bundle -o internal/ui/bundled.go -name jetBrainsMonoRegular JetBrainsMono-Regular.ttf`
- Verify generated file compiles

#### 1.2. Create m3theme.go with Color() method (Medium complexity)

- Create `internal/ui/m3theme.go`
- Define `m3Theme` struct with dark and light color maps
- Implement `Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color`
- Populate dark mode color map with all 18+ M3 color roles:
  - Primary: `#A8C7FA`
  - ForegroundOnPrimary: `#062E6F`
  - Button: `#004A77`
  - Background: `#1B1B1F`
  - InputBackground: `#211F26`
  - HeaderBackground: `#2B2930`
  - MenuBackground: `#36343B`
  - Foreground: `#E6E1E5`
  - Placeholder: `#CAC4D0`
  - InputBorder: `#938F99`
  - Separator: `#49454F`
  - Error: `#F2B8B5`
  - Hover: primary @ 8%
  - Pressed: primary @ 16%
  - Selection: `#004A77` @ 40%
  - OverlayBackground: `#1E1B20`
  - Focus: primary @ 30%
  - Success: `#A8DAB5`
  - Warning: `#FFD599`
- Populate light mode color map with corresponding light values
- Fall back to `theme.DefaultTheme().Color()` for unknown names

#### 1.3. Implement Size() method (Low complexity)

- Add size map to `m3Theme`
- Map M3 sizes:
  - Padding: 6
  - InnerPadding: 12
  - InputRadius: 12
  - SelectionRadius: 8
  - ScrollBarRadius: 6
  - InlineIcon: 24
  - SubHeadingText: 20
  - CaptionText: 12
  - InputBorder: 2
- Fall back to `theme.DefaultTheme().Size()` for unknown names

#### 1.4. Implement Font() and Icon() methods (Low complexity)

- Font(): Return bundled JetBrains Mono for `TextStyle{Monospace: true}`, delegate others to DefaultTheme
- Icon(): Delegate entirely to `theme.DefaultTheme().Icon()`
- Export `NewM3Theme() fyne.Theme` constructor

#### 1.5. Activate theme in main.go (Low complexity)

- Add `a.Settings().SetTheme(ui.NewM3Theme())` after `a := app.New()` in main.go

### Validation Gate

- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] App launches with M3 colors (visually verify dark and light)
- [ ] All existing widgets inherit M3 colors/spacing automatically
- [ ] SQL editor uses JetBrains Mono

---

## Phase 2: Toolbar Enhancement (3 tasks)

**Goal**: Replace icon-only toolbar with M3 importance hierarchy buttons.

**Files**: `internal/ui/toolbar.go`, `internal/ui/app.go` (no change needed — already accepts `fyne.CanvasObject`)

### Tasks

#### 2.1. Replace toolbar field type (Low complexity)

- Change `toolbar *widget.Toolbar` field to `widget fyne.CanvasObject`
- Update `Widget()` return type from `*widget.Toolbar` to `fyne.CanvasObject`

#### 2.2. Create button-based toolbar (Medium complexity)

- Replace `widget.NewToolbar(...)` with `container.NewHBox(...)` of `widget.Button`:
  - Run Query: `widget.NewButtonWithIcon("Run", theme.MediaPlayIcon(), tb.runQuery)` with `Importance = widget.HighImportance`
  - Begin: `widget.NewButtonWithIcon("Begin", theme.MediaRecordIcon(), tb.beginTx)` with `Importance = widget.MediumImportance`
  - Commit: `widget.NewButtonWithIcon("Commit", theme.ConfirmIcon(), tb.commitTx)` with `Importance = widget.MediumImportance`
  - Rollback: `widget.NewButtonWithIcon("Rollback", theme.CancelIcon(), tb.rollbackTx)` with `Importance = widget.MediumImportance`
  - New Terminal: `widget.NewButtonWithIcon("New", theme.ContentAddIcon(), tb.newTerminal)` with `Importance = widget.LowImportance`
- Add `widget.NewSeparator()` between groups

#### 2.3. Verify app.go integration (Low complexity)

- Confirm `container.NewBorder(a.toolbar.Widget(), ...)` still works with new return type
- No code change expected — just verification

### Validation Gate

- [ ] `go build ./...` passes
- [ ] Toolbar shows filled (Run), tonal (Begin/Commit/Rollback), and text (New) buttons
- [ ] All toolbar actions still work (Run Query, Begin/Commit/Rollback, New Terminal)

---

## Phase 3: Sidebar & Results Grid (4 tasks)

**Goal**: Add circle status indicators to sidebar and alternating row backgrounds to results grid.

**Files**: `internal/ui/sidebar.go`, `internal/ui/resultsgrid.go`

### Tasks

#### 3.1. Add fixedSizeLayout helper (Low complexity)

- Create `fixedSizeLayout` struct in a suitable file (e.g., `m3theme.go` or a new `layouts.go`)
- Implements `fyne.Layout` with fixed MinSize and Layout methods
- Used for constraining circle indicator size in sidebar

#### 3.2. Enhance sidebar with circle indicators and tinted background (High complexity)

- Add imports: `canvas`, `color`
- Change `CreateItem` to return `container.NewHBox(circleContainer, label)`:
  - Circle: `canvas.NewCircle(color.Transparent)` inside `fixedSizeLayout{size: 10x10}`
  - Label: `widget.NewLabel("template")`
- Change `UpdateItem` type assertions:
  - Extract circle and label from container hierarchy
  - Connected: green fill (Success color), transparent stroke
  - Disconnected: transparent fill, outline stroke (InputBorder color)
  - Set label text to `conn.Name` (no more unicode bullets)
- Add "Connections" bold header label above sidebar toolbar
- Wrap sidebar in `container.NewStack(bgRect, content)` with surfaceContainerLow background

#### 3.3. Add alternating row backgrounds to results grid (Medium complexity)

- Change `CreateCell` to return `container.NewStack(bgRect, label)`:
  - `canvas.NewRectangle(color.Transparent)` as background
  - `widget.NewLabel("")` as content
- Change `UpdateCell` type assertions:
  - Extract bg rectangle and label from stack
  - Even data rows: surfaceContainer color background
  - Odd data rows: transparent (surface color from theme)
  - Header row: transparent background, bold text

#### 3.4. Update column width formula (Low complexity)

- Change multiplier from 10 to 11
- Change minimum from 100 to 120
- Change maximum from 300 to 350

### Validation Gate

- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Sidebar shows green circles for connected, outline for disconnected
- [ ] Sidebar has tinted background distinct from main content
- [ ] Results grid has alternating row backgrounds
- [ ] Column widths are slightly more generous

---

## Phase 4: Minor Polishes (3 tasks)

**Goal**: Apply remaining small enhancements across completer, terminal, and connection form.

**Files**: `internal/ui/completer.go`, `internal/ui/terminal.go`, `internal/ui/connform.go`

### Tasks

#### 4.1. Update completer background (Low complexity)

- Line 272: Change `canvas.NewRectangle(theme.OverlayBackgroundColor())` to `canvas.NewRectangle(theme.Color(theme.ColorNameMenuBackground, theme.VariantDark))`
- Add border stroke: `c.bg.StrokeColor = theme.Color(theme.ColorNameInputBorder, theme.VariantDark)` and `c.bg.StrokeWidth = 1`

#### 4.2. Bold terminal status label (Low complexity)

- After line 119 in `terminal.go`, add: `t.statusLabel.TextStyle.Bold = true`

#### 4.3. Increase connection form dialog size (Low complexity)

- Line 102 in `connform.go`: Change `fyne.NewSize(450, 400)` to `fyne.NewSize(480, 450)`

### Validation Gate

- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Completer dropdown uses menu background color with border
- [ ] Status label is bold
- [ ] Connection form has adequate space for M3 padding

---

## Final Validation

After all 4 phases are complete:

- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Dark theme: all M3 colors correct, text readable, consistent
- [ ] Light theme: all M3 colors correct, text readable, consistent
- [ ] Toolbar: clear visual hierarchy (filled, tonal, text)
- [ ] Sidebar: circle indicators, tinted background, "Connections" header
- [ ] Results grid: alternating rows, wider columns
- [ ] Completer: menu background with border
- [ ] Terminal: bold status label
- [ ] Connection form: fits with M3 spacing
- [ ] JetBrains Mono in SQL editor
- [ ] All keyboard shortcuts work (Ctrl+Enter, Ctrl+T, Ctrl+W)
- [ ] All toolbar/sidebar actions work
- [ ] No behavioral regressions

---

## Dependency Chart

```
Phase 1 (Theme Foundation)
  ├── Task 1.1 (Bundle font) ──┐
  ├── Task 1.2 (Color method) ──┼── Task 1.4 (Font/Icon) ── Task 1.5 (Activate)
  └── Task 1.3 (Size method) ──┘
        │
        ▼
Phase 2 (Toolbar) ─── can start after Phase 1 complete
        │
        ▼
Phase 3 (Sidebar & Grid) ─── can start after Phase 1 complete (parallel with Phase 2)
  ├── Task 3.1 (Layout helper)
  ├── Task 3.2 (Sidebar) ─── depends on 3.1
  ├── Task 3.3 (Grid alternating rows)
  └── Task 3.4 (Column widths)
        │
        ▼
Phase 4 (Minor Polishes) ─── can start after Phase 1 complete (parallel with 2 & 3)
```

**Note**: Phases 2, 3, and 4 are independent of each other and can be implemented in parallel after Phase 1 is complete. The ordering is for logical grouping, not strict sequencing.

---

## Fallback Plan

If widget-level changes in Phases 2-3 prove problematic (type assertion issues, visual glitches):

- **Minimum viable delivery**: Phase 1 alone delivers ~60% of visual improvement
- **Phase 2 fallback**: Keep existing `widget.Toolbar` but with M3 colors from theme
- **Phase 3 fallback**: Skip container wrapping; sidebar keeps unicode bullets, grid keeps uniform background
