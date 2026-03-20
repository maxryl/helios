package ui

import (
	"errors"
	"image/color"

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

// Sidebar presents the saved connections list with toolbar actions.
type Sidebar struct {
	list      *widget.List
	config    *config.AppConfig
	connMgr   *db.ConnectionManager
	onSelect  func(config.ConnectionConfig)
	onEdit    func(config.ConnectionConfig)
	onDelete  func(string)
	selected  int
	container fyne.CanvasObject
}

// NewSidebar creates a sidebar with a toolbar and scrollable connection list.
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
		selected: -1,
	}

	s.list = widget.NewList(
		func() int {
			return len(s.config.Connections)
		},
		func() fyne.CanvasObject {
			circle := canvas.NewCircle(color.Transparent)
			circle.StrokeWidth = 2
			label := widget.NewLabel("template")
			return container.NewHBox(
				container.New(&fixedSizeLayout{size: fyne.NewSquareSize(12)}, circle),
				label,
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			box := obj.(*fyne.Container)
			circle := box.Objects[0].(*fyne.Container).Objects[0].(*canvas.Circle)
			label := box.Objects[1].(*widget.Label)
			if id >= len(s.config.Connections) {
				return
			}
			conn := s.config.Connections[id]
			if s.connMgr.IsConnected(conn.ID) {
				circle.FillColor = theme.Color(theme.ColorNameSuccess)
				circle.StrokeColor = theme.Color(theme.ColorNameSuccess)
			} else {
				circle.FillColor = color.Transparent
				circle.StrokeColor = theme.Color(theme.ColorNamePlaceHolder)
			}
			circle.Refresh()
			label.SetText(conn.Name)
		},
	)

	s.list.OnSelected = func(id widget.ListItemID) {
		if id >= len(s.config.Connections) {
			return
		}
		s.selected = id
		s.onSelect(s.config.Connections[id])
	}

	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentAddIcon(), func() {
			s.onEdit(config.ConnectionConfig{})
		}),
		widget.NewToolbarAction(theme.DocumentCreateIcon(), func() {
			if s.selected >= 0 && s.selected < len(s.config.Connections) {
				s.onEdit(s.config.Connections[s.selected])
			}
		}),
		widget.NewToolbarAction(theme.DeleteIcon(), func() {
			if s.selected >= 0 && s.selected < len(s.config.Connections) {
				s.onDelete(s.config.Connections[s.selected].ID)
				s.selected = -1
			}
		}),
	)

	header := widget.NewLabel("Connections")
	header.TextStyle.Bold = true

	rightBorder := widget.NewSeparator()
	content := container.NewBorder(container.NewVBox(header, toolbar), nil, nil, nil, s.list)
	s.container = container.NewBorder(nil, nil, nil, rightBorder, content)
	return s
}

// Refresh updates the list display after configuration changes.
func (s *Sidebar) Refresh() {
	s.selected = -1
	s.list.UnselectAll()
	s.list.Refresh()
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
