package ui

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Toolbar provides action buttons for SQL execution and transaction management.
type Toolbar struct {
	toolbar       *widget.Toolbar
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

	tb.toolbar = widget.NewToolbar(
		widget.NewToolbarAction(theme.MediaPlayIcon(), tb.runQuery),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.MediaRecordIcon(), tb.beginTx),
		widget.NewToolbarAction(theme.ConfirmIcon(), tb.commitTx),
		widget.NewToolbarAction(theme.CancelIcon(), tb.rollbackTx),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.ContentAddIcon(), tb.newTerminal),
	)

	return tb
}

// Widget returns the underlying toolbar widget for embedding in layouts.
func (tb *Toolbar) Widget() *widget.Toolbar {
	return tb.toolbar
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
