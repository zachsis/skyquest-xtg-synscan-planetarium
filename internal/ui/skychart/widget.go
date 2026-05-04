package skychart

import (
	"image"
	"image/color"
	"math"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/astro"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/config"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/slew"
)

const refreshInterval = 30 * time.Second

// RenderedStar records a star's screen position for hit detection.
type RenderedStar struct {
	Star    catalog.Star
	ScreenX float64
	ScreenY float64
}

// SkyChartWidget is a custom Fyne widget that renders an all-sky chart.
type SkyChartWidget struct {
	widget.BaseWidget

	catalog  *catalog.Catalog
	cfg      *config.Config
	astroSvc *astro.AstroService

	mu       sync.Mutex
	viewport Viewport
	showGrid bool

	overlays      []OverlayRenderer
	highlightedID string
	highlightStart time.Time

	raster        *canvas.Raster
	refreshTicker *time.Ticker

	renderedStars []RenderedStar
	onStarClicked func(catalog.Star)
	slewService   slew.GoToService
}

// NewSkyChartWidget creates a new sky chart widget.
func NewSkyChartWidget(cat *catalog.Catalog, cfg *config.Config, astroSvc *astro.AstroService) *SkyChartWidget {
	w := &SkyChartWidget{
		catalog:  cat,
		cfg:      cfg,
		astroSvc: astroSvc,
		viewport: DefaultViewport(),
		showGrid: true,
	}
	w.ExtendBaseWidget(w)
	w.raster = canvas.NewRaster(w.drawChart)
	w.startAutoRefresh()
	return w
}

// CreateRenderer implements fyne.Widget.
func (w *SkyChartWidget) CreateRenderer() fyne.WidgetRenderer {
	return &skyChartRenderer{widget: w}
}

// Scrolled implements fyne.Scrollable for zoom.
func (w *SkyChartWidget) Scrolled(ev *fyne.ScrollEvent) {
	w.mu.Lock()
	factor := 1.0 + float64(ev.Scrolled.DY)*0.1
	w.viewport.ApplyZoom(factor)
	w.mu.Unlock()
	w.Refresh()
}

// Dragged implements fyne.Draggable for pan.
func (w *SkyChartWidget) Dragged(ev *fyne.DragEvent) {
	w.mu.Lock()
	if w.viewport.Zoom <= MinZoom {
		w.mu.Unlock()
		return
	}
	// Convert pixel drag to Alt/Az delta.
	// Rough approximation: pixels / effective-radius * π/2 → radians.
	size := w.Size()
	baseR := math.Min(float64(size.Width), float64(size.Height)) / 2
	effR := baseR * w.viewport.Zoom
	dAlt := float64(ev.Dragged.DY) / effR * (math.Pi / 2)
	dAz := -float64(ev.Dragged.DX) / effR * (math.Pi / 2)
	w.viewport.Pan(dAlt, dAz)
	w.mu.Unlock()
	w.Refresh()
}

// DragEnd implements fyne.Draggable.
func (w *SkyChartWidget) DragEnd() {}

// CenterOn centers the chart on the given RA/Dec coordinates.
func (w *SkyChartWidget) CenterOn(ra, dec float64) {
	alt, az := w.astroSvc.AltAz(ra, dec, time.Now())
	altRad := alt * math.Pi / 180
	azRad := az * math.Pi / 180
	w.mu.Lock()
	w.viewport.CenterAlt = altRad
	w.viewport.CenterAz = azRad
	if w.viewport.Zoom < 4.0 {
		w.viewport.Zoom = 4.0
	}
	w.mu.Unlock()
	w.Refresh()
}

// HighlightObject marks an object for pulsing highlight rendering.
func (w *SkyChartWidget) HighlightObject(id string) {
	w.mu.Lock()
	w.highlightedID = id
	w.highlightStart = time.Now()
	w.mu.Unlock()
	w.Refresh()
}

// AddOverlay registers an overlay renderer.
func (w *SkyChartWidget) AddOverlay(o OverlayRenderer) {
	w.mu.Lock()
	w.overlays = append(w.overlays, o)
	w.mu.Unlock()
}

// SetShowGrid toggles the Alt/Az grid.
func (w *SkyChartWidget) SetShowGrid(show bool) {
	w.mu.Lock()
	w.showGrid = show
	w.mu.Unlock()
	w.Refresh()
}

// SetOnStarClicked sets the callback for star selection.
func (w *SkyChartWidget) SetOnStarClicked(f func(catalog.Star)) {
	w.mu.Lock()
	w.onStarClicked = f
	w.mu.Unlock()
}

// SetSlewService sets the GoTo service for slewing to selected stars.
func (w *SkyChartWidget) SetSlewService(svc slew.GoToService) {
	w.mu.Lock()
	w.slewService = svc
	w.mu.Unlock()
}

// RenderedStars returns the most recently rendered star positions for hit detection.
func (w *SkyChartWidget) RenderedStars() []RenderedStar {
	return w.renderedStars
}

func (w *SkyChartWidget) startAutoRefresh() {
	w.refreshTicker = time.NewTicker(refreshInterval)
	go func() {
		for range w.refreshTicker.C {
			w.Refresh()
		}
	}()
}

func (w *SkyChartWidget) stopAutoRefresh() {
	if w.refreshTicker != nil {
		w.refreshTicker.Stop()
		w.refreshTicker = nil
	}
}

func (w *SkyChartWidget) drawChart(width, height int) image.Image {
	if width <= 0 || height <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	DrawBackground(img)

	// Chart is a centered square.
	side := math.Min(float64(width), float64(height))
	baseR := side / 2
	cx := float64(width) / 2
	cy := float64(height) / 2

	w.mu.Lock()
	vp := w.viewport
	showGrid := w.showGrid
	overlays := make([]OverlayRenderer, len(w.overlays))
	copy(overlays, w.overlays)
	w.mu.Unlock()

	effR := vp.EffectiveRadius(baseR)

	// Draw horizon circle.
	DrawHorizonCircle(img, cx, cy, baseR)

	// Draw grid.
	if showGrid {
		DrawGrid(img, cx, cy, baseR, vp.Zoom)
	}

	// Compute Alt/Az for all stars and draw above-horizon ones.
	now := time.Now()
	altAzFunc := func(ra, dec float64) (float64, float64) {
		alt, az := w.astroSvc.AltAz(ra, dec, now)
		return alt * math.Pi / 180, az * math.Pi / 180
	}

	var rendered []RenderedStar
	stars := w.catalog.Stars()
	for i := range stars {
		s := &stars[i]
		altRad, azRad := altAzFunc(s.RA, s.Dec)
		if altRad <= 0 {
			continue
		}
		sx, sy := StereoProject(altRad, azRad, effR)
		screenX := cx + sx
		screenY := cy + sy

		// Clip to image bounds with margin.
		if screenX < -10 || screenX > float64(width)+10 || screenY < -10 || screenY > float64(height)+10 {
			continue
		}

		r := StarRadius(s.Mag)
		col := SpectralColor(s.SpectralType)
		DrawStar(img, screenX, screenY, r, col)
		rendered = append(rendered, RenderedStar{Star: *s, ScreenX: screenX, ScreenY: screenY})
	}
	w.renderedStars = rendered

	// Draw overlays.
	projCtx := &ProjectionContext{
		CenterX: cx,
		CenterY: cy,
		Radius:  effR,
		Zoom:    vp.Zoom,
		AltAz:   altAzFunc,
	}
	for _, o := range overlays {
		if o.Visible() {
			o.Draw(img, projCtx)
		}
	}

	// Draw cardinal labels.
	drawCardinalLabels(img, cx, cy, baseR)

	return img
}

func drawCardinalLabels(img *image.RGBA, cx, cy, baseR float64) {
	labels := []struct {
		text string
		az   float64
	}{
		{"N", 0}, {"E", math.Pi / 2}, {"S", math.Pi}, {"W", 3 * math.Pi / 2},
	}

	for _, l := range labels {
		r := baseR + 12
		x := cx - r*math.Sin(l.az)
		y := cy - r*math.Cos(l.az)
		drawLabelChar(img, int(x), int(y), l.text[0], cardinalColor)
	}
}

// drawLabelChar draws a single uppercase letter at the given position using
// a simple 5x7 bitmap font. Only implements N, S, E, W for cardinal labels.
func drawLabelChar(img *image.RGBA, x, y int, ch byte, col color.RGBA) {
	var bitmap [7]byte
	switch ch {
	case 'N':
		bitmap = [7]byte{0x11, 0x19, 0x15, 0x13, 0x11, 0x11, 0x11}
	case 'S':
		bitmap = [7]byte{0x0E, 0x11, 0x10, 0x0E, 0x01, 0x11, 0x0E}
	case 'E':
		bitmap = [7]byte{0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x1F}
	case 'W':
		bitmap = [7]byte{0x11, 0x11, 0x11, 0x15, 0x15, 0x15, 0x0A}
	default:
		return
	}
	startX := x - 2
	startY := y - 3
	for row := 0; row < 7; row++ {
		for col := 0; col < 5; col++ {
			if bitmap[row]&(1<<(4-col)) != 0 {
				setPixel(img, startX+col, startY+row, cardinalColor)
			}
		}
	}
}

// skyChartRenderer is the Fyne widget renderer for SkyChartWidget.
type skyChartRenderer struct {
	widget *SkyChartWidget
}

func (r *skyChartRenderer) Layout(size fyne.Size) {
	r.widget.raster.Resize(size)
	r.widget.raster.Move(fyne.NewPos(0, 0))
}

func (r *skyChartRenderer) MinSize() fyne.Size {
	return fyne.NewSize(300, 300)
}

func (r *skyChartRenderer) Refresh() {
	r.widget.raster.Refresh()
}

func (r *skyChartRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.widget.raster}
}

func (r *skyChartRenderer) Destroy() {
	r.widget.stopAutoRefresh()
}
