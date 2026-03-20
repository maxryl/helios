# Research Report: Material Design 3 (M3) Theme Overhaul

## Overview

| Field | Value |
|---|---|
| Feature Name | Material Design 3 (M3) Theme Overhaul |
| Feature Type | UI Enhancement — visual redesign and theming |
| Target Component | `internal/ui` package + `main.go` |
| Complexity | Medium |
| Decision | GO |
| Confidence | High |
| Framing | M3-inspired theming for pre-release visual polish |

---

## Executive Summary

The M3 Theme Overhaul is a small, well-scoped UI enhancement (~200-250 lines across 9 files) that replaces Fyne's default theming with an M3-inspired custom theme, adds alternating row backgrounds to the results grid, introduces circle status indicators in the sidebar, and applies visual polish across the toolbar, completer, terminal, and connection form. The recommendation is **GO** with high confidence: the work is technically straightforward, strategically necessary for a pre-release product competing against polished tools like DataGrip and TablePlus, and the codebase is in an ideal state for this change — small, clean, and early enough that modifications are cheap. The primary risks are type assertion panics when wrapping widgets in containers and scope creep toward full M3 compliance, both of which are mitigated by defensive coding and explicit scoping as "M3-inspired."

---

## Feature Overview

### Goals

1. Replace default Fyne theming with a custom M3-inspired color system supporting dark and light variants.
2. Improve results grid readability with alternating row backgrounds.
3. Replace Unicode status indicators in the sidebar with proper circle indicators and a tinted background.
4. Apply toolbar importance hierarchy, M3 menu colors to the completer, bold status labels, and increased connection form size.
5. Achieve visual parity with the polish level expected of a database IDE competing with established tools.

### Scope

| Metric | Value |
|---|---|
| New files | 1 (`internal/ui/m3theme.go`) |
| Modified files | ~8 |
| Estimated LOC | ~200-250 |
| New dependencies | None |

---

## Requirements Summary

### Functional Requirements

- **Custom Theme**: New `internal/ui/m3theme.go` implementing `fyne.Theme` interface with M3 color roles for both dark and light variants.
- **Theme Integration**: One-line change in `main.go` to set the custom theme.
- **Toolbar**: Replace `widget.Toolbar` with HBox of buttons using importance hierarchy (primary, secondary actions).
- **Sidebar**: Circle status indicators replacing Unicode hack, "Connections" header label, tinted background.
- **Results Grid**: Alternating row backgrounds for improved readability, improved column widths.
- **Completer**: M3 menu background color via `theme.OverlayBackgroundColor()`.
- **Terminal**: Bold status label for visual emphasis.
- **Connection Form**: Increased dialog size (from 450x400).

### Non-Functional Requirements

- `go build` and `go test` must pass with no regressions.
- Dark and light theme support — both variants must look correct.
- No behavioral changes to existing functionality.
- No new external dependencies beyond Fyne v2.7.3 and Go 1.24.0.
- O(1) color lookups in grid cells for alternating row backgrounds.

### Open Questions

| # | Question |
|---|---|
| 1 | Should toolbar buttons use Fyne's built-in importance levels or custom M3 color overrides? |
| 2 | Should JetBrains Mono apply only to SQL editor monospace or all monospace text? |
| 3 | Is slight increase in sidebar list item height acceptable from HBox layout? |
| 4 | What specific alternating row colors for dark mode? (WCAG 1.4.11 contrast considerations) |
| 5 | Are there planned automated UI tests, or is visual verification the only acceptance criterion? |

---

## Product Analysis

**Product Viability Score: HIGH**

### User Value

Medium-High. While theming alone is not a feature users seek out, it has significant downstream effects on perception and adoption:

- **Alternating row backgrounds** provide a genuine readability improvement when scanning query results — this is functional, not cosmetic.
- **Circle status indicators** replace a Unicode hack with proper visual feedback, improving the connection status experience.
- **Overall polish** reduces the "hobby project" perception that default Fyne theming creates. For a database IDE, professional appearance is a trust signal — users entrust their database credentials to this tool.

### Competitive Context

| Tool | Visual Quality | Notes |
|---|---|---|
| pgAdmin | Medium | Functional but dated |
| DBeaver | Medium | Java Swing/SWT aesthetic |
| DataGrip | High | Best-in-class IDE polish |
| TablePlus | High | Native, clean, modern |
| Helios (current) | Low | Default Fyne — signals "unfinished" |
| Helios (post-M3) | Medium | Competitive with pgAdmin/DBeaver tier |

### Strategic Alignment

Strong. Visual identity is a first-impression multiplier for an early-stage IDE. Users evaluating database tools often dismiss options that look unfinished within seconds. This work moves Helios from the "clearly a prototype" tier to the "credible tool" tier.

### Priority

**P1** — should ship before first public release. The cost of shipping with default Fyne theming is higher than the cost of this work.

### Concerns

- **Fyne limitations vs M3 expectations**: Fyne does not support elevation, ripple effects, or state layers. The scope must be explicitly framed as "M3-inspired" to prevent expectations of full Material Design compliance.
- **Toolbar importance hierarchy**: May be marginal value relative to its complexity. Consider fallback of standard buttons if implementation proves brittle.
- **No automated visual regression testing**: All visual verification is manual, creating risk of unnoticed regressions in future changes.

---

## Technical Discovery

### Current State

- **Zero custom theming**: All UI components use Fyne defaults. No theme files exist in the codebase.
- **Framework**: Fyne v2.7.3 with full theme interface support, Go 1.24.0.
- **Clean architecture**: Clear separation across `db`, `config`, and `ui` packages. All theme-dependent code uses `theme.*` accessors, meaning a custom theme propagates automatically.

### Integration Points

| Component | Current Implementation | Integration Notes |
|---|---|---|
| Toolbar | `widget.Toolbar` returning via `Widget()` | Consumed as `fyne.CanvasObject` in `container.NewBorder()` — replacement with HBox is compatible |
| Sidebar | `CreateItem` returns `widget.NewLabel("template")` | Wrapping in container for circle indicator requires type assertion safety |
| Results Grid | `CreateCell` returns `widget.NewLabel("")` | Alternating backgrounds require container wrapping; type assertion in `UpdateCell` is a risk |
| Completer | Uses `theme.OverlayBackgroundColor()` at line 272 | Custom theme's overlay color propagates automatically |
| Connection Form | Sized at 450x400 | Simple dimension change |
| Terminal Status | Plain `widget.NewLabel("Ready")` | Change to bold text style |

### Theme Propagation

All existing files use `theme.*` accessor functions rather than hardcoded colors. This means the custom `fyne.Theme` implementation will propagate to all components automatically without requiring changes to every file — only components needing widget-level changes (toolbar, sidebar, grid) require direct modification.

---

## Technical Analysis

**Technical Feasibility Score: HIGH**
**Complexity: MEDIUM**

### Recommended Approach

Implement a custom `fyne.Theme` with M3 color lookup tables plus targeted widget wrapping for specific components:

1. **`internal/ui/m3theme.go`**: Implement `fyne.Theme` interface with `Color()`, `Font()`, `Icon()`, `Size()` methods. Use map-based color tables for dark and light variants. Delegate unknown color names to `theme.DefaultTheme()`.
2. **`main.go`**: Set theme via `app.Settings().SetTheme()`.
3. **Widget modifications**: Wrap labels in containers where needed for backgrounds and indicators.

### Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Type assertion panics in `UpdateItem`/`UpdateCell` when wrapping labels in containers | Medium | High | Defensive type checks with graceful fallback; test both create and update paths |
| M3 color choices look poor in one theme variant | Medium | Low | Test both dark and light variants; use established M3 color palettes as reference |
| Fyne limitations on M3 fidelity (no elevation, ripple, state layers) | Certain | Low | Known constraint; scope explicitly as "M3-inspired" |
| No UI tests to catch visual regressions | Certain | Medium | Establish manual testing protocol for both theme variants, empty states, and completer overlay |
| Font bundling adds ~300-400KB to binary | Certain | Low | Acceptable binary size increase for a desktop application |

### Build vs. Buy

| Component | Decision | Rationale |
|---|---|---|
| M3 theme implementation | Build | No purchasable Fyne M3 themes exist |
| Color palette values | Reference | Use published M3 color system values |
| Font (JetBrains Mono) | Buy (bundle) | Free, open-source font |

---

## Strategic Recommendation

**Recommendation: GO**
**Confidence: HIGH**

### Rationale

This is a small, well-scoped enhancement that is strategically necessary for a pre-release product in a competitive market. The codebase is in an ideal state — small, clean, and early enough that changes are cheap. The technical approach is straightforward with well-understood risks. The primary value is not the theming itself but the credibility it lends to the product at first impression.

### Conditions for Proceeding

| # | Condition | Blocking |
|---|---|---|
| 1 | Scope as "M3-inspired" not "M3 compliant" to prevent scope creep. | Yes |
| 2 | Handle type assertion safety for all widget wrapping in `UpdateItem` and `UpdateCell`. | Yes |
| 3 | Manual testing protocol: verify both dark and light themes, empty states, and completer overlay. | Yes |
| 4 | Add smoke-level tests for theme interface (verify `Color()` returns non-nil for all standard names). | No |

### Alternatives Considered

| Alternative | Tradeoff |
|---|---|
| Minimum viable theming (theme file only, no widget changes) | ~80-100 lines, 1-2 files. Gets 60% of visual improvement with 30% of effort. Valid fallback if widget wrapping proves problematic. |
| Defer until post-release | Risks losing early users to poor first impressions. Cost of change increases as codebase grows. |
| Full M3 compliance | Not achievable within Fyne's constraints. Would require framework-level work far beyond scope. |

### Fallback Plan

If widget-level changes (toolbar HBox, sidebar indicators, grid alternating rows) prove too brittle due to type assertion issues, implement only the custom `fyne.Theme` without widget modifications. This "minimum viable theming" approach (~80-100 lines, 1-2 files) delivers the color system and font changes while deferring widget wrapping to a follow-up.

---

## Summary

| Dimension | Score |
|---|---|
| Product Viability | High |
| Technical Feasibility | High |
| Overall Assessment | High |

### Top 3 Risks

1. Type assertion panics when wrapping sidebar and grid widgets in containers.
2. Scope creep toward full M3 compliance beyond Fyne's capabilities.
3. Dark/light theme inconsistency without automated visual regression testing.

### Next Steps

1. Review this research report and confirm the GO decision.
2. Proceed to planning: `/rpi:plan "helios-design"`
3. During planning, define the exact M3 color palette values for both dark and light variants.
4. Establish the manual testing checklist covering both theme variants, empty states, and overlay components.
5. Implement with fallback awareness — if widget wrapping causes issues, fall back to theme-only changes.
