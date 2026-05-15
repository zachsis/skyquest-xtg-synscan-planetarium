package ui

import (
	"context"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/astro"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/config"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/ephemeris"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/slew"
)

// ObjectBrowserPanel is the full object browser panel. It implements Panel.
type ObjectBrowserPanel struct {
	content fyne.CanvasObject

	registry  *catalog.CatalogRegistry
	engine    *ephemeris.EphemerisEngine
	astroSvc  *astro.AstroService
	cfg       *config.Config
	slewSvc   slew.GoToService
	window    fyne.Window

	// Sub-components.
	search  *searchBar
	filters *objectFilterBar
	table   *objectsTable
	detail  *detailPanel

	// Sorted/filtered result cache.
	results []BrowserResult
	state   FilterState

	// Altitude refresh cancellation.
	cancelRefresh context.CancelFunc

	// OnShowOnChart is called when the user clicks "Show on Chart".
	OnShowOnChart func(ra, dec float64, catalogID string)
}

// NewObjectBrowserPanel constructs the object browser panel.
// engine and slewSvc may be nil if the respective subsystems failed to
// initialise.
func NewObjectBrowserPanel(
	registry *catalog.CatalogRegistry,
	engine *ephemeris.EphemerisEngine,
	astroSvc *astro.AstroService,
	cfg *config.Config,
	slewSvc slew.GoToService,
	window fyne.Window,
) *ObjectBrowserPanel {
	p := &ObjectBrowserPanel{
		registry: registry,
		engine:   engine,
		astroSvc: astroSvc,
		cfg:      cfg,
		slewSvc:  slewSvc,
		window:   window,
		state: FilterState{
			SortField:     "name",
			SortAscending: true,
		},
	}

	// Build sub-components.
	p.search = newSearchBar(func(text string) {
		p.state.Search = text
		p.applyFilters()
	})

	p.filters = newObjectFilterBar(func() {
		p.state.Types = p.filters.types()
		p.state.CatalogSource = p.filters.catalogSource()
		p.state.MaxMagnitude = p.filters.maxMagnitude()
		p.state.AboveHorizon = p.filters.aboveHorizon()
		p.applyFilters()
	})

	p.table = newObjectsTable(
		func(r BrowserResult) {
			p.detail.setResult(r)
		},
		func(field string, asc bool) {
			p.state.SortField = field
			p.state.SortAscending = asc
			p.applyFilters()
		},
	)

	p.detail = newDetailPanel(slewSvc, window)
	p.detail.onShowOnChart = func(ra, dec float64, catalogID string) {
		if p.OnShowOnChart != nil {
			p.OnShowOnChart(ra, dec, catalogID)
		}
	}

	// "Tonight's Best" button.
	tonightBtn := widget.NewButton("Tonight's Best", p.applyTonightsBest)
	p.filters.tonightBtn = tonightBtn

	// Layout.
	toolbar := container.NewVBox(
		buildToolbarRow1(p.search, p.filters),
		buildToolbarRow2(p.filters, tonightBtn),
	)

	split := container.NewVSplit(
		container.NewBorder(toolbar, nil, nil, nil, p.table.table),
		container.NewVScroll(p.detail.content),
	)
	split.SetOffset(0.65)

	p.content = split

	// Run initial filter pass to populate the table.
	p.applyFilters()

	// Start periodic altitude refresh.
	ctx, cancel := context.WithCancel(context.Background())
	p.cancelRefresh = cancel
	go p.altitudeRefreshLoop(ctx)

	return p
}

// Content implements Panel.
func (p *ObjectBrowserPanel) Content() fyne.CanvasObject { return p.content }

// Title implements Panel.
func (p *ObjectBrowserPanel) Title() string { return "Objects" }

// Icon implements Panel.
func (p *ObjectBrowserPanel) Icon() fyne.Resource { return theme.SearchIcon() }

// applyFilters recomputes the filtered+sorted results and refreshes the table.
func (p *ObjectBrowserPanel) applyFilters() {
	if p.registry == nil {
		return
	}
	objects := p.registry.AllObjects()
	p.results = filterAndSort(objects, p.state, p.astroSvc, p.engine)
	p.table.setResults(p.results)
}

// applyTonightsBest applies the "Tonight's Best" preset:
// above horizon, magnitude < cutoff, sort by altitude descending.
func (p *ObjectBrowserPanel) applyTonightsBest() {
	cutoff := 10.0
	if p.cfg != nil && p.cfg.MagnitudeCutoff > 0 {
		cutoff = p.cfg.MagnitudeCutoff
	}

	p.filters.horizonCheck.SetChecked(true)
	p.filters.typeSelect.SetSelected("All Types")
	p.filters.catalogSelect.SetSelected("All Catalogs")

	// Find and set the closest magnitude option.
	best := "< 10.0"
	for _, opt := range magnitudeFilterOptions {
		if opt.value == cutoff {
			best = opt.label
			break
		}
	}
	p.filters.magSelect.SetSelected(best)

	p.state.AboveHorizon = true
	p.state.Types = nil
	p.state.CatalogSource = ""
	p.state.MaxMagnitude = cutoff
	p.state.SortField = "altitude"
	p.state.SortAscending = true // ascending=true means highest first for altitude

	p.table.sortField = "altitude"
	p.table.sortAscending = true

	p.applyFilters()
}

// altitudeRefreshLoop recomputes altitudes every 60 seconds.
func (p *ObjectBrowserPanel) altitudeRefreshLoop(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.applyFilters()
		}
	}
}
