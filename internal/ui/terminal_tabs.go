package ui

import (
	"context"
	"fmt"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"

	"helios/internal/config"
	"helios/internal/db"
)

// TerminalTabs manages SQL terminal tabs and file editor tabs.
type TerminalTabs struct {
	tabs      *container.DocTabs
	terminals map[*container.TabItem]*Terminal
	editors   map[*container.TabItem]*FileEditor
	connMgr   *db.ConnectionManager
	window    fyne.Window
	history   *QueryHistory
	schemas   map[string]*db.SchemaCache
}

// NewTerminalTabs creates a tab container with close-intercept cleanup.
func NewTerminalTabs(connMgr *db.ConnectionManager, window fyne.Window, history *QueryHistory) *TerminalTabs {
	tt := &TerminalTabs{
		terminals: make(map[*container.TabItem]*Terminal),
		editors:   make(map[*container.TabItem]*FileEditor),
		connMgr:   connMgr,
		window:    window,
		history:   history,
		schemas:   make(map[string]*db.SchemaCache),
	}

	tt.tabs = container.NewDocTabs()
	tt.tabs.CloseIntercept = func(tab *container.TabItem) {
		if terminal, ok := tt.terminals[tab]; ok {
			terminal.Close()
			delete(tt.terminals, tab)
			tt.tabs.Remove(tab)
			return
		}
		if fe, ok := tt.editors[tab]; ok {
			fe.ConfirmClose(func() {
				delete(tt.editors, tab)
				tt.tabs.Remove(tab)
			})
			return
		}
		tt.tabs.Remove(tab)
	}

	// Ctrl+S to save active file editor.
	window.Canvas().AddShortcut(
		&desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierControl},
		func(_ fyne.Shortcut) {
			tt.SaveActiveEditor()
		},
	)

	return tt
}

// NewTerminal opens a new SQL terminal tab connected to the given configuration.
func (tt *TerminalTabs) NewTerminal(ctx context.Context, cfg config.ConnectionConfig) error {
	pool, err := tt.connMgr.Connect(ctx, cfg)
	if err != nil {
		return fmt.Errorf("ui: new terminal: %w", err)
	}

	var tab *container.TabItem

	onTxChange := func() {
		if tab == nil {
			return
		}
		terminal := tt.terminals[tab]
		if terminal == nil {
			return
		}
		tab.Text = terminal.ConfigName()
		if terminal.TxState() == TxActive {
			tab.Text += " [TX]"
		}
		tt.tabs.Refresh()
	}

	terminal := NewTerminal(pool, cfg.ID, cfg.Name, tt.window, tt.history, onTxChange)

	schema, ok := tt.schemas[cfg.ID]
	if !ok {
		schema = db.NewSchemaCache()
		tt.schemas[cfg.ID] = schema
		bgPool := pool
		go func() {
			_ = bgPool.Ping(context.Background())
			_ = schema.RefreshSchema(context.Background(), bgPool)
		}()
	}
	terminal.SetCompleter(schema)

	tab = container.NewTabItem(cfg.Name, terminal.Content())
	tt.terminals[tab] = terminal
	tt.tabs.Append(tab)
	tt.tabs.Select(tab)

	return nil
}

// OpenFile opens a file in a new editor tab, or focuses it if already open.
func (tt *TerminalTabs) OpenFile(path string) {
	// Check if already open.
	for tab, fe := range tt.editors {
		if fe.Path() == path {
			tt.tabs.Select(tab)
			return
		}
	}

	var tab *container.TabItem
	fe, err := NewFileEditor(path, tt.window, func(dirty bool) {
		if tab == nil {
			return
		}
		name := filepath.Base(path)
		if dirty {
			name = "● " + name
		}
		tab.Text = name
		tt.tabs.Refresh()
	})
	if err != nil {
		dialog.ShowError(err, tt.window)
		return
	}

	tab = container.NewTabItem(filepath.Base(path), fe.Content())
	tt.editors[tab] = fe
	tt.tabs.Append(tab)
	tt.tabs.Select(tab)
}

// SaveActiveEditor saves the currently active file editor, if any.
func (tt *TerminalTabs) SaveActiveEditor() {
	selected := tt.tabs.Selected()
	if selected == nil {
		return
	}
	if fe, ok := tt.editors[selected]; ok {
		if err := fe.Save(); err != nil {
			dialog.ShowError(err, tt.window)
		}
	}
}

// ActiveTerminal returns the terminal for the currently selected tab, or nil.
func (tt *TerminalTabs) ActiveTerminal() *Terminal {
	selected := tt.tabs.Selected()
	if selected == nil {
		return nil
	}
	return tt.terminals[selected]
}

// CloseActive closes the currently selected tab, triggering cleanup via CloseIntercept.
func (tt *TerminalTabs) CloseActive() {
	selected := tt.tabs.Selected()
	if selected == nil {
		return
	}
	tt.tabs.CloseIntercept(selected)
}

// CloseAll closes every terminal, rolling back active transactions and cancelling queries.
func (tt *TerminalTabs) CloseAll() {
	for tab, terminal := range tt.terminals {
		terminal.Close()
		delete(tt.terminals, tab)
	}
}

// NewTerminalWithText opens a new terminal and sets its editor content.
func (tt *TerminalTabs) NewTerminalWithText(ctx context.Context, cfg config.ConnectionConfig, text string) error {
	if err := tt.NewTerminal(ctx, cfg); err != nil {
		return err
	}
	if t := tt.ActiveTerminal(); t != nil {
		t.editor.SetText(text)
	}
	return nil
}

// Widget returns the underlying DocTabs for embedding in layouts.
func (tt *TerminalTabs) Widget() *container.DocTabs {
	return tt.tabs
}
