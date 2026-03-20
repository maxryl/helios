package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// FileEditor is an editor pane for a single file.
type FileEditor struct {
	path    string
	editor  *widget.Entry
	window  fyne.Window
	content fyne.CanvasObject
	dirty   bool
	onDirty func(bool) // called when dirty state changes
}

// NewFileEditor opens a file and creates an editor for it.
func NewFileEditor(path string, window fyne.Window, onDirty func(bool)) (*FileEditor, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("fileeditor: read: %w", err)
	}

	fe := &FileEditor{
		path:    path,
		window:  window,
		onDirty: onDirty,
	}

	fe.editor = widget.NewMultiLineEntry()
	fe.editor.SetText(string(data))
	fe.editor.Wrapping = fyne.TextWrapOff
	fe.editor.TextStyle.Monospace = true

	fe.editor.OnChanged = func(_ string) {
		if !fe.dirty {
			fe.dirty = true
			if fe.onDirty != nil {
				fe.onDirty(true)
			}
		}
	}

	fe.content = container.NewStack(fe.editor)
	return fe, nil
}

// Content returns the editor widget for embedding in tabs.
func (fe *FileEditor) Content() fyne.CanvasObject {
	return fe.content
}

// Save writes the editor content back to disk.
func (fe *FileEditor) Save() error {
	if err := os.WriteFile(fe.path, []byte(fe.editor.Text), 0644); err != nil {
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
