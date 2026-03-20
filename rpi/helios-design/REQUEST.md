# M3 Design Overhaul for Helios

## Context

Helios is a Fyne-based desktop PostgreSQL IDE. It currently uses Fyne's default theme with no customization. The goal is to apply Material Design 3 (M3) styling — colors, typography, spacing, and component patterns — to give the app a modern, polished look. Since Fyne uses a programmatic theme system (not CSS), this means implementing a custom `fyne.Theme` and enhancing individual widgets.

## Phase 1: Create M3 Theme (`internal/ui/m3theme.go` — NEW)

Implement `fyne.Theme` interface with M3 color roles mapped to Fyne color names. Both light and dark variants.

**Color mappings** (M3 baseline blue scheme):

| Fyne Color | Dark Mode | Light Mode | M3 Role |
|---|---|---|---|
| Primary | `#A8C7FA` | `#0B57D0` | primary |
| ForegroundOnPrimary | `#062E6F` | `#FFFFFF` | onPrimary |
| Button | `#004A77` | `#D3E3FD` | primaryContainer |
| Background | `#1B1B1F` | `#FEFBFF` | surface |
| InputBackground | `#211F26` | `#F3EDF7` | surfaceContainer |
| HeaderBackground | `#2B2930` | `#ECE6F0` | surfaceContainerHigh |
| MenuBackground | `#36343B` | `#E6E0E9` | surfaceContainerHighest |
| Foreground | `#E6E1E5` | `#1C1B1F` | onSurface |
| Placeholder | `#CAC4D0` | `#49454F` | onSurfaceVariant |
| InputBorder | `#938F99` | `#79747E` | outline |
| Separator | `#49454F` | `#CAC4D0` | outlineVariant |
| Error | `#F2B8B5` | `#B3261E` | error |
| Hover | primary @ 8% alpha | primary @ 8% alpha | stateLayer |
| Pressed | primary @ 16% alpha | primary @ 16% alpha | stateLayer |
| Selection | `#004A77` @ 40% | `#D3E3FD` @ 60% | secondaryContainer |
| OverlayBackground | `#1E1B20` | `#F7F2FA` | surfaceContainerLow |
| Focus | primary @ 30% alpha | primary @ 30% alpha | focusRing |
| Success | `#A8DAB5` | `#1B7F37` | custom |
| Warning | `#FFD599` | `#E07800` | custom |

**Size adjustments** (M3 spacing/shape):

| Size | Default | M3 | Rationale |
|---|---|---|---|
| Padding | 4 | 6 | M3 generous spacing |
| InnerPadding | 8 | 12 | M3 component internal padding |
| InputRadius | 5 | 12 | M3 medium shape |
| SelectionRadius | 3 | 8 | Rounder selections |
| ScrollBarRadius | 3 | 6 | Rounder scrollbars |
| InlineIcon | 20 | 24 | M3 standard icon size 24dp |
| SubHeadingText | 18 | 20 | M3 title-large |
| CaptionText | 11 | 12 | M3 label-medium |
| InputBorder | 1 | 2 | M3 outlined field border |

**Font**: Use JetBrains Mono for monospace (SQL editor). Bundle via `fyne bundle`. Delegate non-monospace to Fyne's default sans-serif (adequate for M3).

## Phase 2: Apply Theme (`main.go`)

Add one line after `app.New()`:

```go
a.Settings().SetTheme(ui.NewM3Theme())
```

This alone transforms the entire app's colors, spacing, and radii globally.

## Phase 3: Enhance Toolbar (`internal/ui/toolbar.go`)

Replace `widget.Toolbar` with `container.NewHBox` of `widget.Button` instances using M3 importance hierarchy:

- **Run Query**: `widget.HighImportance` (filled primary) — primary action
- **Begin/Commit/Rollback**: `widget.MediumImportance` (tonal) — secondary actions
- **New Terminal**: `widget.LowImportance` (text button)
- Add `widget.NewSeparator()` between logical groups

Change `Widget()` return type from `*widget.Toolbar` to `fyne.CanvasObject`. This is safe — `app.go:91` uses it in `container.NewBorder()` which accepts `fyne.CanvasObject`.

**Files affected**: `toolbar.go`, `app.go` (type of `toolbar` field stays `*Toolbar`, only the `Widget()` return changes)

## Phase 4: Enhance Sidebar (`internal/ui/sidebar.go`)

1. **Circle status indicators**: Replace `●`/`○` text with `canvas.Circle` objects — green fill when connected, just a stroke outline when disconnected
2. **"Connections" header**: Add a bold label above the toolbar
3. **Tinted background**: Wrap sidebar in `container.NewStack` with a `canvas.Rectangle` using the surfaceContainerLow color to create M3 tonal surface distinction from the main content area

Update `CreateItem` to return `container.NewHBox(circle, label)` and `UpdateItem` to set circle fill/stroke + label text.

## Phase 5: Enhance Results Grid (`internal/ui/resultsgrid.go`)

1. **Alternating row backgrounds**: Change `CreateCell` to return `container.NewStack(bgRect, label)`. In `UpdateCell`, set even rows to surface color, odd rows to surfaceContainer color — M3 data table pattern
2. **Improved column widths**: Slightly more generous — multiply by 11, min 120, max 350

## Phase 6: Minor Enhancements

### `internal/ui/completer.go`
- Line 272: Change `theme.OverlayBackgroundColor()` to `theme.Color(theme.ColorNameMenuBackground)` for M3 menu surface treatment
- Add a border stroke to the background rectangle

### `internal/ui/terminal.go`
- Style status label with bold text: `t.statusLabel.TextStyle.Bold = true`

### `internal/ui/connform.go`
- Increase dialog size to `fyne.NewSize(480, 450)` to accommodate increased M3 padding

## Implementation Order

1. `internal/ui/m3theme.go` (NEW) — foundation, everything depends on it
2. `main.go` — apply theme (entire app transforms)
3. `internal/ui/toolbar.go` + `internal/ui/app.go` — button hierarchy
4. `internal/ui/sidebar.go` — circle indicators + tinted background
5. `internal/ui/resultsgrid.go` — alternating rows
6. `internal/ui/completer.go` — menu background color
7. `internal/ui/terminal.go` — status label styling
8. `internal/ui/connform.go` — dialog size

## Verification

1. `go build ./...` — ensure compilation
2. `go test ./...` — ensure existing tests pass
3. Launch the app and visually verify:
   - Dark/light theme both work (Fyne follows system preference)
   - Toolbar buttons show importance hierarchy (filled vs tonal vs text)
   - Sidebar has tinted background and circle indicators
   - Results grid has alternating row backgrounds
   - Autocomplete dropdown uses menu background color
   - Rounded input fields and selections
   - Overall spacing feels generous and modern
