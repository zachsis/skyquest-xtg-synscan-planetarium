package ui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/astro"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/config"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/slew"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/ui/panels"
)

// MainApp holds the application window, navigation, and panels.
type MainApp struct {
	fyneApp      fyne.App
	window       fyne.Window
	config       *config.Config
	nav          *widget.List
	content      *fyne.Container
	panels       []Panel
	StatusPanel  *StatusPanel // exposed for PositionProvider consumers
}

// NewMainApp creates a new MainApp.
func NewMainApp(a fyne.App, w fyne.Window, cfg *config.Config) *MainApp {
	return &MainApp{
		fyneApp: a,
		window:  w,
		config:  cfg,
	}
}

// Setup initializes the navigation panels and window layout.
func (m *MainApp) Setup() {
	settingsPanel := panels.NewSettingsPanel(m.config, nil)
	astroSvc := astro.NewAstroService(m.config)
	statusPanel := NewStatusPanel(m.config, astroSvc)
	m.StatusPanel = statusPanel

	cat, err := catalog.NewCatalog(astroSvc)
	if err != nil {
		log.Printf("warning: star catalog failed to load: %v", err)
	}

	slewSvc := slew.NewGoToService(statusPanel.Controller())
	trackingPanel := NewTrackingPanel(statusPanel, statusPanel.Controller())
	gotoPanel := NewGoToPanel(statusPanel, slewSvc, astroSvc, m.config)

	var skyChartPanel Panel
	if cat != nil {
		skyChartPanel = NewSkyChartPanel(cat, m.config, astroSvc)
	} else {
		skyChartPanel = NewPlaceholderPanel("Sky Chart", theme.ColorChromaticIcon())
	}

	m.panels = []Panel{
		statusPanel,
		gotoPanel,
		trackingPanel,
		NewAlignmentPanel(statusPanel, slewSvc, astroSvc, m.config),
		skyChartPanel,
		NewPlaceholderPanel("Objects", theme.SearchIcon()),
		settingsPanel,
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
			icon := m.panels[id].Icon()
			if icon == nil {
				icon = theme.SettingsIcon()
			}
			c.Objects[0].(*widget.Icon).SetResource(icon)
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
