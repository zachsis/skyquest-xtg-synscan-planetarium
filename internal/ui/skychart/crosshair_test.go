package skychart

import (
	"image"
	"math"
	"testing"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/telescope"
)

type mockPositionProvider struct {
	state telescope.TelescopeState
}

func (m *mockPositionProvider) CurrentState() telescope.TelescopeState { return m.state }
func (m *mockPositionProvider) Subscribe(func(telescope.TelescopeState)) func() {
	return func() {}
}

func TestCrosshairHiddenWhenDisconnected(t *testing.T) {
	pp := &mockPositionProvider{state: telescope.TelescopeState{Connected: false}}
	overlay := NewCrosshairOverlay(pp)

	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	proj := &ProjectionContext{
		CenterX: 100, CenterY: 100, Radius: 90, Zoom: 1.0,
		AltAz: func(ra, dec float64) (float64, float64) {
			return math.Pi / 4, 0 // arbitrary
		},
	}

	// Should not panic and should not draw anything meaningful.
	overlay.Draw(img, proj)

	// Verify center pixel remains black (no crosshair drawn).
	r, g, b, _ := img.At(100, 100).RGBA()
	if r != 0 || g != 0 || b != 0 {
		t.Error("crosshair should not be drawn when disconnected")
	}
}

func TestCrosshairDrawnWhenConnected(t *testing.T) {
	pp := &mockPositionProvider{state: telescope.TelescopeState{
		Connected: true,
		RA:        6.0,
		Dec:       45.0,
	}}
	overlay := NewCrosshairOverlay(pp)

	img := image.NewRGBA(image.Rect(0, 0, 400, 400))
	proj := &ProjectionContext{
		CenterX: 200, CenterY: 200, Radius: 180, Zoom: 1.0,
		AltAz: func(ra, dec float64) (float64, float64) {
			return math.Pi / 4, math.Pi / 2 // 45° alt, 90° az
		},
	}

	overlay.Draw(img, proj)

	// Compute expected screen position.
	sx, sy := StereoProject(math.Pi/4, math.Pi/2, 180)
	cx := int(math.Round(200 + sx))
	cy := int(math.Round(200 + sy))

	// Check that some crosshair pixels were drawn (offset from center by gap).
	r, g, b, _ := img.At(cx+5, cy).RGBA()
	// crosshairColor is {0, 220, 0, 255} → green channel should be nonzero.
	if g == 0 {
		t.Errorf("expected crosshair pixel at (%d, %d), got black", cx+5, cy)
	}
	if r != 0 || b != 0 {
		t.Errorf("crosshair pixel should be green, got r=%d g=%d b=%d", r>>8, g>>8, b>>8)
	}
}

func TestCrosshairHiddenBelowHorizon(t *testing.T) {
	pp := &mockPositionProvider{state: telescope.TelescopeState{
		Connected: true, RA: 12.0, Dec: -80.0,
	}}
	overlay := NewCrosshairOverlay(pp)

	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	proj := &ProjectionContext{
		CenterX: 100, CenterY: 100, Radius: 90, Zoom: 1.0,
		AltAz: func(ra, dec float64) (float64, float64) {
			return -0.1, 0 // below horizon
		},
	}

	overlay.Draw(img, proj)

	// No crosshair pixels should be drawn. Check along the arms area.
	for d := 3; d <= 10; d++ {
		r, g, b, _ := img.At(100+d, 100).RGBA()
		if g != 0 || r != 0 || b != 0 {
			t.Fatal("crosshair should not be drawn when below horizon")
		}
	}
}

func TestFOVCircleScalesWithRadius(t *testing.T) {
	// Verify that FOV circle screen radius increases linearly with chart radius.
	altRad := math.Pi / 4 // 45°
	fovRad := 60.0 / 60.0 * math.Pi / 180.0

	computeFOVScreenR := func(radius float64) float64 {
		r1 := radius * math.Cos(altRad) / (1.0 + math.Sin(altRad))
		altEdge := altRad - fovRad/2
		r2 := radius * math.Cos(altEdge) / (1.0 + math.Sin(altEdge))
		return math.Abs(r2 - r1)
	}

	r1 := computeFOVScreenR(500)
	r2 := computeFOVScreenR(1000)

	// Should scale proportionally (within 1% tolerance).
	ratio := r2 / r1
	if math.Abs(ratio-2.0) > 0.02 {
		t.Errorf("FOV screen radius should scale linearly: ratio=%f, want ~2.0", ratio)
	}

	// At radius=500, the 60 arcmin circle should be a few pixels.
	if r1 < 1 {
		t.Errorf("FOV circle at radius=500 too small: %f px", r1)
	}
}

func TestSetFOV(t *testing.T) {
	pp := &mockPositionProvider{state: telescope.TelescopeState{Connected: true}}
	overlay := NewCrosshairOverlay(pp)

	overlay.SetFOV(120)
	if overlay.fovArcmin != 120 {
		t.Errorf("FOV: got %g, want 120", overlay.fovArcmin)
	}
}
