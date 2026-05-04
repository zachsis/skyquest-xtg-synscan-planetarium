package main

import (
	"fyne.io/fyne/v2/app"

	"github.com/zhatsis/oriontelescope/internal/ui"
)

func main() {
	a := app.NewWithID("com.zhatsis.oriontelescope")
	w := a.NewWindow("OrionTelescope")

	mainApp := ui.NewMainApp(a, w)
	mainApp.Setup()

	w.ShowAndRun()
}
