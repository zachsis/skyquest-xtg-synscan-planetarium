package ui

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/astro"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/config"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/slew"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/telescope"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/ui/skychart"
)

// SkyChartPanel wraps the SkyChartWidget into a Panel with overlay controls.
type SkyChartPanel struct {
	chart   *skychart.SkyChartWidget
	content fyne.CanvasObject
}

// NewSkyChartPanel creates the sky chart panel.
func NewSkyChartPanel(cat *catalog.Catalog, cfg *config.Config, astroSvc *astro.AstroService, pos telescope.PositionProvider, slewSvc slew.GoToService) *SkyChartPanel {
	chart := skychart.NewSkyChartWidget(cat, cfg, astroSvc)

	// Register constellation overlay.
	conOverlay := skychart.NewConstellationOverlay(cat)
	chart.AddOverlay(conOverlay)

	// Register DSO & planet overlay.
	dsoOverlay := skychart.NewDSOOverlay(cat.Registry, cfg)
	chart.AddOverlay(dsoOverlay)
	chart.SetDSOOverlay(dsoOverlay)

	// Register crosshair overlay.
	crosshairOverlay := skychart.NewCrosshairOverlay(pos)
	chart.AddOverlay(crosshairOverlay)

	// Wire slew service for GoTo from star popup.
	chart.SetSlewService(slewSvc)

	showGrid := widget.NewCheck("Grid", func(checked bool) {
		chart.SetShowGrid(checked)
	})
	showGrid.SetChecked(true)

	showLines := widget.NewCheck("Constellation Lines", func(checked bool) {
		conOverlay.SetShowLines(checked)
		chart.Refresh()
	})
	showLines.SetChecked(true)

	showLabels := widget.NewCheck("Constellation Labels", func(checked bool) {
		conOverlay.SetShowLabels(checked)
		chart.Refresh()
	})
	showLabels.SetChecked(true)

	showDSOs := widget.NewCheck("Deep-Sky Objects", func(checked bool) {
		dsoOverlay.SetShowDSOs(checked)
		chart.Refresh()
	})
	showDSOs.SetChecked(true)

	showPlanets := widget.NewCheck("Planets", func(checked bool) {
		dsoOverlay.SetShowPlanets(checked)
		chart.Refresh()
	})
	showPlanets.SetChecked(true)

	fovEntry := widget.NewEntry()
	fovEntry.SetText("60")
	fovEntry.SetPlaceHolder("FOV (arcmin)")
	fovEntry.OnChanged = func(s string) {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 && v < 600 {
			crosshairOverlay.SetFOV(v)
			chart.Refresh()
		}
	}

	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		chart.Refresh()
	})

	controls := container.NewVBox(
		widget.NewLabel("Overlays"),
		widget.NewSeparator(),
		showGrid,
		showLines,
		showLabels,
		showDSOs,
		showPlanets,
		widget.NewSeparator(),
		widget.NewLabel("Eyepiece FOV"),
		fovEntry,
		widget.NewSeparator(),
		refreshBtn,
	)

	p := &SkyChartPanel{
		chart: chart,
		content: container.NewBorder(nil, nil, nil, controls,
			chart,
		),
	}
	return p
}

// Chart returns the underlying SkyChartWidget for external wiring.
func (p *SkyChartPanel) Chart() *skychart.SkyChartWidget { return p.chart }

// Content implements Panel.
func (p *SkyChartPanel) Content() fyne.CanvasObject { return p.content }

// Title implements Panel.
func (p *SkyChartPanel) Title() string { return "Sky Chart" }

// Icon implements Panel.
func (p *SkyChartPanel) Icon() fyne.Resource { return theme.ColorChromaticIcon() }
