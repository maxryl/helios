# M3 Design Overhaul for Helios -- Technical Specification

**Codebase**: Go 1.24.0, Fyne v2.7.3, module name `helios`
**Scope**: 1 new file, 8 modified files, ~200-250 LOC

---

## Architecture Overview

The M3 theme implementation follows Fyne's native theming architecture. A single
`fyne.Theme` implementation propagates colors, sizes, and fonts to all widgets
automatically. Widget-level changes are only needed where structural modifications
are required (toolbar button types, sidebar indicators, grid cell backgrounds).

```
main.go
  └── app.Settings().SetTheme(NewM3Theme())
        └── m3theme.go implements fyne.Theme
              ├── Color(name, variant) → M3 color lookup tables
              ├── Font(style) → JetBrains Mono for monospace
              ├── Icon(name) → delegate to DefaultTheme
              └── Size(name) → M3 spacing/shape values
```

---

## File-by-File Specification

### 1. `internal/ui/m3theme.go` (NEW -- ~120 lines)

```go
package ui

import (
    "image/color"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/theme"
)

type m3Theme struct {
    dark  map[fyne.ThemeColorName]color.Color
    light map[fyne.ThemeColorName]color.Color
    sizes map[fyne.ThemeSizeName]float32
}

func NewM3Theme() fyne.Theme { ... }

func (t *m3Theme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
    // Look up in dark/light map based on variant
    // Fall back to theme.DefaultTheme() for unknown names
}

func (t *m3Theme) Font(style fyne.TextStyle) fyne.Resource {
    // Return bundled JetBrains Mono for Monospace style
    // Delegate to theme.DefaultTheme() for other styles
}

func (t *m3Theme) Icon(name fyne.ThemeIconName) fyne.Resource {
    return theme.DefaultTheme().Icon(name)
}

func (t *m3Theme) Size(name fyne.ThemeSizeName) float32 {
    // Look up in sizes map
    // Fall back to theme.DefaultTheme() for unknown names
}
```

#### Color Lookup Tables

**Dark mode map:**

| Fyne Color Name                      | Hex       | NRGBA                                                      |
|--------------------------------------|-----------|-------------------------------------------------------------|
| `theme.ColorNamePrimary`             | `#A8C7FA` | `color.NRGBA{R: 0xA8, G: 0xC7, B: 0xFA, A: 0xFF}`        |
| `theme.ColorNameForegroundOnPrimary` | `#062E6F` | `color.NRGBA{R: 0x06, G: 0x2E, B: 0x6F, A: 0xFF}`        |
| `theme.ColorNameButton`              | `#004A77` | `color.NRGBA{R: 0x00, G: 0x4A, B: 0x77, A: 0xFF}`        |
| `theme.ColorNameBackground`          | `#1B1B1F` | `color.NRGBA{R: 0x1B, G: 0x1B, B: 0x1F, A: 0xFF}`        |
| `theme.ColorNameInputBackground`     | `#211F26` | `color.NRGBA{R: 0x21, G: 0x1F, B: 0x26, A: 0xFF}`        |
| `theme.ColorNameHeaderBackground`    | `#2B2930` | `color.NRGBA{R: 0x2B, G: 0x29, B: 0x30, A: 0xFF}`        |
| `theme.ColorNameMenuBackground`      | `#36343B` | `color.NRGBA{R: 0x36, G: 0x34, B: 0x3B, A: 0xFF}`        |
| `theme.ColorNameForeground`          | `#E6E1E5` | `color.NRGBA{R: 0xE6, G: 0xE1, B: 0xE5, A: 0xFF}`        |
| `theme.ColorNamePlaceHolder`         | `#CAC4D0` | `color.NRGBA{R: 0xCA, G: 0xC4, B: 0xD0, A: 0xFF}`        |
| `theme.ColorNameInputBorder`         | `#938F99` | `color.NRGBA{R: 0x93, G: 0x8F, B: 0x99, A: 0xFF}`        |
| `theme.ColorNameSeparator`           | `#49454F` | `color.NRGBA{R: 0x49, G: 0x45, B: 0x4F, A: 0xFF}`        |
| `theme.ColorNameError`               | `#F2B8B5` | `color.NRGBA{R: 0xF2, G: 0xB8, B: 0xB5, A: 0xFF}`        |
| `theme.ColorNameHover`               | primary @ 8%  | `color.NRGBA{R: 0xA8, G: 0xC7, B: 0xFA, A: 0x14}`    |
| `theme.ColorNamePressed`             | primary @ 16% | `color.NRGBA{R: 0xA8, G: 0xC7, B: 0xFA, A: 0x29}`    |
| `theme.ColorNameSelection`           | `#004A77` @ 40% | `color.NRGBA{R: 0x00, G: 0x4A, B: 0x77, A: 0x66}`  |
| `theme.ColorNameOverlayBackground`   | `#1E1B20` | `color.NRGBA{R: 0x1E, G: 0x1B, B: 0x20, A: 0xFF}`        |
| `theme.ColorNameFocus`               | primary @ 30% | `color.NRGBA{R: 0xA8, G: 0xC7, B: 0xFA, A: 0x4D}`    |
| `theme.ColorNameSuccess`             | `#A8DAB5` | `color.NRGBA{R: 0xA8, G: 0xDA, B: 0xB5, A: 0xFF}`        |
| `theme.ColorNameWarning`             | `#FFD599` | `color.NRGBA{R: 0xFF, G: 0xD5, B: 0x99, A: 0xFF}`        |

**Light mode map:** Same structure with light variant colors from REQUEST.md.

#### Size Map

| Fyne Size Name                    | Value |
|-----------------------------------|-------|
| `theme.SizeNamePadding`           | 6     |
| `theme.SizeNameInnerPadding`      | 12    |
| `theme.SizeNameInputRadius`       | 12    |
| `theme.SizeNameSelectionRadius`   | 8     |
| `theme.SizeNameScrollBarRadius`   | 6     |
| `theme.SizeNameInlineIcon`        | 24    |
| `theme.SizeNameSubHeadingText`    | 20    |
| `theme.SizeNameCaptionText`       | 12    |
| `theme.SizeNameInputBorder`       | 2     |

#### Font Handling

Bundle JetBrains Mono via `fyne bundle -o internal/ui/bundled.go JetBrainsMono-Regular.ttf`.
Return the bundled resource for `fyne.TextStyle{Monospace: true}`. Delegate all other
styles to `theme.DefaultTheme().Font(style)`.

```go
func (t *m3Theme) Font(style fyne.TextStyle) fyne.Resource {
    if style.Monospace {
        return resourceJetBrainsMonoRegularTtf
    }
    return theme.DefaultTheme().Font(style)
}
```

---

### 2. `main.go` (1 line change)

Current (line 14):

```go
a := app.New()
```

After:

```go
a := app.New()
a.Settings().SetTheme(ui.NewM3Theme())
```

---

### 3. `internal/ui/toolbar.go` (Major refactor -- ~30 lines changed)

**Current**: Uses `widget.NewToolbar` with `ToolbarAction` items. `Widget()` returns
`*widget.Toolbar`.

**After**:

- Replace `toolbar *widget.Toolbar` field with `widget fyne.CanvasObject`.
- Build an HBox of `widget.Button` with icons and importance levels.

```go
runBtn := widget.NewButtonWithIcon("Run", theme.MediaPlayIcon(), tb.runQuery)
runBtn.Importance = widget.HighImportance

beginBtn := widget.NewButtonWithIcon("Begin", theme.MediaRecordIcon(), tb.beginTx)
beginBtn.Importance = widget.MediumImportance
// ... similar for Commit, Rollback

newTermBtn := widget.NewButtonWithIcon("New", theme.ContentAddIcon(), tb.newTerminal)
newTermBtn.Importance = widget.LowImportance

tb.widget = container.NewHBox(
    runBtn,
    widget.NewSeparator(),
    beginBtn, commitBtn, rollbackBtn,
    widget.NewSeparator(),
    newTermBtn,
)
```

- `Widget()` returns `fyne.CanvasObject` (was `*widget.Toolbar`).
- No changes needed in `app.go` -- `container.NewBorder()` already accepts
  `fyne.CanvasObject`.

---

### 4. `internal/ui/sidebar.go` (~25 lines changed)

#### CreateItem Change (line 52-53)

```go
// Before:
func() fyne.CanvasObject { return widget.NewLabel("template") }

// After:
func() fyne.CanvasObject {
    circle := canvas.NewCircle(color.Transparent)
    circle.StrokeColor = theme.Color(theme.ColorNameInputBorder, theme.VariantDark)
    circle.StrokeWidth = 1.5
    label := widget.NewLabel("template")
    return container.NewHBox(
        container.New(&fixedSizeLayout{size: fyne.NewSquareSize(10)}, circle),
        label,
    )
}
```

#### UpdateItem Change (line 55-66)

```go
// Before: label := obj.(*widget.Label)
// After:
box := obj.(*fyne.Container)
circle := box.Objects[0].(*fyne.Container).Objects[0].(*canvas.Circle)
label := box.Objects[1].(*widget.Label)

if s.connMgr.IsConnected(conn.ID) {
    circle.FillColor = theme.Color(theme.ColorNameSuccess, /* variant */)
    circle.StrokeColor = color.Transparent
} else {
    circle.FillColor = color.Transparent
    circle.StrokeColor = theme.Color(theme.ColorNameInputBorder, /* variant */)
}
circle.Refresh()
label.SetText(conn.Name)
```

#### Tinted Background

Wrap the sidebar container:

```go
bg := canvas.NewRectangle(theme.Color(theme.ColorNameOverlayBackground, /* variant */))
s.container = container.NewStack(bg, container.NewBorder(header, toolbar, nil, nil, s.list))
```

#### "Connections" Header

Add a bold label above the toolbar.

**Note**: A `fixedSizeLayout` helper is needed for the circle (or reuse the existing
`fixedHeightLayout` pattern from completer.go adapted for both dimensions).

---

### 5. `internal/ui/resultsgrid.go` (~20 lines changed)

#### CreateCell Change (line 26-28)

```go
// Before:
func() fyne.CanvasObject { return widget.NewLabel("") }

// After:
func() fyne.CanvasObject {
    bg := canvas.NewRectangle(color.Transparent)
    label := widget.NewLabel("")
    return container.NewStack(bg, label)
}
```

#### UpdateCell Change (line 29-39)

```go
// Before: label := cell.(*widget.Label)
// After:
stack := cell.(*fyne.Container)
bg := stack.Objects[0].(*canvas.Rectangle)
label := stack.Objects[1].(*widget.Label)

if id.Row == 0 {
    label.SetText(rg.columns[id.Col])
    label.TextStyle.Bold = true
    bg.FillColor = color.Transparent
} else {
    label.SetText(rg.rows[id.Row-1][id.Col])
    label.TextStyle.Bold = false
    if id.Row%2 == 0 {
        bg.FillColor = theme.Color(theme.ColorNameInputBackground, /* variant */)
    } else {
        bg.FillColor = color.Transparent
    }
}
bg.Refresh()
label.Refresh()
```

#### Column Widths (lines 51-58)

```go
w := len(col) * 11    // was 10
if w < 120 { w = 120 } // was 100
if w > 350 { w = 350 } // was 300
```

---

### 6. `internal/ui/completer.go` (2-3 lines changed)

Line 272:

```go
// Before:
c.bg = canvas.NewRectangle(theme.OverlayBackgroundColor())

// After:
c.bg = canvas.NewRectangle(theme.Color(theme.ColorNameMenuBackground, theme.VariantDark))
c.bg.StrokeColor = theme.Color(theme.ColorNameInputBorder, theme.VariantDark)
c.bg.StrokeWidth = 1
```

**Note**: The variant should ideally be determined dynamically. Since Fyne's
`canvas.Rectangle` does not auto-update on theme change, colors are set at creation
time and may not update on theme switch. Accept this limitation for v1; Fyne's
standard widgets handle variant switching internally.

---

### 7. `internal/ui/terminal.go` (1 line added)

After line 119 (`t.statusLabel = widget.NewLabel("Ready")`):

```go
t.statusLabel.TextStyle.Bold = true
```

---

### 8. `internal/ui/connform.go` (1 line changed)

Line 102:

```go
// Before:
dlg.Resize(fyne.NewSize(450, 400))

// After:
dlg.Resize(fyne.NewSize(480, 450))
```

---

## Font Bundling

JetBrains Mono must be bundled as a Go resource:

```bash
# Download JetBrains Mono Regular TTF
# Run from internal/ui/:
fyne bundle -o bundled.go -name jetBrainsMonoRegular JetBrainsMono-Regular.ttf
```

This generates `internal/ui/bundled.go` containing the font as a `fyne.StaticResource`.

---

## Helper Types

### fixedSizeLayout (for circle indicators in sidebar)

```go
type fixedSizeLayout struct {
    size fyne.Size
}

func (l *fixedSizeLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
    return l.size
}

func (l *fixedSizeLayout) Layout(objects []fyne.CanvasObject, _ fyne.Size) {
    for _, o := range objects {
        o.Move(fyne.NewPos(0, 0))
        o.Resize(l.size)
    }
}
```

---

## Testing Strategy

**Compilation**: `go build ./...` -- verifies all type assertions and interface
implementations compile.

**Existing tests**: `go test ./...` -- no UI tests exist, but db and config tests must
pass unchanged.

**Manual verification checklist**:

1. Dark mode: all colors correct, text readable.
2. Light mode: all colors correct, text readable.
3. Toolbar: Run button visually prominent (filled), transaction buttons tonal,
   New Terminal subtle.
4. Sidebar: green circles for connected, outline for disconnected.
5. Results grid: alternating row backgrounds visible but subtle.
6. Completer: menu background distinct from main surface.
7. Status label: bold text.
8. Connection form: fits within larger dialog.

---

## Risk Mitigations

1. **Type assertion safety**: All `UpdateItem`/`UpdateCell` callbacks change from
   `obj.(*widget.Label)` to container type assertions. If Fyne recycles objects
   unexpectedly, these panic. Mitigation: the type returned by `CreateItem`/`CreateCell`
   is deterministic -- Fyne always passes the same type back.

2. **Theme variant detection**: Some canvas objects (like the completer background) are
   created once but need to reflect the current theme variant. For static rectangles,
   colors are set at creation time and may not update on theme change. Mitigation: accept
   this limitation for v1; Fyne's standard widgets handle variant switching internally.

3. **Font fallback**: If JetBrains Mono bundling fails or is deferred, the `Font()`
   method can delegate to `DefaultTheme` for all styles, which is a safe no-op.

---

## Dependencies

- No new Go module dependencies.
- JetBrains Mono font (SIL OFL 1.1 license -- permits bundling).
- All changes use existing Fyne v2.7.3 APIs.
