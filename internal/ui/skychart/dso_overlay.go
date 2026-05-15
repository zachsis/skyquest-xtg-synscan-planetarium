package skychart

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/config"
)

var (
	dsoOutlineColor = color.RGBA{100, 180, 200, 200}
	dsoLabelColor   = color.RGBA{140, 210, 230, 220}
)

// planetColors maps planet/solar-system body names to their display colour.
var planetColors = map[string]color.RGBA{
	"Mercury": {180, 180, 180, 255},
	"Venus":   {255, 255, 200, 255},
	"Mars":    {255, 100, 80, 255},
	"Jupiter": {255, 200, 140, 255},
	"Saturn":  {255, 220, 130, 255},
	"Uranus":  {130, 220, 255, 255},
	"Neptune": {100, 140, 255, 255},
	"Moon":    {240, 240, 210, 255},
	"Sun":     {255, 220, 50, 255},
}

// defaultSolarColor is used for solar-system objects not found in planetColors.
var defaultSolarColor = color.RGBA{255, 200, 100, 255}

// VisibleObject records a rendered DSO/planet screen position for hit detection.
type VisibleObject struct {
	ScreenX float64
	ScreenY float64
	Object  catalog.CelestialObject
}

// DSOOverlay draws deep-sky objects and solar-system objects from the registry.
// It implements OverlayRenderer.
type DSOOverlay struct {
	registry    *catalog.CatalogRegistry
	cfg         *config.Config
	showDSOs    bool
	showPlanets bool

	mu             sync.Mutex
	visibleObjects []VisibleObject
}

// NewDSOOverlay creates a DSOOverlay with both DSOs and planets visible.
func NewDSOOverlay(registry *catalog.CatalogRegistry, cfg *config.Config) *DSOOverlay {
	return &DSOOverlay{
		registry:    registry,
		cfg:         cfg,
		showDSOs:    true,
		showPlanets: true,
	}
}

// SetShowDSOs toggles DSO (galaxy, nebula, cluster) rendering.
func (o *DSOOverlay) SetShowDSOs(show bool) {
	o.mu.Lock()
	o.showDSOs = show
	o.mu.Unlock()
}

// SetShowPlanets toggles planet/Moon/Sun rendering.
func (o *DSOOverlay) SetShowPlanets(show bool) {
	o.mu.Lock()
	o.showPlanets = show
	o.mu.Unlock()
}

// Name implements OverlayRenderer.
func (o *DSOOverlay) Name() string { return "DSO & Planets" }

// Visible implements OverlayRenderer.
func (o *DSOOverlay) Visible() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.showDSOs || o.showPlanets
}

// VisibleObjects returns a copy of the currently rendered object list.
// Safe to call from any goroutine.
func (o *DSOOverlay) VisibleObjects() []VisibleObject {
	o.mu.Lock()
	defer o.mu.Unlock()
	cp := make([]VisibleObject, len(o.visibleObjects))
	copy(cp, o.visibleObjects)
	return cp
}

// Draw implements OverlayRenderer. Called from the render goroutine.
func (o *DSOOverlay) Draw(img *image.RGBA, proj *ProjectionContext) {
	o.mu.Lock()
	showDSOs := o.showDSOs
	showPlanets := o.showPlanets
	o.visibleObjects = o.visibleObjects[:0]
	o.mu.Unlock()

	objects := o.registry.AllObjects()

	var rendered []VisibleObject
	for _, obj := range objects {
		// Skip named_stars to avoid duplicate rendering with HYG catalog.
		if obj.CatalogSource == "named_stars" {
			continue
		}

		if !o.shouldRender(&obj, showDSOs, showPlanets) {
			continue
		}

		altRad, azRad := proj.AltAz(obj.RAJ2000, obj.DecJ2000)
		if altRad <= 0 {
			continue
		}

		sx, sy := StereoProject(altRad, azRad, proj.Radius)
		screenX := proj.CenterX + sx
		screenY := proj.CenterY + sy

		// Clip objects well outside the image to avoid expensive drawing.
		bounds := img.Bounds()
		margin := 20.0
		if screenX < float64(bounds.Min.X)-margin ||
			screenX > float64(bounds.Max.X)+margin ||
			screenY < float64(bounds.Min.Y)-margin ||
			screenY > float64(bounds.Max.Y)+margin {
			continue
		}

		o.drawObject(img, screenX, screenY, &obj)

		// Draw label for bright objects.
		if o.cfg != nil && obj.HasMagnitude() &&
			obj.Magnitude <= o.cfg.MagnitudeCutoff {
			label := obj.DisplayName()
			drawText(img, int(screenX)+8, int(screenY)+4, label, dsoLabelColor)
		}

		rendered = append(rendered, VisibleObject{
			ScreenX: screenX,
			ScreenY: screenY,
			Object:  obj,
		})
	}

	o.mu.Lock()
	o.visibleObjects = rendered
	o.mu.Unlock()
}

// shouldRender returns true if this object should be rendered given current
// toggle state.
func (o *DSOOverlay) shouldRender(
	obj *catalog.CelestialObject,
	showDSOs, showPlanets bool,
) bool {
	switch obj.Type {
	case catalog.ObjectTypePlanet,
		catalog.ObjectTypeMoon,
		catalog.ObjectTypeSun:
		return showPlanets
	case catalog.ObjectTypeGalaxy,
		catalog.ObjectTypeNebula,
		catalog.ObjectTypePlanetaryNeb,
		catalog.ObjectTypeOpenCluster,
		catalog.ObjectTypeGlobularClust,
		catalog.ObjectTypeSupernovaRem:
		return showDSOs
	default:
		return false
	}
}

// drawObject dispatches to the appropriate shape renderer based on ObjectType.
func (o *DSOOverlay) drawObject(
	img *image.RGBA,
	cx, cy float64,
	obj *catalog.CelestialObject,
) {
	switch obj.Type {
	case catalog.ObjectTypeGalaxy:
		drawEllipseOutline(img, cx, cy, 9, 5, dsoOutlineColor)
	case catalog.ObjectTypeNebula,
		catalog.ObjectTypePlanetaryNeb,
		catalog.ObjectTypeSupernovaRem:
		drawSquareOutline(img, cx, cy, 8, dsoOutlineColor)
	case catalog.ObjectTypeOpenCluster:
		drawDottedCircle(img, cx, cy, 7, dsoOutlineColor)
	case catalog.ObjectTypeGlobularClust:
		drawCircleCross(img, cx, cy, 7, dsoOutlineColor)
	case catalog.ObjectTypeSun:
		col := planetColors["Sun"]
		drawCircle(img, cx, cy, 7, col)
		// Radiating lines at N/S/E/W.
		for _, angle := range []float64{0, math.Pi / 2, math.Pi, 3 * math.Pi / 2} {
			x1 := cx + 9*math.Sin(angle)
			y1 := cy - 9*math.Cos(angle)
			x2 := cx + 13*math.Sin(angle)
			y2 := cy - 13*math.Cos(angle)
			drawLine(img, x1, y1, x2, y2, col)
		}
	case catalog.ObjectTypeMoon:
		col := planetColors["Moon"]
		fillCircle(img, cx, cy, 7, col)
	case catalog.ObjectTypePlanet:
		col := planetColor(obj.Name)
		fillCircle(img, cx, cy, 6, col)
	}
}

// PlanetColor returns the display colour for a named solar-system body.
// Exported so tests and popup code can use it.
func PlanetColor(name string) color.RGBA {
	return planetColor(name)
}

func planetColor(name string) color.RGBA {
	if c, ok := planetColors[name]; ok {
		return c
	}
	return defaultSolarColor
}

// --- shape helpers ---

// drawEllipseOutline draws a horizontal ellipse outline using the midpoint
// ellipse algorithm.
func drawEllipseOutline(
	img *image.RGBA,
	cx, cy, rx, ry float64,
	col color.RGBA,
) {
	// Midpoint ellipse algorithm.
	x := 0.0
	y := ry
	rx2 := rx * rx
	ry2 := ry * ry
	twoRx2 := 2 * rx2
	twoRy2 := 2 * ry2
	p := ry2 - rx2*ry + 0.25*rx2

	dx := 0.0
	dy := twoRx2 * y

	plotEllipsePoints(img, cx, cy, x, y, col)

	// Region 1.
	for dx < dy {
		x++
		dx += twoRy2
		if p < 0 {
			p += ry2 + dx
		} else {
			y--
			dy -= twoRx2
			p += ry2 + dx - dy
		}
		plotEllipsePoints(img, cx, cy, x, y, col)
	}

	// Region 2.
	p = ry2*(x+0.5)*(x+0.5) + rx2*(y-1)*(y-1) - rx2*ry2

	for y > 0 {
		y--
		dy -= twoRx2
		if p > 0 {
			p += rx2 - dy
		} else {
			x++
			dx += twoRy2
			p += rx2 - dy + dx
		}
		plotEllipsePoints(img, cx, cy, x, y, col)
	}
}

func plotEllipsePoints(img *image.RGBA, cx, cy, x, y float64, col color.RGBA) {
	icx := int(math.Round(cx))
	icy := int(math.Round(cy))
	ix := int(math.Round(x))
	iy := int(math.Round(y))
	setPixel(img, icx+ix, icy+iy, col)
	setPixel(img, icx-ix, icy+iy, col)
	setPixel(img, icx+ix, icy-iy, col)
	setPixel(img, icx-ix, icy-iy, col)
}

// drawSquareOutline draws an axis-aligned square outline centred at (cx, cy).
func drawSquareOutline(img *image.RGBA, cx, cy, half float64, col color.RGBA) {
	x0 := cx - half
	y0 := cy - half
	x1 := cx + half
	y1 := cy + half
	drawLine(img, x0, y0, x1, y0, col)
	drawLine(img, x1, y0, x1, y1, col)
	drawLine(img, x1, y1, x0, y1, col)
	drawLine(img, x0, y1, x0, y0, col)
}

// drawDottedCircle draws 12 small filled dots evenly spaced around a circle.
func drawDottedCircle(img *image.RGBA, cx, cy, r float64, col color.RGBA) {
	const nDots = 12
	for i := 0; i < nDots; i++ {
		angle := float64(i) * 2 * math.Pi / nDots
		dx := r * math.Sin(angle)
		dy := -r * math.Cos(angle)
		fillCircle(img, cx+dx, cy+dy, 1.5, col)
	}
}

// drawCircleCross draws a circle with a horizontal and vertical line through
// its centre (globular cluster symbol).
func drawCircleCross(img *image.RGBA, cx, cy, r float64, col color.RGBA) {
	drawCircle(img, cx, cy, r, col)
	drawLine(img, cx-r, cy, cx+r, cy, col)
	drawLine(img, cx, cy-r, cx, cy+r, col)
}
