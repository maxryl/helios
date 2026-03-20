package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2/container"

	"helios/internal/config"
	"helios/internal/db"
)

// TerminalTabs manages a set of SQL terminal tabs, each backed by a database connection.
type TerminalTabs struct {
	tabs      *container.DocTabs
	terminals map[*container.TabItem]*Terminal
	connMgr   *db.ConnectionManager
	schemas   map[string]*db.SchemaCache // config ID -> schema cache
}

// NewTerminalTabs creates a tab container with close-intercept cleanup.
func NewTerminalTabs(connMgr *db.ConnectionManager) *TerminalTabs {
	tt := &TerminalTabs{
		terminals: make(map[*container.TabItem]*Terminal),
		connMgr:   connMgr,
		schemas:   make(map[string]*db.SchemaCache),
	}

	tt.tabs = container.NewDocTabs()
	tt.tabs.CloseIntercept = func(tab *container.TabItem) {
		if terminal, ok := tt.terminals[tab]; ok {
			terminal.Close()
			delete(tt.terminals, tab)
		}
		tt.tabs.Remove(tab)
	}

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

	terminal := NewTerminal(pool, cfg.ID, cfg.Name, onTxChange)

	// Wire autocomplete: reuse or create a schema cache for this connection.
	schema, ok := tt.schemas[cfg.ID]
	if !ok {
		schema = db.NewSchemaCache()
		tt.schemas[cfg.ID] = schema
		// Populate the cache in the background with a dedicated context
		// so it is not tied to the caller's context lifecycle.
		bgPool := pool
		go func() {
			_ = bgPool.Ping(context.Background()) // already verified during Connect
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
	// Trigger CloseIntercept which handles cleanup and removal.
	tt.tabs.CloseIntercept(selected)
}

// CloseAll closes every terminal, rolling back active transactions and cancelling queries.
// This is intended for application shutdown only; it does not remove tabs from the DocTabs widget.
func (tt *TerminalTabs) CloseAll() {
	for tab, terminal := range tt.terminals {
		terminal.Close()
		delete(tt.terminals, tab)
	}
}

// Widget returns the underlying DocTabs for embedding in layouts.
func (tt *TerminalTabs) Widget() *container.DocTabs {
	return tt.tabs
}
