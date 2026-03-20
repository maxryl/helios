package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Custom color names for Helios-specific roles not in Fyne's standard set.
const (
	colorNameGridStripe fyne.ThemeColorName = "gridStripe"
)

type m3Theme struct {
	dark  map[fyne.ThemeColorName]color.Color
	light map[fyne.ThemeColorName]color.Color
	sizes map[fyne.ThemeSizeName]float32
}

// NewM3Theme returns a Material Design 3 inspired theme for Helios.
func NewM3Theme() fyne.Theme {
	return &m3Theme{
		dark: map[fyne.ThemeColorName]color.Color{
			theme.ColorNamePrimary:             color.NRGBA{R: 0xA8, G: 0xC7, B: 0xFA, A: 0xFF},
			theme.ColorNameForegroundOnPrimary: color.NRGBA{R: 0x06, G: 0x2E, B: 0x6F, A: 0xFF},
			theme.ColorNameButton:              color.NRGBA{R: 0x00, G: 0x4A, B: 0x77, A: 0xFF},
			theme.ColorNameBackground:          color.NRGBA{R: 0x1B, G: 0x1B, B: 0x1F, A: 0xFF},
			theme.ColorNameInputBackground:     color.NRGBA{R: 0x21, G: 0x1F, B: 0x26, A: 0xFF},
			theme.ColorNameHeaderBackground:    color.NRGBA{R: 0x2B, G: 0x29, B: 0x30, A: 0xFF},
			theme.ColorNameMenuBackground:      color.NRGBA{R: 0x36, G: 0x34, B: 0x3B, A: 0xFF},
			theme.ColorNameForeground:          color.NRGBA{R: 0xE6, G: 0xE1, B: 0xE5, A: 0xFF},
			theme.ColorNamePlaceHolder:         color.NRGBA{R: 0xCA, G: 0xC4, B: 0xD0, A: 0xFF},
			theme.ColorNameInputBorder:         color.NRGBA{R: 0x93, G: 0x8F, B: 0x99, A: 0xFF},
			theme.ColorNameSeparator:           color.NRGBA{R: 0x49, G: 0x45, B: 0x4F, A: 0xFF},
			theme.ColorNameError:               color.NRGBA{R: 0xF2, G: 0xB8, B: 0xB5, A: 0xFF},
			theme.ColorNameHover:               color.NRGBA{R: 0xA8, G: 0xC7, B: 0xFA, A: 0x14},
			theme.ColorNamePressed:             color.NRGBA{R: 0xA8, G: 0xC7, B: 0xFA, A: 0x29},
			theme.ColorNameSelection:           color.NRGBA{R: 0x00, G: 0x4A, B: 0x77, A: 0x66},
			theme.ColorNameOverlayBackground:   color.NRGBA{R: 0x1E, G: 0x1B, B: 0x20, A: 0xFF},
			theme.ColorNameFocus:               color.NRGBA{R: 0xA8, G: 0xC7, B: 0xFA, A: 0x4D},
			theme.ColorNameSuccess:             color.NRGBA{R: 0xA8, G: 0xDA, B: 0xB5, A: 0xFF},
			theme.ColorNameWarning:             color.NRGBA{R: 0xFF, G: 0xD5, B: 0x99, A: 0xFF},
			// Visible alternating row color — noticeably lighter than background
			colorNameGridStripe: color.NRGBA{R: 0x2B, G: 0x29, B: 0x30, A: 0xFF},
		},
		light: map[fyne.ThemeColorName]color.Color{
			theme.ColorNamePrimary:             color.NRGBA{R: 0x0B, G: 0x57, B: 0xD0, A: 0xFF},
			theme.ColorNameForegroundOnPrimary: color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
			theme.ColorNameButton:              color.NRGBA{R: 0xD3, G: 0xE3, B: 0xFD, A: 0xFF},
			theme.ColorNameBackground:          color.NRGBA{R: 0xFE, G: 0xFB, B: 0xFF, A: 0xFF},
			theme.ColorNameInputBackground:     color.NRGBA{R: 0xF3, G: 0xED, B: 0xF7, A: 0xFF},
			theme.ColorNameHeaderBackground:    color.NRGBA{R: 0xEC, G: 0xE6, B: 0xF0, A: 0xFF},
			theme.ColorNameMenuBackground:      color.NRGBA{R: 0xE6, G: 0xE0, B: 0xE9, A: 0xFF},
			theme.ColorNameForeground:          color.NRGBA{R: 0x1C, G: 0x1B, B: 0x1F, A: 0xFF},
			theme.ColorNamePlaceHolder:         color.NRGBA{R: 0x49, G: 0x45, B: 0x4F, A: 0xFF},
			theme.ColorNameInputBorder:         color.NRGBA{R: 0x79, G: 0x74, B: 0x7E, A: 0xFF},
			theme.ColorNameSeparator:           color.NRGBA{R: 0xCA, G: 0xC4, B: 0xD0, A: 0xFF},
			theme.ColorNameError:               color.NRGBA{R: 0xB3, G: 0x26, B: 0x1E, A: 0xFF},
			theme.ColorNameHover:               color.NRGBA{R: 0x0B, G: 0x57, B: 0xD0, A: 0x14},
			theme.ColorNamePressed:             color.NRGBA{R: 0x0B, G: 0x57, B: 0xD0, A: 0x29},
			theme.ColorNameSelection:           color.NRGBA{R: 0xD3, G: 0xE3, B: 0xFD, A: 0x99},
			theme.ColorNameOverlayBackground:   color.NRGBA{R: 0xF7, G: 0xF2, B: 0xFA, A: 0xFF},
			theme.ColorNameFocus:               color.NRGBA{R: 0x0B, G: 0x57, B: 0xD0, A: 0x4D},
			theme.ColorNameSuccess:             color.NRGBA{R: 0x1B, G: 0x7F, B: 0x37, A: 0xFF},
			theme.ColorNameWarning:             color.NRGBA{R: 0xE0, G: 0x78, B: 0x00, A: 0xFF},
			// Visible alternating row color for light mode
			colorNameGridStripe: color.NRGBA{R: 0xF3, G: 0xED, B: 0xF7, A: 0xFF},
		},
		sizes: map[fyne.ThemeSizeName]float32{
			theme.SizeNamePadding:         2,
			theme.SizeNameInnerPadding:    4,
			theme.SizeNameInputRadius:     4,
			theme.SizeNameSelectionRadius: 3,
			theme.SizeNameScrollBarRadius: 3,
			theme.SizeNameInlineIcon:      20,
			theme.SizeNameSubHeadingText:  16,
			theme.SizeNameCaptionText:     11,
			theme.SizeNameInputBorder:     1,
		},
	}
}

func (t *m3Theme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	var m map[fyne.ThemeColorName]color.Color
	if variant == theme.VariantLight {
		m = t.light
	} else {
		m = t.dark
	}
	if c, ok := m[name]; ok {
		return c
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *m3Theme) Font(style fyne.TextStyle) fyne.Resource {
	if style.Monospace {
		return jetBrainsMonoRegular
	}
	return theme.DefaultTheme().Font(style)
}

func (t *m3Theme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *m3Theme) Size(name fyne.ThemeSizeName) float32 {
	if s, ok := t.sizes[name]; ok {
		return s
	}
	return theme.DefaultTheme().Size(name)
}
