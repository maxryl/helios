package ui

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
)

// ExportCSV writes columns and rows to a CSV file.
func ExportCSV(path string, columns []string, rows [][]string) error {
	return exportDelimited(path, ',', columns, rows)
}

// ExportTSV writes columns and rows to a TSV file.
func ExportTSV(path string, columns []string, rows [][]string) error {
	return exportDelimited(path, '\t', columns, rows)
}

func exportDelimited(path string, delimiter rune, columns []string, rows [][]string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("export: create file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Comma = delimiter

	if err := w.Write(columns); err != nil {
		return fmt.Errorf("export: write header: %w", err)
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return fmt.Errorf("export: write row: %w", err)
		}
	}
	w.Flush()
	return w.Error()
}

// ExportXLSX writes columns and rows to an Excel .xlsx file.
func ExportXLSX(path string, columns []string, rows [][]string) error {
	f := excelize.NewFile()
	sheet := "Sheet1"

	// Header row.
	for j, col := range columns {
		cell, _ := excelize.CoordinatesToCellName(j+1, 1)
		f.SetCellValue(sheet, cell, col)
	}

	// Bold header style.
	style, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	lastCol, _ := excelize.CoordinatesToCellName(len(columns), 1)
	f.SetCellStyle(sheet, "A1", lastCol, style)

	// Data rows.
	for i, row := range rows {
		for j, val := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			f.SetCellValue(sheet, cell, val)
		}
	}

	if err := f.SaveAs(path); err != nil {
		return fmt.Errorf("export: save xlsx: %w", err)
	}
	return nil
}
