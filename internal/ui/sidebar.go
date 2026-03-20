package ui

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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
	onSelect  func(config.ConnectionConfig)
	onEdit    func(config.ConnectionConfig)
	onDelete  func(string)
	container fyne.CanvasObject

	mu             sync.RWMutex
	metaByID       map[string]*db.DatabaseMeta // connID -> metadata
	lastSelectedID string                       // last selected connection ID
}

// NewSidebar creates a sidebar with a toolbar and schema browser tree.
func NewSidebar(
	cfg *config.AppConfig,
	connMgr *db.ConnectionManager,
	onSelect func(config.ConnectionConfig),
	onEdit func(config.ConnectionConfig),
	onDelete func(string),
) *Sidebar {
	s := &Sidebar{
		config:   cfg,
		connMgr:  connMgr,
		onSelect: onSelect,
		onEdit:   onEdit,
		onDelete: onDelete,
		metaByID: make(map[string]*db.DatabaseMeta),
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

		// Only open terminal when clicking a connection root node.
		if len(parts) == 1 {
			for _, conn := range s.config.Connections {
				if conn.ID == connID {
					s.onSelect(conn)
					break
				}
			}
		}
	}

	s.tree.OnBranchOpened = func(uid widget.TreeNodeID) {
		parts := strings.Split(uid, sep)
		if len(parts) == 1 {
			connID := parts[0]
			s.mu.RLock()
			_, loaded := s.metaByID[connID]
			s.mu.RUnlock()
			if !loaded && s.connMgr.IsConnected(connID) {
				go s.loadMeta(connID)
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
	)

	header := widget.NewLabel("Connections")
	header.TextStyle.Bold = true

	rightBorder := widget.NewSeparator()
	content := container.NewBorder(container.NewVBox(header, toolbar), nil, nil, nil, s.tree)
	s.container = container.NewBorder(nil, nil, nil, rightBorder, content)
	return s
}

func (s *Sidebar) loadMeta(connID string) {
	pool := s.connMgr.Pool(connID)
	if pool == nil {
		return
	}
	meta, err := db.FetchDatabaseMeta(context.Background(), pool)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.metaByID[connID] = meta
	s.mu.Unlock()
	fyne.Do(func() {
		s.tree.Refresh()
	})
}

// childUIDs returns the children of a tree node.
func (s *Sidebar) childUIDs(uid widget.TreeNodeID) []widget.TreeNodeID {
	if uid == "" {
		// Root: return connection IDs.
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
		// Connection node -> schemas
		if meta == nil {
			return nil
		}
		ids := make([]widget.TreeNodeID, len(meta.Schemas))
		for i, schema := range meta.Schemas {
			ids[i] = uid + sep + schema.Name
		}
		return ids

	case 2:
		// Schema node -> Tables, Functions
		return []widget.TreeNodeID{
			uid + sep + "Tables",
			uid + sep + "Functions",
		}

	case 3:
		// Category node (Tables or Functions)
		schemaName := parts[1]
		category := parts[2]
		schema := s.findSchema(meta, schemaName)
		if schema == nil {
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
		// Table node -> Columns, Indexes, Constraints, Triggers
		if parts[2] != "Tables" {
			return nil // Function leaf
		}
		return []widget.TreeNodeID{
			uid + sep + "Columns",
			uid + sep + "Indexes",
			uid + sep + "Constraints",
			uid + sep + "Triggers",
		}

	case 5:
		// Sub-category node (Columns, Indexes, etc.)
		schemaName := parts[1]
		tableName := parts[3]
		subCat := parts[4]
		schema := s.findSchema(meta, schemaName)
		if schema == nil {
			return nil
		}
		table := s.findTable(schema, tableName)
		if table == nil {
			return nil
		}
		switch subCat {
		case "Columns":
			ids := make([]widget.TreeNodeID, len(table.Columns))
			for i, c := range table.Columns {
				label := c.Name + " (" + c.DataType + ")"
				ids[i] = uid + sep + label
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
		// Tables/Functions category
		return true
	case 4:
		// Table = branch, Function = leaf
		return parts[2] == "Tables"
	case 5:
		// Sub-category (Columns, Indexes, etc.)
		return true
	default:
		return false // leaf items (columns, indexes, etc.)
	}
}

// createNode creates a template tree node widget.
func (s *Sidebar) createNode(branch bool) fyne.CanvasObject {
	icon := canvas.NewCircle(color.Transparent)
	icon.StrokeWidth = 2
	label := widget.NewLabel("template")
	return container.NewHBox(
		container.New(&fixedSizeLayout{size: fyne.NewSquareSize(12)}, icon),
		label,
	)
}

// updateNode updates a tree node's display based on its ID.
func (s *Sidebar) updateNode(uid widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
	box := obj.(*fyne.Container)
	icon := box.Objects[0].(*fyne.Container).Objects[0].(*canvas.Circle)
	label := box.Objects[1].(*widget.Label)

	parts := strings.Split(uid, sep)
	depth := len(parts)
	displayName := parts[depth-1]

	// Default: hide circle.
	icon.FillColor = color.Transparent
	icon.StrokeColor = color.Transparent

	switch depth {
	case 1:
		// Connection node: show status circle.
		connID := parts[0]
		connName := connID
		for _, conn := range s.config.Connections {
			if conn.ID == connID {
				connName = conn.Name
				break
			}
		}
		displayName = connName
		if s.connMgr.IsConnected(connID) {
			icon.FillColor = theme.Color(theme.ColorNameSuccess)
			icon.StrokeColor = theme.Color(theme.ColorNameSuccess)
		} else {
			icon.StrokeColor = theme.Color(theme.ColorNamePlaceHolder)
		}
	case 2:
		// Schema node
		label.TextStyle.Bold = true
	case 3:
		// Category (Tables, Functions)
		label.TextStyle.Bold = true
		displayName = fmt.Sprintf("%s", parts[2])
	case 4:
		label.TextStyle.Bold = false
	case 5:
		// Sub-category (Columns, Indexes, etc.)
		label.TextStyle.Bold = true
	default:
		// Leaf
		label.TextStyle.Bold = false
	}

	icon.Refresh()
	label.SetText(displayName)
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

// SetMeta stores pre-loaded metadata for a connection.
func (s *Sidebar) SetMeta(connID string, meta *db.DatabaseMeta) {
	s.mu.Lock()
	s.metaByID[connID] = meta
	s.mu.Unlock()
	fyne.Do(func() {
		s.tree.Refresh()
	})
}

// Widget returns the sidebar's top-level canvas object for embedding in layouts.
func (s *Sidebar) Widget() fyne.CanvasObject {
	return s.container
}

// fixedSizeLayout constrains children to a fixed size.
type fixedSizeLayout struct {
	size fyne.Size
}

func (l *fixedSizeLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return l.size
}

func (l *fixedSizeLayout) Layout(objects []fyne.CanvasObject, _ fyne.Size) {
	for _, o := range objects {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(l.size)
	}
}
