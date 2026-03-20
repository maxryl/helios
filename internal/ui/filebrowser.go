package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// FileBrowser shows a directory tree and lets the user open files.
type FileBrowser struct {
	tree       *widget.Tree
	rootDir    string
	window     fyne.Window
	onOpenFile func(path string)
	container  fyne.CanvasObject
}

// NewFileBrowser creates a file browser panel.
func NewFileBrowser(window fyne.Window, onOpenFile func(path string)) *FileBrowser {
	fb := &FileBrowser{
		window:     window,
		onOpenFile: onOpenFile,
	}

	fb.tree = widget.NewTree(
		fb.childUIDs,
		fb.isBranch,
		func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(uid widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			label.SetText(filepath.Base(uid))
		},
	)

	fb.tree.OnSelected = func(uid widget.TreeNodeID) {
		info, err := os.Stat(uid)
		if err != nil || info.IsDir() {
			return
		}
		fb.onOpenFile(uid)
		fb.tree.UnselectAll()
	}

	openDirBtn := widget.NewButtonWithIcon("Open Folder", theme.FolderOpenIcon(), fb.pickDirectory)
	openDirBtn.Importance = widget.MediumImportance

	newFileBtn := widget.NewButtonWithIcon("New File", theme.DocumentCreateIcon(), fb.newFile)
	newFileBtn.Importance = widget.LowImportance

	toolbar := container.NewHBox(openDirBtn, newFileBtn)

	fb.container = container.NewBorder(toolbar, nil, nil, nil, fb.tree)
	return fb
}

func (fb *FileBrowser) pickDirectory() {
	dlg := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}
		fb.SetRoot(uri.Path())
	}, fb.window)
	dlg.Show()
}

func (fb *FileBrowser) newFile() {
	if fb.rootDir == "" {
		dialog.ShowInformation("No Folder", "Open a folder first.", fb.window)
		return
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("filename.sql")

	dialog.ShowForm("New File", "Create", "Cancel",
		[]*widget.FormItem{widget.NewFormItem("Name", nameEntry)},
		func(ok bool) {
			if !ok || strings.TrimSpace(nameEntry.Text) == "" {
				return
			}
			path := filepath.Join(fb.rootDir, nameEntry.Text)
			if err := os.WriteFile(path, []byte(""), 0644); err != nil {
				dialog.ShowError(err, fb.window)
				return
			}
			fb.tree.Refresh()
			fb.onOpenFile(path)
		}, fb.window)
}

// SetRoot sets the root directory and refreshes the tree.
func (fb *FileBrowser) SetRoot(dir string) {
	fb.rootDir = dir
	fb.tree.Refresh()
}

func (fb *FileBrowser) childUIDs(uid widget.TreeNodeID) []widget.TreeNodeID {
	dir := uid
	if dir == "" {
		if fb.rootDir == "" {
			return nil
		}
		dir = fb.rootDir
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	// Sort: directories first, then files, both alphabetical.
	sort.Slice(entries, func(i, j int) bool {
		di, dj := entries[i].IsDir(), entries[j].IsDir()
		if di != dj {
			return di
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	var ids []widget.TreeNodeID
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue // skip hidden files
		}
		ids = append(ids, filepath.Join(dir, name))
	}
	return ids
}

func (fb *FileBrowser) isBranch(uid widget.TreeNodeID) bool {
	if uid == "" {
		return true
	}
	info, err := os.Stat(uid)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// Widget returns the file browser for embedding in layouts.
func (fb *FileBrowser) Widget() fyne.CanvasObject {
	return fb.container
}
