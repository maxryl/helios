package ui

import (
	"context"

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
	onNewTerminal func()
}

// NewToolbar creates a toolbar wired to the given terminal tabs.
// The onNewTerminal callback is invoked when the New Terminal button is clicked.
func NewToolbar(tabs *TerminalTabs, window fyne.Window, onNewTerminal func()) *Toolbar {
	tb := &Toolbar{
		tabs:          tabs,
		window:        window,
		onNewTerminal: onNewTerminal,
	}

	runBtn := widget.NewButtonWithIcon("Run", theme.MediaPlayIcon(), tb.runQuery)
	runBtn.Importance = widget.HighImportance

	beginBtn := widget.NewButtonWithIcon("Begin", theme.MediaRecordIcon(), tb.beginTx)
	beginBtn.Importance = widget.MediumImportance

	commitBtn := widget.NewButtonWithIcon("Commit", theme.ConfirmIcon(), tb.commitTx)
	commitBtn.Importance = widget.MediumImportance

	rollbackBtn := widget.NewButtonWithIcon("Rollback", theme.CancelIcon(), tb.rollbackTx)
	rollbackBtn.Importance = widget.MediumImportance

	newTermBtn := widget.NewButtonWithIcon("New", theme.ContentAddIcon(), tb.newTerminal)
	newTermBtn.Importance = widget.LowImportance

	buttons := container.NewHBox(
		runBtn,
		widget.NewSeparator(),
		beginBtn, commitBtn, rollbackBtn,
		widget.NewSeparator(),
		newTermBtn,
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
