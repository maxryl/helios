# PRD: M3 Design Overhaul for Helios

**Product**: Helios -- Desktop PostgreSQL IDE (Go + Fyne v2.7.3, pre-release)
**Feature Type**: UI Enhancement -- M3-inspired visual redesign
**Priority**: P1 -- ship before first public release
**Date**: 2026-03-20

---

## Context and Why Now

Helios is a Fyne-based desktop PostgreSQL IDE offering connection management, a SQL editor with autocomplete, query execution with a results grid, transaction management, and terminal tabs. It currently ships with Fyne's default theme and zero visual customization.

Helios competes with pgAdmin, DBeaver, DataGrip, and TablePlus. Visual identity is a first-impression multiplier: users evaluating a new tool form quality judgments within seconds. Shipping with stock Fyne defaults signals "prototype," not "product." Applying an M3-inspired design system before the first public release is the highest-leverage move to close that perception gap.

**Research summary**: GO with HIGH confidence. Product viability HIGH, user value Medium-High, strategic alignment strong.

**Scope**: ~200-250 lines of changes across 1 new file (`m3theme.go`) and ~8 modified files. No new dependencies. No behavioral changes -- purely visual.

---

## Users and Jobs to Be Done

### Primary Persona: Developer / DBA

Developers and database administrators who use Helios as their daily PostgreSQL IDE.

| Job to Be Done | Current Pain |
|---|---|
| Scan query results quickly during long sessions | Default grid has no row differentiation; eyes lose track |
| Distinguish connected vs. disconnected databases at a glance | Unicode indicators are ambiguous and render inconsistently across platforms |
| Identify primary actions (Run Query) without hunting | All toolbar buttons look identical |
| Work comfortably in dark and light environments | Default Fyne dark theme lacks contrast hierarchy |

---

## Success Metrics

### Leading Indicators

- All existing unit and integration tests pass after the change (`go test ./...`).
- Dark and light modes both render correctly following OS preference.
- No new external dependencies introduced.

### Lagging Indicators

- Visual quality assessment moves from "Low" (default Fyne) to "Medium" (competitive with pgAdmin/DBeaver tier).
- Zero bug reports attributable to theme changes in the first 30 days after public release.

---

## Functional Requirements

### FR-1: Custom M3 Theme with Color Roles

Implement an M3-inspired color system with 18+ color roles mapped to Fyne color names, providing both dark and light variants.

**Acceptance Criteria**:
- App launches with M3-inspired colors, not Fyne defaults.
- Dark and light modes both render correctly, following OS preference.
- Color lookups are O(1) (map-based).

### FR-2: M3 Size Adjustments

Increase padding, use larger corner radii, and apply M3-standard icon sizes.

**Acceptance Criteria**:
- Padding, radii, and icon sizes visibly differ from Fyne defaults.
- No layout overflow or clipping on standard display sizes.

### FR-3: JetBrains Mono for SQL Editor

Use JetBrains Mono as the monospace font in the SQL editor.

**Acceptance Criteria**:
- SQL editor text renders in JetBrains Mono when the font is available on the system.
- Falls back gracefully to Fyne's default monospace if JetBrains Mono is absent.

### FR-4: Toolbar Importance Hierarchy

Apply visual weight to toolbar buttons: HighImportance for Run Query, MediumImportance for transaction controls, LowImportance for New Terminal.

**Acceptance Criteria**:
- Run Query is visually prominent (primary color treatment).
- Transaction controls are visually secondary.
- New Terminal is visually tertiary.
- A user can identify the primary action without reading labels.

### FR-5: Sidebar Connection Indicators

Replace Unicode status characters with proper circle indicators. Green filled circle for connected, outline circle for disconnected. Add a "Connections" header and a tinted background to the sidebar.

**Acceptance Criteria**:
- Connected databases show a green filled circle.
- Disconnected databases show an outline circle.
- Sidebar has a distinct tinted background and a "Connections" header.

### FR-6: Results Grid Alternating Rows

Apply alternating row backgrounds using surface/surfaceContainer colors. Widen default column widths.

**Acceptance Criteria**:
- Even and odd rows have visibly different background colors.
- Column widths are wider than the current default.
- Row differentiation is present in both dark and light modes.

### FR-7: Completer Styling

Use menu background color instead of overlay background for the autocomplete completer.

**Acceptance Criteria**:
- Completer popup uses the menu background color.
- Completer is visually distinguishable from the editor surface behind it.

### FR-8: Terminal Status Label

Render the terminal status label in bold.

**Acceptance Criteria**:
- Terminal status text is bold.

### FR-9: Connection Form Dialog Size

Increase the connection form dialog to 480x450.

**Acceptance Criteria**:
- Connection form dialog opens at 480x450 minimum.
- All form fields are visible without scrolling on standard displays.

---

## Non-Functional Requirements

### Performance

- Color lookups must be O(1). The theme must not introduce per-frame allocations or measurable render overhead.

### Scale

- N/A (desktop application, single user).

### SLOs / SLAs

- N/A (pre-release desktop software).

### Privacy and Security

- No change. The theme touches only visual presentation; no data handling is modified.

### Observability

- No new telemetry required. Existing `go build ./...` and `go test ./...` serve as the verification gate.

### Compatibility

- `go build ./...` must pass.
- `go test ./...` must pass.
- No new external dependencies.
- No behavioral regression in any existing feature.

---

## Scope

### In Scope

- M3 color system (dark + light) via a single new `m3theme.go` file.
- Size adjustments (padding, radii, icon sizes).
- JetBrains Mono font for the SQL editor.
- Toolbar importance hierarchy.
- Sidebar circle indicators and tinted background.
- Results grid alternating rows and wider columns.
- Completer, terminal, and connection form polish.

### Out of Scope

- Full M3 compliance (elevation, ripple, state layers -- Fyne does not support these).
- Custom widget renderers.
- Theme configuration UI or user preferences.
- Animated transitions.
- Custom icon set.

---

## Rollout Plan

1. **Implementation**: Create `m3theme.go`, modify ~8 existing files. Estimated ~200-250 lines changed.
2. **Verification**: `go build ./...` and `go test ./...` pass. Manual visual inspection of dark and light modes.
3. **Merge**: Land on main before first public release.
4. **No feature flag needed**: This is a full replacement of the default theme with no user-facing toggle. The change is purely visual with no behavioral impact, making a gradual rollout unnecessary.

---

## Risks and Open Questions

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| JetBrains Mono not installed on user's system | Medium | Low | Fyne falls back to default monospace automatically |
| M3 colors look wrong on non-sRGB displays | Low | Low | Use well-tested M3 reference palette values |
| Alternating row colors reduce contrast in one mode | Low | Medium | Test both dark and light modes manually before merge |
| Future Fyne version changes theme API | Medium | Low | Theme implementation is a single file; easy to update |

### Open Questions

1. Should the M3 theme eventually support a user-selectable accent color, or is a fixed palette sufficient for v1?
2. If JetBrains Mono is bundled as an embedded font in the future, does that count as a "new dependency" under current policy?
