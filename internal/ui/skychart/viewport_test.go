package skychart

import (
	"math"
	"testing"
)

func TestDefaultViewport(t *testing.T) {
	v := DefaultViewport()
	if v.Zoom != 1.0 {
		t.Errorf("zoom: got %g, want 1.0", v.Zoom)
	}
	if math.Abs(v.CenterAlt-math.Pi/2) > 1e-10 {
		t.Errorf("alt: got %g, want π/2", v.CenterAlt)
	}
}

func TestZoomClamping(t *testing.T) {
	v := DefaultViewport()
	v.ApplyZoom(100)
	if v.Zoom != MaxZoom {
		t.Errorf("max clamp: got %g, want %g", v.Zoom, MaxZoom)
	}
	v.ApplyZoom(0.001)
	if v.Zoom != MinZoom {
		t.Errorf("min clamp: got %g, want %g", v.Zoom, MinZoom)
	}
}

func TestPanDisabledAtMinZoom(t *testing.T) {
	v := DefaultViewport()
	origAlt := v.CenterAlt
	origAz := v.CenterAz
	v.Pan(0.1, 0.1)
	if v.CenterAlt != origAlt || v.CenterAz != origAz {
		t.Error("pan should be disabled at min zoom")
	}
}

func TestPanClampsAlt(t *testing.T) {
	v := Viewport{Zoom: 5.0, CenterAlt: math.Pi / 4, CenterAz: 0}
	v.Pan(10, 0) // way above zenith
	if v.CenterAlt > math.Pi/2+1e-10 {
		t.Errorf("alt should clamp to π/2, got %g", v.CenterAlt)
	}
	v.Pan(-100, 0) // way below horizon
	if v.CenterAlt < -1e-10 {
		t.Errorf("alt should clamp to 0, got %g", v.CenterAlt)
	}
}

func TestPanWrapsAz(t *testing.T) {
	v := Viewport{Zoom: 5.0, CenterAlt: math.Pi / 4, CenterAz: 0.1}
	v.Pan(0, -0.5) // should wrap around
	if v.CenterAz < 0 || v.CenterAz >= 2*math.Pi {
		t.Errorf("az should wrap: got %g", v.CenterAz)
	}
}
