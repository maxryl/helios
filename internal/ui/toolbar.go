package ui

import (
	"context"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Toolbar provides action buttons for SQL execution and transaction management.
type Toolbar struct {
	widget        fyne.CanvasObject
	tabs          *TerminalTabs
	window        fyne.Window
	history       *QueryHistory
	onNewTerminal func()
}

// NewToolbar creates a toolbar wired to the given terminal tabs.
// The onNewTerminal callback is invoked when the New Terminal button is clicked.
func NewToolbar(tabs *TerminalTabs, window fyne.Window, history *QueryHistory, onNewTerminal func()) *Toolbar {
	tb := &Toolbar{
		tabs:          tabs,
		window:        window,
		history:       history,
		onNewTerminal: onNewTerminal,
	}

	runBtn := widget.NewButtonWithIcon("Run", theme.MediaPlayIcon(), tb.runQuery)
	runBtn.Importance = widget.HighImportance

	explainBtn := widget.NewButtonWithIcon("Explain", theme.InfoIcon(), tb.explainQuery)
	explainBtn.Importance = widget.MediumImportance

	beginBtn := widget.NewButtonWithIcon("Begin", theme.MediaRecordIcon(), tb.beginTx)
	beginBtn.Importance = widget.MediumImportance

	commitBtn := widget.NewButtonWithIcon("Commit", theme.ConfirmIcon(), tb.commitTx)
	commitBtn.Importance = widget.MediumImportance

	rollbackBtn := widget.NewButtonWithIcon("Rollback", theme.CancelIcon(), tb.rollbackTx)
	rollbackBtn.Importance = widget.MediumImportance

	newTermBtn := widget.NewButtonWithIcon("New", theme.ContentAddIcon(), tb.newTerminal)
	newTermBtn.Importance = widget.LowImportance

	historyBtn := widget.NewButtonWithIcon("History", theme.HistoryIcon(), tb.showHistory)
	historyBtn.Importance = widget.LowImportance

	buttons := container.NewHBox(
		runBtn,
		explainBtn,
		widget.NewSeparator(),
		beginBtn, commitBtn, rollbackBtn,
		widget.NewSeparator(),
		newTermBtn,
		historyBtn,
		layout.NewSpacer(),
	)

	separator := widget.NewSeparator()
	tb.widget = container.NewVBox(
		container.NewPadded(buttons),
		separator,
	)

	return tb
}

// Widget returns the toolbar for embedding in layouts.
func (tb *Toolbar) Widget() fyne.CanvasObject {
	return tb.widget
}

func (tb *Toolbar) activeOrError() *Terminal {
	t := tb.tabs.ActiveTerminal()
	if t == nil {
		dialog.ShowInformation("No Terminal", "Open a terminal tab first.", tb.window)
	}
	return t
}

func (tb *Toolbar) runQuery() {
	if t := tb.activeOrError(); t != nil {
		t.RunQuery()
	}
}

func (tb *Toolbar) explainQuery() {
	if t := tb.activeOrError(); t != nil {
		RunExplainAnalyze(t)
	}
}

func (tb *Toolbar) beginTx() {
	t := tb.activeOrError()
	if t == nil {
		return
	}
	if err := t.BeginTx(context.Background()); err != nil {
		dialog.ShowError(err, tb.window)
	}
}

func (tb *Toolbar) commitTx() {
	t := tb.activeOrError()
	if t == nil {
		return
	}
	if err := t.CommitTx(context.Background()); err != nil {
		dialog.ShowError(err, tb.window)
	}
}

func (tb *Toolbar) rollbackTx() {
	t := tb.activeOrError()
	if t == nil {
		return
	}
	if err := t.RollbackTx(context.Background()); err != nil {
		dialog.ShowError(err, tb.window)
	}
}

func (tb *Toolbar) newTerminal() {
	if tb.onNewTerminal != nil {
		tb.onNewTerminal()
	}
}

func (tb *Toolbar) showHistory() {
	entries := tb.history.Entries()

	list := widget.NewList(
		func() int { return len(entries) },
		func() fyne.CanvasObject {
			return container.NewVBox(
				widget.NewLabel("query"),
				widget.NewLabel("meta"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			box := obj.(*fyne.Container)
			sqlLabel := box.Objects[0].(*widget.Label)
			metaLabel := box.Objects[1].(*widget.Label)
			e := entries[id]
			// Truncate long queries for display.
			preview := strings.ReplaceAll(e.SQL, "\n", " ")
			if len(preview) > 120 {
				preview = preview[:120] + "…"
			}
			sqlLabel.SetText(preview)
			sqlLabel.TextStyle.Monospace = true
			metaLabel.SetText(fmt.Sprintf("%s — %s", e.ConnName, e.Timestamp.Format("2006-01-02 15:04:05")))
		},
	)

	var d dialog.Dialog

	list.OnSelected = func(id widget.ListItemID) {
		// Paste the query into the active terminal's editor.
		if t := tb.tabs.ActiveTerminal(); t != nil {
			t.editor.SetText(entries[id].SQL)
		}
		d.Hide()
	}

	clearBtn := widget.NewButton("Clear History", func() {
		tb.history.Clear()
		d.Hide()
	})
	clearBtn.Importance = widget.DangerImportance

	closeBtn := widget.NewButton("Close", func() {
		d.Hide()
	})

	buttons := container.NewHBox(clearBtn, layout.NewSpacer(), closeBtn)

	scroll := container.NewStack(list)
	content := container.NewBorder(nil, buttons, nil, nil, scroll)

	d = dialog.NewCustomWithoutButtons("Query History", content, tb.window)
	d.Resize(fyne.NewSize(700, 500))
	d.Show()
}
