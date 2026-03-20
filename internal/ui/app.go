package ui

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"

	"helios/internal/config"
	"helios/internal/db"
)

// App is the top-level Helios application window.
type App struct {
	fyneApp fyne.App
	window  fyne.Window
	config  *config.AppConfig
	cfgPath string
	connMgr *db.ConnectionManager
	tabs    *TerminalTabs
	toolbar *Toolbar
	sidebar *Sidebar
}

// NewApp creates the main Helios window with menus, shortcuts, and a tab container.
func NewApp(fyneApp fyne.App, cfg *config.AppConfig, cfgPath string, connMgr *db.ConnectionManager) *App {
	a := &App{
		fyneApp: fyneApp,
		config:  cfg,
		cfgPath: cfgPath,
		connMgr: connMgr,
	}

	a.window = fyneApp.NewWindow("Helios")
	a.tabs = NewTerminalTabs(connMgr)

	a.sidebar = NewSidebar(cfg, connMgr,
		// onSelect: open a new terminal tab for the chosen connection.
		func(cc config.ConnectionConfig) {
			if err := a.tabs.NewTerminal(context.Background(), cc); err != nil {
				dialog.ShowError(err, a.window)
			}
		},
		// onEdit: show connection dialog for add (empty ID) or edit (existing ID).
		func(cc config.ConnectionConfig) {
			var existing *config.ConnectionConfig
			if cc.ID != "" {
				existing = &cc
			}
			ShowConnectionDialog(a.window, existing, func(updated config.ConnectionConfig) {
				if updated.ID == "" {
					a.config.Add(updated)
				} else {
					if err := a.config.Update(updated); err != nil {
						dialog.ShowError(err, a.window)
						return
					}
				}
				if err := a.config.Save(a.cfgPath); err != nil {
					dialog.ShowError(err, a.window)
				}
				a.sidebar.Refresh()
			})
		},
		// onDelete: disconnect, remove from config, and persist.
		func(id string) {
			dialog.ShowConfirm("Delete Connection", "Are you sure you want to delete this connection?", func(ok bool) {
				if !ok {
					return
				}
				_ = a.connMgr.Disconnect(id)
				if err := a.config.Remove(id); err != nil {
					dialog.ShowError(err, a.window)
					return
				}
				if err := a.config.Save(a.cfgPath); err != nil {
					dialog.ShowError(err, a.window)
				}
				a.sidebar.Refresh()
			}, a.window)
		},
	)

	a.toolbar = NewToolbar(a.tabs, a.window, a.openTerminalForFirstConnection)

	hsplit := container.NewHSplit(a.sidebar.Widget(), a.tabs.Widget())
	hsplit.SetOffset(0.2)

	content := container.NewBorder(a.toolbar.Widget(), nil, nil, nil, hsplit)
	a.window.SetContent(content)

	a.window.SetMainMenu(a.makeMenu())
	a.addShortcuts()

	a.window.SetOnClosed(func() {
		a.tabs.CloseAll()
		connMgr.CloseAll()
	})

	a.window.Resize(fyne.NewSize(1200, 800))

	return a
}

// Show displays the window and starts the Fyne event loop.
func (a *App) Show() {
	a.window.ShowAndRun()
}

// makeMenu builds the application main menu.
func (a *App) makeMenu() *fyne.MainMenu {
	newConn := fyne.NewMenuItem("New Connection", func() {
		ShowConnectionDialog(a.window, nil, func(cfg config.ConnectionConfig) {
			a.config.Add(cfg)
			if err := a.config.Save(a.cfgPath); err != nil {
				dialog.ShowError(err, a.window)
			}
			a.sidebar.Refresh()
		})
	})
	newTerm := fyne.NewMenuItem("New Terminal", func() {
		a.openTerminalForFirstConnection()
	})
	quit := fyne.NewMenuItem("Quit", func() {
		a.fyneApp.Quit()
	})

	fileMenu := fyne.NewMenu("File", newConn, newTerm, fyne.NewMenuItemSeparator(), quit)
	return fyne.NewMainMenu(fileMenu)
}

// addShortcuts registers keyboard shortcuts on the window canvas.
func (a *App) addShortcuts() {
	canvas := a.window.Canvas()

	// Ctrl+Enter: run query in active terminal.
	canvas.AddShortcut(
		&desktop.CustomShortcut{KeyName: fyne.KeyReturn, Modifier: fyne.KeyModifierControl},
		func(_ fyne.Shortcut) {
			if t := a.tabs.ActiveTerminal(); t != nil {
				t.RunQuery()
			}
		},
	)

	// Ctrl+T: new terminal for the first saved connection.
	canvas.AddShortcut(
		&desktop.CustomShortcut{KeyName: fyne.KeyT, Modifier: fyne.KeyModifierControl},
		func(_ fyne.Shortcut) {
			a.openTerminalForFirstConnection()
		},
	)

	// Ctrl+W: close current tab.
	canvas.AddShortcut(
		&desktop.CustomShortcut{KeyName: fyne.KeyW, Modifier: fyne.KeyModifierControl},
		func(_ fyne.Shortcut) {
			a.tabs.CloseActive()
		},
	)
}

// openTerminalForFirstConnection opens a terminal tab for the first saved connection.
// If no connections are saved, a dialog prompts the user to create one first.
func (a *App) openTerminalForFirstConnection() {
	if len(a.config.Connections) == 0 {
		dialog.ShowInformation("No Connections", "Add a connection first.", a.window)
		return
	}
	cfg := a.config.Connections[0]
	if err := a.tabs.NewTerminal(context.Background(), cfg); err != nil {
		dialog.ShowError(err, a.window)
	}
}
