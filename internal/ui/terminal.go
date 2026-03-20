package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"helios/internal/db"
)

// sqlEditor is a multiline Entry that intercepts Ctrl+Enter to run queries
// and arrow/enter keys to navigate autocomplete suggestions.
type sqlEditor struct {
	widget.Entry
	onCtrlEnter func()
	completer   *Completer
}

func newSQLEditor(onCtrlEnter func()) *sqlEditor {
	e := &sqlEditor{onCtrlEnter: onCtrlEnter}
	e.MultiLine = true
	e.Wrapping = fyne.TextWrapOff
	e.TextStyle.Monospace = true
	e.ExtendBaseWidget(e)
	return e
}

func (e *sqlEditor) TypedShortcut(s fyne.Shortcut) {
	if cs, ok := s.(*desktop.CustomShortcut); ok {
		if cs.KeyName == fyne.KeyReturn && cs.Modifier == fyne.KeyModifierControl {
			if e.onCtrlEnter != nil {
				e.onCtrlEnter()
			}
			return
		}
	}
	e.Entry.TypedShortcut(s)
}

func (e *sqlEditor) TypedKey(ev *fyne.KeyEvent) {
	if e.completer != nil && e.completer.visible {
		switch ev.Name {
		case fyne.KeyDown:
			e.completer.SelectNext()
			return
		case fyne.KeyUp:
			e.completer.SelectPrevious()
			return
		case fyne.KeyReturn, fyne.KeyEnter:
			// Accept the completion and swallow the key so that
			// Entry does not insert a newline.
			e.completer.AcceptSelected()
			return
		case fyne.KeyEscape:
			e.completer.Dismiss()
			return
		case fyne.KeyTab:
			e.completer.AcceptSelected()
			return
		}
	}
	e.Entry.TypedKey(ev)
}

// TxState represents the transaction state of a terminal.
type TxState int

const (
	// TxNone means no transaction is active.
	TxNone TxState = iota
	// TxActive means a transaction is in progress.
	TxActive
)

// Terminal is a SQL editing and execution pane backed by a single database connection.
type Terminal struct {
	mu           sync.Mutex
	editor       *widget.Entry  // reference for text access (Text, SelectedText, OnChanged)
	editorWidget fyne.CanvasObject // the actual widget used in layout (sqlEditor)
	results      *ResultsGrid
	statusLabel  *widget.Label
	window       fyne.Window
	history      *QueryHistory
	pool         *pgxpool.Pool
	tx           pgx.Tx
	txState      TxState
	cancel       context.CancelFunc
	configID     string
	configName   string
	onTxChange   func()
	completer       *Completer
	completerHolder *fyne.Container
	content         fyne.CanvasObject
}

// NewTerminal creates a Terminal bound to the given connection pool.
// The onTxChange callback is invoked whenever the transaction state changes.
func NewTerminal(pool *pgxpool.Pool, configID, configName string, window fyne.Window, history *QueryHistory, onTxChange func()) *Terminal {
	t := &Terminal{
		pool:       pool,
		configID:   configID,
		configName: configName,
		window:     window,
		history:    history,
		onTxChange: onTxChange,
	}

	se := newSQLEditor(func() { t.RunQuery() })
	se.SetPlaceHolder("Enter SQL...")
	t.editor = &se.Entry
	t.editorWidget = se

	t.results = NewResultsGrid()
	t.statusLabel = widget.NewLabel("Ready")
	t.statusLabel.TextStyle.Bold = true
	t.completerHolder = container.NewStack()

	// Top pane: editor with completer suggestions below it.
	editorWithCompleter := container.NewBorder(nil, t.completerHolder, nil, nil, t.editorWidget)

	csvBtn := widget.NewButton("CSV", func() { t.exportAs("csv") })
	csvBtn.Importance = widget.LowImportance
	tsvBtn := widget.NewButton("TSV", func() { t.exportAs("tsv") })
	tsvBtn.Importance = widget.LowImportance
	xlsxBtn := widget.NewButton("XLSX", func() { t.exportAs("xlsx") })
	xlsxBtn.Importance = widget.LowImportance

	exportButtons := container.NewHBox(csvBtn, tsvBtn, xlsxBtn)

	statusBg := canvas.NewRectangle(theme.Color(theme.ColorNameHeaderBackground))
	statusContent := container.NewHBox(t.statusLabel, layout.NewSpacer(), exportButtons)
	statusBar := container.NewStack(statusBg, container.NewPadded(statusContent))
	resultsPane := container.NewBorder(statusBar, nil, nil, nil, t.results.Widget())
	split := container.NewVSplit(editorWithCompleter, resultsPane)
	split.SetOffset(0.3)
	t.content = split

	return t
}

func (t *Terminal) exportAs(format string) {
	cols := t.results.columns
	rows := t.results.rows
	if len(cols) == 0 {
		dialog.ShowInformation("No Data", "Run a query first.", t.window)
		return
	}

	var ext string
	switch format {
	case "csv":
		ext = ".csv"
	case "tsv":
		ext = ".tsv"
	case "xlsx":
		ext = ".xlsx"
	}

	dlg := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}
		path := writer.URI().Path()
		writer.Close()

		var exportErr error
		switch format {
		case "csv":
			exportErr = ExportCSV(path, cols, rows)
		case "tsv":
			exportErr = ExportTSV(path, cols, rows)
		case "xlsx":
			exportErr = ExportXLSX(path, cols, rows)
		}
		if exportErr != nil {
			fyne.Do(func() {
				dialog.ShowError(exportErr, t.window)
			})
		} else {
			fyne.Do(func() {
				dialog.ShowInformation("Export Complete",
					fmt.Sprintf("Saved %d rows to %s", len(rows), path), t.window)
			})
		}
	}, t.window)

	dlg.SetFilter(storage.NewExtensionFileFilter([]string{ext}))
	dlg.SetFileName("export" + ext)
	dlg.Show()
}

// Content returns the pre-built canvas object for embedding in tabs.
func (t *Terminal) Content() fyne.CanvasObject {
	return t.content
}

// RunQuery executes the selected text (or the full editor contents) against the database.
// Results are streamed in batches so the UI stays responsive for large result sets.
func (t *Terminal) RunQuery() {
	sql := t.editor.SelectedText()
	if sql == "" {
		sql = t.editor.Text
	}
	if strings.TrimSpace(sql) == "" {
		return
	}

	t.mu.Lock()
	if t.cancel != nil {
		t.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	querier := t.querier()
	t.mu.Unlock()

	t.statusLabel.SetText("Running...")
	if t.history != nil {
		t.history.Add(sql, t.configName)
	}

	if !db.IsQuerySQL(sql) {
		// Non-SELECT: use simple exec path.
		go func() {
			result := db.ExecuteQuery(ctx, querier, sql)
			fyne.Do(func() {
				if result.Error != nil {
					t.statusLabel.SetText(fmt.Sprintf("Error: %s", result.Error))
				} else {
					t.statusLabel.SetText(fmt.Sprintf("%s (%s)", result.Message, result.Duration.Round(time.Millisecond)))
				}
				t.results.SetData(nil, nil)
			})
		}()
		return
	}

	// SELECT: stream results in batches.
	go func() {
		var allRows [][]string
		firstBatch := true

		result := db.ExecuteQueryStreaming(ctx, querier, sql, 500, func(columns []string, batch [][]string, totalSoFar int) bool {
			allRows = append(allRows, batch...)

			if firstBatch {
				// Show first batch immediately for instant feedback.
				firstBatch = false
				snapshot := make([][]string, len(allRows))
				copy(snapshot, allRows)
				cols := columns
				fyne.Do(func() {
					t.results.SetData(cols, snapshot)
					t.statusLabel.SetText(fmt.Sprintf("Loading... %d rows", totalSoFar))
				})
			} else {
				// Update status during loading.
				fyne.Do(func() {
					t.statusLabel.SetText(fmt.Sprintf("Loading... %d rows", totalSoFar))
				})
			}
			return true
		})

		// Final update with all data.
		fyne.Do(func() {
			if result.Error != nil {
				t.statusLabel.SetText(fmt.Sprintf("Error: %s", result.Error))
				t.results.SetData(nil, nil)
				return
			}

			if result.RowCount > 0 {
				t.results.SetData(result.Columns, allRows)
				status := fmt.Sprintf("%d rows (%s)", result.RowCount, result.Duration.Round(time.Millisecond))
				if result.Truncated {
					status += fmt.Sprintf(" — limited to %d rows", db.MaxRows)
				}
				t.statusLabel.SetText(status)
			} else {
				t.results.SetData(nil, nil)
				t.statusLabel.SetText(fmt.Sprintf("0 rows (%s)", result.Duration.Round(time.Millisecond)))
			}
		})
	}()
}

// BeginTx starts a new transaction. Returns an error if one is already active.
func (t *Terminal) BeginTx(ctx context.Context) error {
	t.mu.Lock()
	if t.txState != TxNone {
		t.mu.Unlock()
		return fmt.Errorf("ui: begin tx: transaction already active")
	}
	tx, err := t.pool.Begin(ctx)
	if err != nil {
		t.mu.Unlock()
		return fmt.Errorf("ui: begin tx: %w", err)
	}
	t.tx = tx
	t.txState = TxActive
	cb := t.onTxChange
	t.mu.Unlock()
	if cb != nil {
		cb()
	}
	return nil
}

// CommitTx commits the active transaction. Returns an error if no transaction is active.
func (t *Terminal) CommitTx(ctx context.Context) error {
	t.mu.Lock()
	if t.txState != TxActive {
		t.mu.Unlock()
		return fmt.Errorf("ui: commit tx: no active transaction")
	}
	if err := t.tx.Commit(ctx); err != nil {
		t.mu.Unlock()
		return fmt.Errorf("ui: commit tx: %w", err)
	}
	t.tx = nil
	t.txState = TxNone
	cb := t.onTxChange
	t.mu.Unlock()
	if cb != nil {
		cb()
	}
	return nil
}

// RollbackTx rolls back the active transaction. Returns an error if no transaction is active.
func (t *Terminal) RollbackTx(ctx context.Context) error {
	t.mu.Lock()
	if t.txState != TxActive {
		t.mu.Unlock()
		return fmt.Errorf("ui: rollback tx: no active transaction")
	}
	if err := t.tx.Rollback(ctx); err != nil {
		t.mu.Unlock()
		return fmt.Errorf("ui: rollback tx: %w", err)
	}
	t.tx = nil
	t.txState = TxNone
	cb := t.onTxChange
	t.mu.Unlock()
	if cb != nil {
		cb()
	}
	return nil
}

// Close cleans up the terminal by rolling back any active transaction and cancelling running queries.
func (t *Terminal) Close() {
	t.mu.Lock()
	var cb func()
	if t.txState == TxActive {
		_ = t.tx.Rollback(context.Background())
		t.tx = nil
		t.txState = TxNone
		cb = t.onTxChange
	}
	if t.cancel != nil {
		t.cancel()
	}
	t.mu.Unlock()
	if cb != nil {
		cb()
	}
}

// querier returns the active transaction if one exists, otherwise the pool.
func (t *Terminal) querier() db.Querier {
	if t.txState == TxActive {
		return t.tx
	}
	return t.pool
}

// TxState returns the current transaction state.
func (t *Terminal) TxState() TxState {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.txState
}

// ConfigName returns the connection name associated with this terminal.
func (t *Terminal) ConfigName() string {
	return t.configName
}

// SetCompleter wires autocomplete to this terminal's editor.
func (t *Terminal) SetCompleter(schema *db.SchemaCache) {
	t.completer = NewCompleter(t.editor, schema)
	t.completerHolder.Add(t.completer.Widget())
	// Give the sqlEditor a reference so it can forward arrow/enter keys.
	if se, ok := t.editorWidget.(*sqlEditor); ok {
		se.completer = t.completer
	}
	t.editor.OnChanged = func(text string) {
		t.completer.OnTextChanged(text)
	}
}
