package ui

import (
	"errors"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"helios/internal/config"
)

var errFieldRequired = errors.New("this field is required")

// ShowConnectionDialog displays a modal form for creating or editing a connection.
// When existing is non-nil the form is pre-populated for editing; otherwise fields
// start empty for a new connection. On confirmation onSave is called with the
// constructed ConnectionConfig.
func ShowConnectionDialog(window fyne.Window, existing *config.ConnectionConfig, onSave func(config.ConnectionConfig)) {
	requiredValidator := func(s string) error {
		if s == "" {
			return errFieldRequired
		}
		return nil
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("My Database")
	nameEntry.Validator = requiredValidator

	hostEntry := widget.NewEntry()
	hostEntry.SetPlaceHolder("localhost")
	hostEntry.Validator = requiredValidator

	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("5432")

	userEntry := widget.NewEntry()
	userEntry.SetPlaceHolder("postgres")

	passwordEntry := widget.NewPasswordEntry()

	dbEntry := widget.NewEntry()
	dbEntry.SetPlaceHolder("postgres")

	sslSelect := widget.NewSelect(
		[]string{"disable", "require", "verify-ca", "verify-full"},
		nil,
	)
	sslSelect.SetSelected("disable")

	var existingID string
	if existing != nil {
		existingID = existing.ID
		nameEntry.SetText(existing.Name)
		hostEntry.SetText(existing.Host)
		if existing.Port != 0 {
			portEntry.SetText(strconv.Itoa(existing.Port))
		}
		userEntry.SetText(existing.User)
		passwordEntry.SetText(existing.Password)
		dbEntry.SetText(existing.DBName)
		if existing.SSLMode != "" {
			sslSelect.SetSelected(existing.SSLMode)
		}
	}

	items := []*widget.FormItem{
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("Host", hostEntry),
		widget.NewFormItem("Port", portEntry),
		widget.NewFormItem("User", userEntry),
		widget.NewFormItem("Password", passwordEntry),
		widget.NewFormItem("Database", dbEntry),
		widget.NewFormItem("SSL Mode", sslSelect),
	}

	dlg := dialog.NewForm("Connection", "Save", "Cancel", items, func(ok bool) {
		if !ok {
			return
		}

		port := 5432
		if v, err := strconv.Atoi(portEntry.Text); err == nil && v > 0 {
			port = v
		}

		cfg := config.ConnectionConfig{
			ID:       existingID,
			Name:     nameEntry.Text,
			Host:     hostEntry.Text,
			Port:     port,
			User:     userEntry.Text,
			Password: passwordEntry.Text,
			DBName:   dbEntry.Text,
			SSLMode:  sslSelect.Selected,
		}
		onSave(cfg)
	}, window)

	dlg.Resize(fyne.NewSize(480, 450))
	dlg.Show()
}
