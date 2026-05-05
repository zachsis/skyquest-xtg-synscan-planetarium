package skychart

import (
	"math"
	"testing"
)

func TestStereoProjectZenith(t *testing.T) {
	x, y := StereoProject(math.Pi/2, 0, 100)
	if math.Abs(x) > 1e-10 || math.Abs(y) > 1e-10 {
		t.Errorf("zenith: got (%g, %g), want (0, 0)", x, y)
	}
}

func TestStereoProjectHorizonNorth(t *testing.T) {
	x, y := StereoProject(0, 0, 100)
	// Horizon north: r=100, az=0 → x=0, y=-100
	if math.Abs(x) > 1e-6 {
		t.Errorf("horizon north x: got %g, want 0", x)
	}
	if math.Abs(y+100) > 1e-6 {
		t.Errorf("horizon north y: got %g, want -100", y)
	}
}

func TestStereoProjectHorizonEast(t *testing.T) {
	x, y := StereoProject(0, math.Pi/2, 100)
	// East (az=π/2): x = -100*sin(π/2) = -100, y = -100*cos(π/2) = 0
	if math.Abs(x+100) > 1e-6 {
		t.Errorf("horizon east x: got %g, want -100", x)
	}
	if math.Abs(y) > 1e-6 {
		t.Errorf("horizon east y: got %g, want 0", y)
	}
}

func TestStereoRoundTrip(t *testing.T) {
	cases := []struct {
		alt, az float64
	}{
		{math.Pi / 2, 0},            // zenith
		{math.Pi / 4, 0},            // 45° alt, north
		{math.Pi / 6, math.Pi},      // 30° alt, south
		{math.Pi / 3, 3 * math.Pi / 2}, // 60° alt, west
		{0.1, 1.5},                  // near horizon
	}

	for _, tc := range cases {
		x, y := StereoProject(tc.alt, tc.az, 200)
		gotAlt, gotAz := StereoUnproject(x, y, 200)
		if math.Abs(gotAlt-tc.alt) > 1e-6 {
			t.Errorf("alt round-trip: in=%g, out=%g", tc.alt, gotAlt)
		}
		// Normalize azimuth comparison.
		diff := math.Abs(gotAz - tc.az)
		if diff > math.Pi {
			diff = 2*math.Pi - diff
		}
		if diff > 1e-6 {
			t.Errorf("az round-trip: in=%g, out=%g", tc.az, gotAz)
		}
	}
}
