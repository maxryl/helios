package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// ResultsGrid displays query results in a table with bold column headers.
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
			return widget.NewLabel("")
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*widget.Label)
			if id.Row == 0 {
				label.SetText(rg.columns[id.Col])
				label.TextStyle.Bold = true
			} else {
				label.SetText(rg.rows[id.Row-1][id.Col])
				label.TextStyle.Bold = false
			}
			label.Refresh()
		},
	)

	return rg
}

// SetData replaces the displayed columns and rows, then refreshes the table.
func (rg *ResultsGrid) SetData(columns []string, rows [][]string) {
	rg.columns = columns
	rg.rows = rows

	// Adjust column widths based on header length.
	for i, col := range rg.columns {
		w := len(col) * 10
		if w < 100 {
			w = 100
		}
		if w > 300 {
			w = 300
		}
		rg.table.SetColumnWidth(i, float32(w))
	}

	rg.table.Refresh()
}

// Widget returns the underlying table widget for embedding in layouts.
func (rg *ResultsGrid) Widget() *widget.Table {
	return rg.table
}
