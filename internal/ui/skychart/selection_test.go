package skychart

import (
	"math"
	"testing"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
)

func TestFindNearestStarWithinRadius(t *testing.T) {
	rendered := []RenderedStar{
		{Star: catalog.Star{HipID: 1, Name: "Alpha"}, ScreenX: 100, ScreenY: 100},
		{Star: catalog.Star{HipID: 2, Name: "Beta"}, ScreenX: 200, ScreenY: 200},
		{Star: catalog.Star{HipID: 3, Name: "Gamma"}, ScreenX: 150, ScreenY: 150},
	}

	// Click near Alpha.
	result := FindNearestStar(rendered, 105, 103)
	if result == nil {
		t.Fatal("expected to find Alpha")
	}
	if result.Star.HipID != 1 {
		t.Errorf("expected HIP 1, got HIP %d", result.Star.HipID)
	}
}

func TestFindNearestStarOutsideRadius(t *testing.T) {
	rendered := []RenderedStar{
		{Star: catalog.Star{HipID: 1}, ScreenX: 100, ScreenY: 100},
	}

	// Click far from the star (>15px).
	result := FindNearestStar(rendered, 120, 120)
	dist := math.Sqrt(20*20 + 20*20) // ~28px
	if dist <= hitRadiusPx {
		t.Fatal("test setup error: expected click to be outside hit radius")
	}
	if result != nil {
		t.Error("expected nil for click outside hit radius")
	}
}

func TestFindNearestStarEmpty(t *testing.T) {
	result := FindNearestStar(nil, 100, 100)
	if result != nil {
		t.Error("expected nil for empty rendered list")
	}
}

func TestFindNearestStarSelectsClosest(t *testing.T) {
	rendered := []RenderedStar{
		{Star: catalog.Star{HipID: 1}, ScreenX: 100, ScreenY: 100},
		{Star: catalog.Star{HipID: 2}, ScreenX: 108, ScreenY: 100},
	}

	// Click at (109, 100) — closer to HIP 2 (1px) than HIP 1 (9px).
	result := FindNearestStar(rendered, 109, 100)
	if result == nil {
		t.Fatal("expected to find a star")
	}
	if result.Star.HipID != 2 {
		t.Errorf("expected HIP 2 (closest), got HIP %d", result.Star.HipID)
	}
}

func TestFindNearestStarExactlyAtRadius(t *testing.T) {
	rendered := []RenderedStar{
		{Star: catalog.Star{HipID: 1}, ScreenX: 100, ScreenY: 100},
	}

	// Click exactly 15px away (horizontal).
	result := FindNearestStar(rendered, 115, 100)
	if result == nil {
		t.Fatal("expected to find star at exactly hit radius boundary")
	}
	if result.Star.HipID != 1 {
		t.Errorf("expected HIP 1, got HIP %d", result.Star.HipID)
	}
}

func TestFormatRA(t *testing.T) {
	// Sirius: RA ~6.752 hours → 06:45:07.2
	got := formatRA(6.752)
	if got != "06:45:07.2" {
		t.Errorf("formatRA(6.752) = %q, want 06:45:07.2", got)
	}
}

func TestFormatDec(t *testing.T) {
	got := formatDec(-16.7161)
	// -16.7161° → -16:42:57.96 ≈ -16:42:58.0
	if len(got) == 0 || got[0] != '-' {
		t.Errorf("formatDec(-16.7161) = %q, want negative prefix", got)
	}
}
