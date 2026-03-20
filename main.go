package main

import (
	"log"

	"fyne.io/fyne/v2/app"

	"helios/internal/config"
	"helios/internal/db"
	"helios/internal/ui"
)

func main() {
	a := app.New()

	cfg := &config.AppConfig{}
	cfgPath, err := config.DefaultPath()
	if err != nil {
		log.Fatal(err)
	}
	if err := cfg.Load(cfgPath); err != nil {
		log.Fatal(err)
	}

	connMgr := db.NewConnectionManager()

	heliosApp := ui.NewApp(a, cfg, cfgPath, connMgr)
	heliosApp.Show()
}
