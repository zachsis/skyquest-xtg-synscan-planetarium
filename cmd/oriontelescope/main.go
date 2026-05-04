package main

import (
	"log"

	"fyne.io/fyne/v2/app"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/config"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/ui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	a := app.NewWithID("com.zhatsis.oriontelescope")
	w := a.NewWindow("OrionTelescope")

	mainApp := ui.NewMainApp(a, w, cfg)
	mainApp.Setup()

	w.ShowAndRun()
}
