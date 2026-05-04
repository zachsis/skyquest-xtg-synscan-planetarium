package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// MainApp holds the application window, navigation, and panels.
type MainApp struct {
	fyneApp fyne.App
	window  fyne.Window
	nav     *widget.List
	content *fyne.Container
	panels  []Panel
}

// NewMainApp creates a new MainApp.
func NewMainApp(a fyne.App, w fyne.Window) *MainApp {
	return &MainApp{
		fyneApp: a,
		window:  w,
	}
}

// Setup initializes the navigation panels and window layout.
func (m *MainApp) Setup() {
	m.panels = []Panel{
		NewPlaceholderPanel("Status", theme.InfoIcon()),
		NewPlaceholderPanel("GoTo", theme.NavigateNextIcon()),
		NewPlaceholderPanel("Tracking", theme.MediaPlayIcon()),
		NewPlaceholderPanel("Alignment", theme.VisibilityIcon()),
		NewPlaceholderPanel("Sky Chart", theme.ColorChromaticIcon()),
		NewPlaceholderPanel("Objects", theme.SearchIcon()),
		NewPlaceholderPanel("Settings", theme.SettingsIcon()),
	}

	m.content = container.NewStack(m.panels[0].Content())

	m.nav = widget.NewList(
		func() int { return len(m.panels) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.DocumentIcon()),
				widget.NewLabel("Template"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			c := obj.(*fyne.Container)
			c.Objects[0].(*widget.Icon).SetResource(m.panels[id].Icon())
			c.Objects[1].(*widget.Label).SetText(m.panels[id].Title())
		},
	)
	m.nav.OnSelected = func(id widget.ListItemID) {
		m.content.Objects = []fyne.CanvasObject{m.panels[id].Content()}
		m.content.Refresh()
	}

	// Select Status panel by default.
	m.nav.Select(0)

	split := container.NewHSplit(m.nav, m.content)
	split.SetOffset(0.2)

	m.window.SetContent(split)
	m.window.Resize(fyne.NewSize(1024, 768))
}
