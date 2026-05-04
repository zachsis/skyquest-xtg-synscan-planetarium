package skychart

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
)

var (
	lineColor  = color.RGBA{80, 120, 160, 180}
	labelColor = color.RGBA{120, 140, 160, 200}
)

// ConstellationOverlay draws constellation stick figures and labels.
type ConstellationOverlay struct {
	cat        *catalog.Catalog
	lines      []catalog.ConstellationLine
	starsByCon map[string][]int

	showLines  bool
	showLabels bool
}

// NewConstellationOverlay creates a constellation overlay.
func NewConstellationOverlay(cat *catalog.Catalog) *ConstellationOverlay {
	return &ConstellationOverlay{
		cat:        cat,
		lines:      catalog.ConstellationLines,
		starsByCon: catalog.StarsByConstellation(catalog.ConstellationLines),
		showLines:  true,
		showLabels: true,
	}
}

// SetShowLines toggles constellation line visibility.
func (o *ConstellationOverlay) SetShowLines(show bool) { o.showLines = show }

// SetShowLabels toggles constellation label visibility.
func (o *ConstellationOverlay) SetShowLabels(show bool) { o.showLabels = show }

// Name implements OverlayRenderer.
func (o *ConstellationOverlay) Name() string { return "Constellations" }

// Visible implements OverlayRenderer.
func (o *ConstellationOverlay) Visible() bool { return o.showLines || o.showLabels }

// Draw implements OverlayRenderer.
func (o *ConstellationOverlay) Draw(img *image.RGBA, proj *ProjectionContext) {
	if o.showLines {
		o.drawLines(img, proj)
	}
	if o.showLabels {
		o.drawLabels(img, proj)
	}
}

func (o *ConstellationOverlay) drawLines(img *image.RGBA, proj *ProjectionContext) {
	for _, line := range o.lines {
		s1, ok1 := o.cat.StarByHipID(line.Hip1)
		s2, ok2 := o.cat.StarByHipID(line.Hip2)
		if !ok1 || !ok2 {
			continue
		}
		alt1, az1 := proj.AltAz(s1.RA, s1.Dec)
		alt2, az2 := proj.AltAz(s2.RA, s2.Dec)
		if alt1 < 0 || alt2 < 0 {
			continue
		}
		sx1, sy1 := StereoProject(alt1, az1, proj.Radius)
		sx2, sy2 := StereoProject(alt2, az2, proj.Radius)
		drawLine(img,
			proj.CenterX+sx1, proj.CenterY+sy1,
			proj.CenterX+sx2, proj.CenterY+sy2,
			lineColor,
		)
	}
}

func (o *ConstellationOverlay) drawLabels(img *image.RGBA, proj *ProjectionContext) {
	for conName, hipIDs := range o.starsByCon {
		var sumX, sumY float64
		var count int
		for _, hipID := range hipIDs {
			s, ok := o.cat.StarByHipID(hipID)
			if !ok {
				continue
			}
			alt, az := proj.AltAz(s.RA, s.Dec)
			if alt < 0 {
				continue
			}
			sx, sy := StereoProject(alt, az, proj.Radius)
			sumX += proj.CenterX + sx
			sumY += proj.CenterY + sy
			count++
		}
		if count < 3 {
			continue // too few visible stars
		}
		cx := sumX / float64(count)
		cy := sumY / float64(count)
		// Offset label slightly below centroid.
		drawText(img, int(cx)-len(conName)*3, int(cy)+8, conName, labelColor)
	}
}

// drawText draws a string using basicfont.Face7x13.
func drawText(img *image.RGBA, x, y int, text string, col color.RGBA) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}

// drawLine draws a line using Bresenham's algorithm — same as renderer.go's
// drawLine but accessible here since both are in the same package.
func drawConstellationLine(img *image.RGBA, x1, y1, x2, y2 float64, col color.RGBA) {
	ix1, iy1 := int(math.Round(x1)), int(math.Round(y1))
	ix2, iy2 := int(math.Round(x2)), int(math.Round(y2))

	dx := abs(ix2 - ix1)
	dy := abs(iy2 - iy1)
	sx, sy := 1, 1
	if ix1 > ix2 {
		sx = -1
	}
	if iy1 > iy2 {
		sy = -1
	}
	err := dx - dy

	for {
		setPixel(img, ix1, iy1, col)
		if ix1 == ix2 && iy1 == iy2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			ix1 += sx
		}
		if e2 < dx {
			err += dx
			iy1 += sy
		}
	}
}
