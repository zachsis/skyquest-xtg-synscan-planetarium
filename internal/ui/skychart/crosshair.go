package skychart

import (
	"image"
	"image/color"
	"math"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/telescope"
)

var crosshairColor = color.RGBA{0, 220, 0, 255}
var fovColor = color.RGBA{0, 180, 0, 100}

// CrosshairOverlay draws the telescope position crosshair and FOV circle.
type CrosshairOverlay struct {
	pos       telescope.PositionProvider
	fovArcmin float64 // eyepiece FOV in arcminutes
	visible   bool
}

// NewCrosshairOverlay creates a crosshair overlay subscribed to live position.
func NewCrosshairOverlay(pos telescope.PositionProvider) *CrosshairOverlay {
	return &CrosshairOverlay{
		pos:       pos,
		fovArcmin: 60,
		visible:   true,
	}
}

// SetFOV sets the eyepiece field of view in arcminutes.
func (o *CrosshairOverlay) SetFOV(arcmin float64) { o.fovArcmin = arcmin }

// Name implements OverlayRenderer.
func (o *CrosshairOverlay) Name() string { return "Crosshair" }

// Visible implements OverlayRenderer.
func (o *CrosshairOverlay) Visible() bool { return o.visible }

// Draw implements OverlayRenderer.
func (o *CrosshairOverlay) Draw(img *image.RGBA, proj *ProjectionContext) {
	state := o.pos.CurrentState()
	if !state.Connected {
		return
	}

	altRad, azRad := proj.AltAz(state.RA, state.Dec)
	if altRad < 0 {
		return // below horizon
	}

	sx, sy := StereoProject(altRad, azRad, proj.Radius)
	cx := int(math.Round(proj.CenterX + sx))
	cy := int(math.Round(proj.CenterY + sy))

	// Draw crosshair lines (20px each arm, 2px center gap).
	for d := 3; d <= 10; d++ {
		setPixel(img, cx+d, cy, crosshairColor)
		setPixel(img, cx-d, cy, crosshairColor)
		setPixel(img, cx, cy+d, crosshairColor)
		setPixel(img, cx, cy-d, crosshairColor)
	}

	// Draw FOV circle.
	if o.fovArcmin > 0 {
		fovRad := o.fovArcmin / 60.0 * math.Pi / 180.0
		// Compute projected size of FOV at this altitude.
		r1 := proj.Radius * math.Cos(altRad) / (1.0 + math.Sin(altRad))
		altEdge := altRad - fovRad/2
		if altEdge < 0 {
			altEdge = 0
		}
		r2 := proj.Radius * math.Cos(altEdge) / (1.0 + math.Sin(altEdge))
		fovR := math.Abs(r2 - r1)
		if fovR > 2 {
			drawCircle(img, float64(cx), float64(cy), fovR, fovColor)
		}
	}
}
