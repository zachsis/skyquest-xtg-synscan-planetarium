package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/astro"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/config"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/ui/skychart"
)

// SkyChartPanel wraps the SkyChartWidget into a Panel with overlay controls.
type SkyChartPanel struct {
	chart   *skychart.SkyChartWidget
	content fyne.CanvasObject
}

// NewSkyChartPanel creates the sky chart panel.
func NewSkyChartPanel(cat *catalog.Catalog, cfg *config.Config, astroSvc *astro.AstroService) *SkyChartPanel {
	chart := skychart.NewSkyChartWidget(cat, cfg, astroSvc)

	// Register constellation overlay.
	conOverlay := skychart.NewConstellationOverlay(cat)
	chart.AddOverlay(conOverlay)

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

	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		chart.Refresh()
	})

	controls := container.NewVBox(
		widget.NewLabel("Overlays"),
		widget.NewSeparator(),
		showGrid,
		showLines,
		showLabels,
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
