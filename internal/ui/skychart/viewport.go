package skychart

import "math"

const (
	MinZoom = 1.0
	MaxZoom = 20.0
)

// Viewport manages the pan/zoom state for the sky chart.
type Viewport struct {
	Zoom      float64 // 1.0 = full sky, 20.0 = max zoom
	CenterAlt float64 // radians, default π/2 (zenith)
	CenterAz  float64 // radians, default 0
}

// DefaultViewport returns a viewport centered on the zenith at 1x zoom.
func DefaultViewport() Viewport {
	return Viewport{
		Zoom:      1.0,
		CenterAlt: math.Pi / 2,
		CenterAz:  0,
	}
}

// ApplyZoom multiplies the zoom by the given factor, clamping to [MinZoom, MaxZoom].
func (v *Viewport) ApplyZoom(factor float64) {
	v.Zoom = math.Max(MinZoom, math.Min(MaxZoom, v.Zoom*factor))
}

// Pan shifts the viewport center by the given Alt/Az deltas (radians).
// Alt is clamped to [0, π/2], Az wraps around [0, 2π).
func (v *Viewport) Pan(dAlt, dAz float64) {
	if v.Zoom <= MinZoom {
		return // no pan at full sky
	}
	v.CenterAlt = math.Max(0, math.Min(math.Pi/2, v.CenterAlt+dAlt))
	v.CenterAz += dAz
	for v.CenterAz < 0 {
		v.CenterAz += 2 * math.Pi
	}
	for v.CenterAz >= 2*math.Pi {
		v.CenterAz -= 2 * math.Pi
	}
}

// EffectiveRadius returns the projected radius adjusted for zoom.
func (v *Viewport) EffectiveRadius(baseRadius float64) float64 {
	return baseRadius * v.Zoom
}
