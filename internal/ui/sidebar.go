package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"helios/internal/config"
	"helios/internal/db"
)

// errRequiredField is returned when a required form field is left empty.
var errRequiredField = errors.New("ui: Name and Host are required")

// Tree node path separator. Using \t to avoid conflicts with identifiers.
const sep = "\t"

// Sidebar presents a schema browser tree with expandable connections.
type Sidebar struct {
	tree      *widget.Tree
	config    *config.AppConfig
	connMgr   *db.ConnectionManager
	window    fyne.Window
	onSelect          func(config.ConnectionConfig)
	onEdit            func(config.ConnectionConfig)
	onDelete          func(string)
	onOpenWithText    func(config.ConnectionConfig, string)
	onImport          func([]config.ConnectionConfig) // import connections from JSON
	container         fyne.CanvasObject

	mu             sync.RWMutex
	metaByID       map[string]*db.DatabaseMeta // connID -> metadata (schemas only at first)
	lastSelectedID string
}

// NewSidebar creates a sidebar with a toolbar and schema browser tree.
func NewSidebar(
	cfg *config.AppConfig,
	connMgr *db.ConnectionManager,
	window fyne.Window,
	onSelect func(config.ConnectionConfig),
	onEdit func(config.ConnectionConfig),
	onDelete func(string),
	onOpenWithText func(config.ConnectionConfig, string),
	onImport func([]config.ConnectionConfig),
) *Sidebar {
	s := &Sidebar{
		config:         cfg,
		connMgr:        connMgr,
		window:         window,
		onSelect:       onSelect,
		onEdit:         onEdit,
		onDelete:       onDelete,
		onOpenWithText: onOpenWithText,
		onImport:       onImport,
		metaByID:       make(map[string]*db.DatabaseMeta),
	}

	s.tree = widget.NewTree(
		s.childUIDs,
		s.isBranch,
		s.createNode,
		s.updateNode,
	)

	s.tree.OnSelected = func(uid widget.TreeNodeID) {
		parts := strings.Split(uid, sep)
		connID := parts[0]
		s.lastSelectedID = connID

		switch {
		case len(parts) == 1:
			// Connection click -> open terminal
			for _, conn := range s.config.Connections {
				if conn.ID == connID {
					s.onSelect(conn)
					break
				}
			}
		case len(parts) == 4 && parts[2] == "Functions":
			// Function click -> show definition popup
			schemaName := parts[1]
			funcName := parts[3]
			go s.showFunctionDetail(connID, schemaName, funcName)
		}
	}

	s.tree.OnBranchOpened = func(uid widget.TreeNodeID) {
		parts := strings.Split(uid, sep)
		connID := parts[0]

		switch len(parts) {
		case 1:
			// Connection expanded -> connect if needed, then fetch schemas.
			if !s.connMgr.IsConnected(connID) {
				// Force a connection by opening a terminal.
				for _, conn := range s.config.Connections {
					if conn.ID == connID {
						s.onSelect(conn)
						break
					}
				}
			}
			s.mu.RLock()
			_, loaded := s.metaByID[connID]
			s.mu.RUnlock()
			if !loaded {
				go s.loadSchemas(connID)
			}
		case 2:
			// Schema expanded -> fetch tables + functions for this schema
			schemaName := parts[1]
			s.mu.RLock()
			meta := s.metaByID[connID]
			s.mu.RUnlock()
			if meta != nil {
				schema := s.findSchema(meta, schemaName)
				if schema != nil && !schema.Loaded {
					go s.loadSchemaContent(connID, schema)
				}
			}
		case 4:
			// Table expanded -> fetch columns/indexes/constraints/triggers
			if parts[2] != "Tables" {
				return
			}
			schemaName := parts[1]
			tableName := parts[3]
			s.mu.RLock()
			meta := s.metaByID[connID]
			s.mu.RUnlock()
			if meta != nil {
				schema := s.findSchema(meta, schemaName)
				if schema != nil {
					table := s.findTable(schema, tableName)
					if table != nil && !table.Loaded {
						go s.loadTableDetail(connID, schemaName, table)
					}
				}
			}
		}
	}

	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentAddIcon(), func() {
			s.onEdit(config.ConnectionConfig{})
		}),
		widget.NewToolbarAction(theme.DocumentCreateIcon(), func() {
			if s.lastSelectedID == "" {
				return
			}
			for _, conn := range s.config.Connections {
				if conn.ID == s.lastSelectedID {
					s.onEdit(conn)
					return
				}
			}
		}),
		widget.NewToolbarAction(theme.DeleteIcon(), func() {
			if s.lastSelectedID != "" {
				s.onDelete(s.lastSelectedID)
				s.lastSelectedID = ""
			}
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.DownloadIcon(), func() {
			s.importConnections()
		}),
	)

	header := widget.NewLabel("Connections")
	header.TextStyle.Bold = true

	rightBorder := widget.NewSeparator()
	content := container.NewBorder(container.NewVBox(header, toolbar), nil, nil, nil, s.tree)
	s.container = container.NewBorder(nil, nil, nil, rightBorder, content)
	return s
}

// loadSchemas fetches only schema names for a connection.
func (s *Sidebar) loadSchemas(connID string) {
	pool := s.connMgr.Pool(connID)
	if pool == nil {
		return
	}
	schemas, err := db.FetchSchemas(context.Background(), pool)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.metaByID[connID] = &db.DatabaseMeta{Schemas: schemas}
	s.mu.Unlock()
	fyne.Do(func() { s.tree.Refresh() })
}

// loadSchemaContent fetches tables + functions for one schema.
func (s *Sidebar) loadSchemaContent(connID string, schema *db.SchemaMeta) {
	pool := s.connMgr.Pool(connID)
	if pool == nil {
		return
	}
	if err := db.FetchSchemaContent(context.Background(), pool, schema); err != nil {
		return
	}
	fyne.Do(func() { s.tree.Refresh() })
}

// loadTableDetail fetches columns/indexes/constraints/triggers for one table.
func (s *Sidebar) loadTableDetail(connID, schemaName string, table *db.TableMeta) {
	pool := s.connMgr.Pool(connID)
	if pool == nil {
		return
	}
	if err := db.FetchTableDetail(context.Background(), pool, schemaName, table); err != nil {
		return
	}
	fyne.Do(func() { s.tree.Refresh() })
}

// childUIDs returns the children of a tree node.
func (s *Sidebar) childUIDs(uid widget.TreeNodeID) []widget.TreeNodeID {
	if uid == "" {
		ids := make([]widget.TreeNodeID, len(s.config.Connections))
		for i, conn := range s.config.Connections {
			ids[i] = conn.ID
		}
		return ids
	}

	parts := strings.Split(uid, sep)
	connID := parts[0]

	s.mu.RLock()
	meta := s.metaByID[connID]
	s.mu.RUnlock()

	switch len(parts) {
	case 1:
		if meta == nil {
			return nil
		}
		ids := make([]widget.TreeNodeID, len(meta.Schemas))
		for i, schema := range meta.Schemas {
			ids[i] = uid + sep + schema.Name
		}
		return ids

	case 2:
		return []widget.TreeNodeID{
			uid + sep + "Tables",
			uid + sep + "Functions",
		}

	case 3:
		schemaName := parts[1]
		category := parts[2]
		schema := s.findSchema(meta, schemaName)
		if schema == nil || !schema.Loaded {
			return nil
		}
		switch category {
		case "Tables":
			ids := make([]widget.TreeNodeID, len(schema.Tables))
			for i, t := range schema.Tables {
				ids[i] = uid + sep + t.Name
			}
			return ids
		case "Functions":
			ids := make([]widget.TreeNodeID, len(schema.Functions))
			for i, fn := range schema.Functions {
				ids[i] = uid + sep + fn
			}
			return ids
		}

	case 4:
		if parts[2] != "Tables" {
			return nil
		}
		return []widget.TreeNodeID{
			uid + sep + "Columns",
			uid + sep + "Indexes",
			uid + sep + "Constraints",
			uid + sep + "Triggers",
		}

	case 5:
		schemaName := parts[1]
		tableName := parts[3]
		subCat := parts[4]
		schema := s.findSchema(meta, schemaName)
		if schema == nil {
			return nil
		}
		table := s.findTable(schema, tableName)
		if table == nil || !table.Loaded {
			return nil
		}
		switch subCat {
		case "Columns":
			ids := make([]widget.TreeNodeID, len(table.Columns))
			for i, c := range table.Columns {
				ids[i] = uid + sep + c.Name + " (" + c.DataType + ")"
			}
			return ids
		case "Indexes":
			ids := make([]widget.TreeNodeID, len(table.Indexes))
			for i, idx := range table.Indexes {
				ids[i] = uid + sep + idx
			}
			return ids
		case "Constraints":
			ids := make([]widget.TreeNodeID, len(table.Constraints))
			for i, con := range table.Constraints {
				ids[i] = uid + sep + con
			}
			return ids
		case "Triggers":
			ids := make([]widget.TreeNodeID, len(table.Triggers))
			for i, trig := range table.Triggers {
				ids[i] = uid + sep + trig
			}
			return ids
		}
	}

	return nil
}

// isBranch returns whether a node can be expanded.
func (s *Sidebar) isBranch(uid widget.TreeNodeID) bool {
	parts := strings.Split(uid, sep)
	switch len(parts) {
	case 1:
		return true // connection
	case 2:
		return true // schema
	case 3:
		return true // Tables/Functions category
	case 4:
		return parts[2] == "Tables" // Table = branch, Function = leaf
	case 5:
		return true // Sub-category
	default:
		return false
	}
}

func (s *Sidebar) createNode(branch bool) fyne.CanvasObject {
	return widget.NewLabel("template")
}

func (s *Sidebar) updateNode(uid widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
	label := obj.(*widget.Label)
	parts := strings.Split(uid, sep)
	depth := len(parts)
	displayName := parts[depth-1]

	label.TextStyle.Bold = false

	switch depth {
	case 1:
		connID := parts[0]
		connName := connID
		for _, conn := range s.config.Connections {
			if conn.ID == connID {
				connName = conn.Name
				break
			}
		}
		if s.connMgr.IsConnected(connID) {
			displayName = "● " + connName
		} else {
			displayName = "○ " + connName
		}
		label.TextStyle.Bold = true
	case 2:
		label.TextStyle.Bold = true
	case 3:
		label.TextStyle.Bold = true
	case 5:
		label.TextStyle.Bold = true
	}

	label.SetText(displayName)
}

func (s *Sidebar) showFunctionDetail(connID, schemaName, funcName string) {
	pool := s.connMgr.Pool(connID)
	if pool == nil {
		return
	}
	def, err := db.FetchFunctionDef(context.Background(), pool, schemaName, funcName)
	if err != nil {
		fyne.Do(func() {
			dialog.ShowError(fmt.Errorf("failed to load function: %w", err), s.window)
		})
		return
	}

	fyne.Do(func() {
		entry := widget.NewMultiLineEntry()
		entry.SetText(def)
		entry.TextStyle.Monospace = true

		scroll := container.NewScroll(entry)
		scroll.SetMinSize(fyne.NewSize(600, 400))

		var d dialog.Dialog
		copyBtn := widget.NewButton("Open in Terminal", func() {
			for _, conn := range s.config.Connections {
				if conn.ID == connID {
					s.onOpenWithText(conn, def)
					break
				}
			}
			d.Hide()
		})
		copyBtn.Importance = widget.HighImportance

		closeBtn := widget.NewButton("Close", func() {
			d.Hide()
		})

		buttons := container.NewHBox(copyBtn, closeBtn)
		content := container.NewBorder(nil, buttons, nil, nil, scroll)

		d = dialog.NewCustomWithoutButtons(schemaName+"."+funcName, content, s.window)
		d.Resize(fyne.NewSize(650, 500))
		d.Show()
	})
}

func (s *Sidebar) importConnections() {
	dlg := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()

		data, err := io.ReadAll(reader)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to read file: %w", err), s.window)
			return
		}

		conns, err := parseConnectionJSON(data)
		if err != nil {
			dialog.ShowError(fmt.Errorf("invalid JSON: %w", err), s.window)
			return
		}

		if len(conns) == 0 {
			dialog.ShowInformation("Import", "No connections found in file.", s.window)
			return
		}

		s.onImport(conns)
		dialog.ShowInformation("Import", fmt.Sprintf("Imported %d connection(s).", len(conns)), s.window)
	}, s.window)
	dlg.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	dlg.Show()
}

// parseConnectionJSON handles both a single object and an array of objects.
func parseConnectionJSON(data []byte) ([]config.ConnectionConfig, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("empty file")
	}

	// Try array first.
	if data[0] == '[' {
		var conns []config.ConnectionConfig
		if err := json.Unmarshal(data, &conns); err != nil {
			return nil, err
		}
		return conns, nil
	}

	// Try single object.
	var conn config.ConnectionConfig
	if err := json.Unmarshal(data, &conn); err != nil {
		return nil, err
	}
	return []config.ConnectionConfig{conn}, nil
}

func (s *Sidebar) findSchema(meta *db.DatabaseMeta, name string) *db.SchemaMeta {
	if meta == nil {
		return nil
	}
	for i := range meta.Schemas {
		if meta.Schemas[i].Name == name {
			return &meta.Schemas[i]
		}
	}
	return nil
}

func (s *Sidebar) findTable(schema *db.SchemaMeta, name string) *db.TableMeta {
	for i := range schema.Tables {
		if schema.Tables[i].Name == name {
			return &schema.Tables[i]
		}
	}
	return nil
}

// Refresh updates the tree display after configuration changes.
func (s *Sidebar) Refresh() {
	s.tree.Refresh()
}

// Widget returns the sidebar's top-level canvas object for embedding in layouts.
func (s *Sidebar) Widget() fyne.CanvasObject {
	return s.container
}
