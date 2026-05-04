package skychart

import "image"

// OverlayRenderer allows external packages to draw on the sky chart.
type OverlayRenderer interface {
	Draw(img *image.RGBA, proj *ProjectionContext)
	Name() string
	Visible() bool
}

// ProjectionContext provides the projection parameters needed by overlays.
type ProjectionContext struct {
	CenterX float64 // screen center X
	CenterY float64 // screen center Y
	Radius  float64 // effective chart radius (after zoom)
	Zoom    float64
	// AltAz converts J2000 RA/Dec (hours, degrees) to current Alt/Az (radians).
	AltAz func(ra, dec float64) (alt, az float64)
}
