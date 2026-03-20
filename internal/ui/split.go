package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const splitHandleWidth float32 = 6

// SmoothHSplit is a horizontal split container that only moves a preview
// line during drag and defers the actual child relayout to DragEnd.
type SmoothHSplit struct {
	widget.BaseWidget
	Leading  fyne.CanvasObject
	Trailing fyne.CanvasObject
	offset   float32 // 0.0–1.0
	pendingX float32 // drag preview X in parent coords, 0 = no drag
	bar      *canvas.Rectangle
	preview  *canvas.Rectangle
	handle   *splitHandle
}

// NewSmoothHSplit creates a smooth horizontal split.
func NewSmoothHSplit(leading, trailing fyne.CanvasObject) *SmoothHSplit {
	s := &SmoothHSplit{
		Leading:  leading,
		Trailing: trailing,
		offset:   0.2,
		bar:      canvas.NewRectangle(theme.Color(theme.ColorNameSeparator)),
		preview:  canvas.NewRectangle(color.Transparent),
	}
	s.handle = &splitHandle{parent: s}
	s.handle.ExtendBaseWidget(s.handle)
	s.ExtendBaseWidget(s)
	return s
}

// SetOffset sets the divider position.
func (s *SmoothHSplit) SetOffset(offset float32) {
	s.offset = clamp(offset, 0.05, 0.95)
	s.Refresh()
}

func (s *SmoothHSplit) CreateRenderer() fyne.WidgetRenderer {
	return &smoothSplitRenderer{
		split:   s,
		objects: []fyne.CanvasObject{s.Leading, s.Trailing, s.bar, s.handle, s.preview},
	}
}

type smoothSplitRenderer struct {
	split   *SmoothHSplit
	objects []fyne.CanvasObject
}

func (r *smoothSplitRenderer) Layout(size fyne.Size) {
	s := r.split
	divX := size.Width * s.offset
	hw := splitHandleWidth

	s.Leading.Move(fyne.NewPos(0, 0))
	s.Leading.Resize(fyne.NewSize(divX-hw/2, size.Height))

	s.bar.Move(fyne.NewPos(divX-0.5, 0))
	s.bar.Resize(fyne.NewSize(1, size.Height))

	s.handle.Move(fyne.NewPos(divX-hw/2, 0))
	s.handle.Resize(fyne.NewSize(hw, size.Height))

	s.Trailing.Move(fyne.NewPos(divX+hw/2, 0))
	s.Trailing.Resize(fyne.NewSize(size.Width-divX-hw/2, size.Height))

	// Hide preview when not dragging.
	if s.pendingX <= 0 {
		s.preview.Resize(fyne.NewSize(0, 0))
	}
}

func (r *smoothSplitRenderer) MinSize() fyne.Size {
	s := r.split
	lMin := s.Leading.MinSize()
	tMin := s.Trailing.MinSize()
	return fyne.NewSize(lMin.Width+tMin.Width+splitHandleWidth, fyne.Max(lMin.Height, tMin.Height))
}

func (r *smoothSplitRenderer) Refresh() {
	s := r.split
	s.bar.FillColor = theme.Color(theme.ColorNameSeparator)
	s.bar.Refresh()
	r.objects[0] = s.Leading
	r.objects[1] = s.Trailing
	r.Layout(s.Size())
}

func (r *smoothSplitRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *smoothSplitRenderer) Destroy() {}

// splitHandle is the invisible draggable divider area.
type splitHandle struct {
	widget.BaseWidget
	parent *SmoothHSplit
}

func (h *splitHandle) CreateRenderer() fyne.WidgetRenderer {
	// Invisible — just a hit area.
	bg := canvas.NewRectangle(color.Transparent)
	return widget.NewSimpleRenderer(bg)
}

func (h *splitHandle) Cursor() desktop.Cursor {
	return desktop.HResizeCursor
}

func (h *splitHandle) Dragged(e *fyne.DragEvent) {
	s := h.parent
	size := s.Size()
	if size.Width <= 0 {
		return
	}

	handlePos := h.Position()
	newX := handlePos.X + e.Position.X
	newX = clamp(newX, size.Width*0.05, size.Width*0.95)

	// Just move the preview line — no child relayout.
	s.pendingX = newX
	s.preview.FillColor = theme.Color(theme.ColorNamePrimary)
	s.preview.Move(fyne.NewPos(newX-1, 0))
	s.preview.Resize(fyne.NewSize(2, size.Height))
	s.preview.Refresh()
}

func (h *splitHandle) DragEnd() {
	s := h.parent
	if s.pendingX <= 0 {
		return
	}
	size := s.Size()
	if size.Width > 0 {
		s.offset = clamp(s.pendingX/size.Width, 0.05, 0.95)
	}
	s.pendingX = 0
	s.preview.FillColor = color.Transparent
	s.preview.Resize(fyne.NewSize(0, 0))
	s.preview.Refresh()
	s.Refresh() // Single relayout of children.
}

func clamp(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
