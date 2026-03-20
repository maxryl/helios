package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ResultsGrid displays query results in a table with bold column headers
// and alternating row backgrounds for readability.
type ResultsGrid struct {
	table   *widget.Table
	columns []string
	rows    [][]string
}

// NewResultsGrid creates a ResultsGrid backed by a widget.Table.
func NewResultsGrid() *ResultsGrid {
	rg := &ResultsGrid{}

	rg.table = widget.NewTable(
		func() (int, int) {
			if len(rg.columns) == 0 {
				return 0, 0
			}
			return len(rg.rows) + 1, len(rg.columns)
		},
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			label := widget.NewLabel("")
			label.Truncation = fyne.TextTruncateEllipsis
			return container.NewStack(bg, container.NewPadded(label))
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			stack := cell.(*fyne.Container)
			bg := stack.Objects[0].(*canvas.Rectangle)
			padded := stack.Objects[1].(*fyne.Container)
			label := padded.Objects[0].(*widget.Label)
			if id.Row == 0 {
				// Header row: distinct background, bold text.
				label.SetText(rg.columns[id.Col])
				label.TextStyle.Bold = true
				bg.FillColor = theme.Color(theme.ColorNameHeaderBackground)
			} else {
				label.SetText(rg.rows[id.Row-1][id.Col])
				label.TextStyle.Bold = false
				// Alternating row stripes for readability.
				if id.Row%2 == 0 {
					bg.FillColor = theme.Color(colorNameGridStripe)
				} else {
					bg.FillColor = color.Transparent
				}
			}
			bg.Refresh()
			label.Refresh()
		},
	)

	return rg
}

// SetData replaces the displayed columns and rows, then refreshes the table.
func (rg *ResultsGrid) SetData(columns []string, rows [][]string) {
	rg.columns = columns
	rg.rows = rows

	// Compute column widths from header + sample of data rows.
	for i, col := range rg.columns {
		w := len(col)
		// Sample first 50 rows for width estimation.
		for j := 0; j < len(rg.rows) && j < 50; j++ {
			if i < len(rg.rows[j]) && len(rg.rows[j][i]) > w {
				w = len(rg.rows[j][i])
			}
		}
		pw := w * 9
		if pw < 80 {
			pw = 80
		}
		if pw > 350 {
			pw = 350
		}
		rg.table.SetColumnWidth(i, float32(pw))
	}

	rg.table.Refresh()
}

// Widget returns the underlying table widget for embedding in layouts.
func (rg *ResultsGrid) Widget() *widget.Table {
	return rg.table
}
