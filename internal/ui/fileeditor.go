package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// Files under this size use a full editable Entry.
// Files above use a virtualized line-based viewer with paged editing.
const editableThreshold = 512 * 1024 // 512 KB

// FileEditor is an editor pane for a single file.
// Small files get a full Entry editor. Large files get a virtualized
// read-only list with a page-based editor for the visible region.
type FileEditor struct {
	path    string
	window  fyne.Window
	content fyne.CanvasObject
	dirty   bool
	onDirty func(bool)

	// Small file mode.
	editor *widget.Entry

	// Large file mode.
	lines      []string
	list       *widget.List
	pageEditor *widget.Entry
	pageStart  int // first line index of current edit page
	pageSize   int
	editing    bool
	statusBar  *widget.Label
}

// NewFileEditor opens a file and creates an editor for it.
func NewFileEditor(path string, window fyne.Window, onDirty func(bool)) (*FileEditor, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("fileeditor: stat: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("fileeditor: read: %w", err)
	}

	if isBinary(data) {
		return nil, fmt.Errorf("binary file — cannot open in text editor")
	}

	fe := &FileEditor{
		path:    path,
		window:  window,
		onDirty: onDirty,
	}

	if info.Size() <= editableThreshold {
		fe.buildSmallEditor(string(data))
	} else {
		fe.buildLargeViewer(string(data))
	}

	return fe, nil
}

func (fe *FileEditor) buildSmallEditor(text string) {
	fe.editor = widget.NewMultiLineEntry()
	fe.editor.SetText(text)
	fe.editor.Wrapping = fyne.TextWrapOff
	fe.editor.TextStyle.Monospace = true
	fe.editor.OnChanged = func(_ string) {
		fe.markDirty()
	}
	fe.content = container.NewStack(fe.editor)
}

func (fe *FileEditor) buildLargeViewer(text string) {
	fe.lines = strings.Split(text, "\n")
	fe.pageSize = 200

	fe.list = widget.NewList(
		func() int { return len(fe.lines) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.TextStyle.Monospace = true
			label.Truncation = fyne.TextTruncateEllipsis
			return label
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id < len(fe.lines) {
				label.SetText(fmt.Sprintf("%5d  %s", id+1, fe.lines[id]))
			}
		},
	)

	fe.statusBar = widget.NewLabel(fmt.Sprintf("%d lines — read-only view (click Edit Page to modify)", len(fe.lines)))
	fe.statusBar.TextStyle.Bold = true

	editPageBtn := widget.NewButton("Edit Page", func() {
		fe.openPageEditor()
	})
	editPageBtn.Importance = widget.MediumImportance

	goToBtn := widget.NewButton("Go to Line", func() {
		fe.goToLine()
	})
	goToBtn.Importance = widget.LowImportance

	toolbar := container.NewHBox(editPageBtn, goToBtn, layout.NewSpacer(), fe.statusBar)
	fe.content = container.NewBorder(toolbar, nil, nil, nil, fe.list)
}

func (fe *FileEditor) openPageEditor() {
	// Determine which lines are visible. Use the list's scroll offset
	// to pick the page, or default to the beginning.
	start := fe.pageStart
	end := start + fe.pageSize
	if end > len(fe.lines) {
		end = len(fe.lines)
	}

	pageText := strings.Join(fe.lines[start:end], "\n")

	fe.pageEditor = widget.NewMultiLineEntry()
	fe.pageEditor.SetText(pageText)
	fe.pageEditor.Wrapping = fyne.TextWrapOff
	fe.pageEditor.TextStyle.Monospace = true

	title := fmt.Sprintf("Editing lines %d–%d of %d", start+1, end, len(fe.lines))

	var d dialog.Dialog

	saveBtn := widget.NewButton("Apply Changes", func() {
		// Replace lines in the original data.
		newLines := strings.Split(fe.pageEditor.Text, "\n")
		updated := make([]string, 0, len(fe.lines)-fe.pageSize+len(newLines))
		updated = append(updated, fe.lines[:start]...)
		updated = append(updated, newLines...)
		if end < len(fe.lines) {
			updated = append(updated, fe.lines[end:]...)
		}
		fe.lines = updated
		fe.list.Refresh()
		fe.markDirty()
		fe.statusBar.SetText(fmt.Sprintf("%d lines — modified", len(fe.lines)))
		d.Hide()
	})
	saveBtn.Importance = widget.HighImportance

	closeBtn := widget.NewButton("Cancel", func() {
		d.Hide()
	})

	buttons := container.NewHBox(saveBtn, closeBtn)
	scroll := container.NewScroll(fe.pageEditor)
	scroll.SetMinSize(fyne.NewSize(700, 400))
	content := container.NewBorder(nil, buttons, nil, nil, scroll)

	d = dialog.NewCustomWithoutButtons(title, content, fe.window)
	d.Resize(fyne.NewSize(750, 500))
	d.Show()
}

func (fe *FileEditor) goToLine() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Line number")

	dialog.ShowForm("Go to Line", "Go", "Cancel",
		[]*widget.FormItem{widget.NewFormItem("Line", entry)},
		func(ok bool) {
			if !ok {
				return
			}
			var line int
			if _, err := fmt.Sscanf(entry.Text, "%d", &line); err != nil || line < 1 {
				return
			}
			line-- // 0-indexed
			if line >= len(fe.lines) {
				line = len(fe.lines) - 1
			}
			fe.pageStart = line
			fe.list.ScrollTo(widget.ListItemID(line))
		}, fe.window)
}

func (fe *FileEditor) markDirty() {
	if !fe.dirty {
		fe.dirty = true
		if fe.onDirty != nil {
			fe.onDirty(true)
		}
	}
}

// Content returns the editor widget for embedding in tabs.
func (fe *FileEditor) Content() fyne.CanvasObject {
	return fe.content
}

// Save writes the editor content back to disk.
func (fe *FileEditor) Save() error {
	var text string
	if fe.editor != nil {
		text = fe.editor.Text
	} else {
		text = strings.Join(fe.lines, "\n")
	}
	if err := os.WriteFile(fe.path, []byte(text), 0644); err != nil {
		return fmt.Errorf("fileeditor: save: %w", err)
	}
	fe.dirty = false
	if fe.onDirty != nil {
		fe.onDirty(false)
	}
	return nil
}

// Path returns the file path.
func (fe *FileEditor) Path() string {
	return fe.path
}

// FileName returns just the base name.
func (fe *FileEditor) FileName() string {
	return filepath.Base(fe.path)
}

// isBinary checks the first 8KB for null bytes to detect binary content.
func isBinary(data []byte) bool {
	check := data
	if len(check) > 8192 {
		check = check[:8192]
	}
	for _, b := range check {
		if b == 0 {
			return true
		}
	}
	return false
}

// IsDirty returns whether the editor has unsaved changes.
func (fe *FileEditor) IsDirty() bool {
	return fe.dirty
}

// ConfirmClose prompts the user to save if dirty, then calls onClose.
func (fe *FileEditor) ConfirmClose(onClose func()) {
	if !fe.dirty {
		onClose()
		return
	}
	dialog.ShowConfirm("Unsaved Changes",
		fmt.Sprintf("Save changes to %s?", fe.FileName()),
		func(save bool) {
			if save {
				_ = fe.Save()
			}
			onClose()
		}, fe.window)
}
