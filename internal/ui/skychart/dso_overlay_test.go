package skychart

import (
	"image/color"
	"math"
	"testing"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
)

// --- PlanetColor tests ---

func TestPlanetColorKnownPlanets(t *testing.T) {
	tests := []struct {
		name string
		want color.RGBA
	}{
		{"Mercury", color.RGBA{180, 180, 180, 255}},
		{"Venus", color.RGBA{255, 255, 200, 255}},
		{"Mars", color.RGBA{255, 100, 80, 255}},
		{"Jupiter", color.RGBA{255, 200, 140, 255}},
		{"Saturn", color.RGBA{255, 220, 130, 255}},
		{"Uranus", color.RGBA{130, 220, 255, 255}},
		{"Neptune", color.RGBA{100, 140, 255, 255}},
		{"Moon", color.RGBA{240, 240, 210, 255}},
		{"Sun", color.RGBA{255, 220, 50, 255}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PlanetColor(tt.name)
			if got != tt.want {
				t.Errorf(
					"PlanetColor(%q) = %v, want %v",
					tt.name, got, tt.want,
				)
			}
		})
	}
}

func TestPlanetColorUnknownBody(t *testing.T) {
	got := PlanetColor("Pluto")
	// Unknown bodies must return the non-zero default colour (not black).
	if got == (color.RGBA{}) {
		t.Error("PlanetColor(unknown) returned zero colour, expected default")
	}
}

// --- FindNearestObject tests ---

func TestFindNearestObjectWithinRadius(t *testing.T) {
	objects := []VisibleObject{
		{ScreenX: 100, ScreenY: 100, Object: catalog.CelestialObject{Name: "M31"}},
		{ScreenX: 200, ScreenY: 200, Object: catalog.CelestialObject{Name: "M42"}},
	}

	result := FindNearestObject(objects, 103, 104)
	if result == nil {
		t.Fatal("expected to find M31")
	}
	if result.Object.Name != "M31" {
		t.Errorf("expected M31, got %s", result.Object.Name)
	}
}

func TestFindNearestObjectOutsideRadius(t *testing.T) {
	objects := []VisibleObject{
		{ScreenX: 100, ScreenY: 100, Object: catalog.CelestialObject{Name: "M31"}},
	}

	// 28px away – well outside hitRadiusPx=15.
	result := FindNearestObject(objects, 120, 120)
	dist := math.Sqrt(20*20 + 20*20)
	if dist <= hitRadiusPx {
		t.Fatal("test setup error: click should be outside hit radius")
	}
	if result != nil {
		t.Errorf("expected nil for click outside hit radius, got %s", result.Object.Name)
	}
}

func TestFindNearestObjectEmpty(t *testing.T) {
	result := FindNearestObject(nil, 100, 100)
	if result != nil {
		t.Error("expected nil for empty visible list")
	}
}

func TestFindNearestObjectSelectsClosest(t *testing.T) {
	objects := []VisibleObject{
		{ScreenX: 100, ScreenY: 100, Object: catalog.CelestialObject{Name: "Far"}},
		{ScreenX: 108, ScreenY: 100, Object: catalog.CelestialObject{Name: "Near"}},
	}

	// (109, 100): 9px from Far, 1px from Near.
	result := FindNearestObject(objects, 109, 100)
	if result == nil {
		t.Fatal("expected to find an object")
	}
	if result.Object.Name != "Near" {
		t.Errorf("expected Near (closest), got %s", result.Object.Name)
	}
}

func TestFindNearestObjectAtExactRadius(t *testing.T) {
	objects := []VisibleObject{
		{ScreenX: 100, ScreenY: 100, Object: catalog.CelestialObject{Name: "M1"}},
	}

	// Exactly hitRadiusPx away (horizontal).
	result := FindNearestObject(objects, 100+hitRadiusPx, 100)
	if result == nil {
		t.Fatal("expected to find object at exact hit radius boundary")
	}
}

// --- shouldRender tests ---

func TestDSOOverlayShouldRenderDSOTypes(t *testing.T) {
	o := &DSOOverlay{showDSOs: true, showPlanets: false}

	dsoTypes := []catalog.ObjectType{
		catalog.ObjectTypeGalaxy,
		catalog.ObjectTypeNebula,
		catalog.ObjectTypePlanetaryNeb,
		catalog.ObjectTypeOpenCluster,
		catalog.ObjectTypeGlobularClust,
		catalog.ObjectTypeSupernovaRem,
	}

	for _, typ := range dsoTypes {
		obj := &catalog.CelestialObject{Type: typ}
		if !o.shouldRender(obj, true, false) {
			t.Errorf("shouldRender(%s) = false with showDSOs=true", typ)
		}
		if o.shouldRender(obj, false, true) {
			t.Errorf("shouldRender(%s) = true with showDSOs=false", typ)
		}
	}
}

func TestDSOOverlayShouldRenderPlanetTypes(t *testing.T) {
	o := &DSOOverlay{showDSOs: false, showPlanets: true}

	solarTypes := []catalog.ObjectType{
		catalog.ObjectTypePlanet,
		catalog.ObjectTypeMoon,
		catalog.ObjectTypeSun,
	}

	for _, typ := range solarTypes {
		obj := &catalog.CelestialObject{Type: typ}
		if !o.shouldRender(obj, false, true) {
			t.Errorf("shouldRender(%s) = false with showPlanets=true", typ)
		}
		if o.shouldRender(obj, true, false) {
			t.Errorf("shouldRender(%s) = true with showPlanets=false", typ)
		}
	}
}

func TestDSOOverlayShouldNotRenderNamedStars(t *testing.T) {
	// named_stars are filtered before shouldRender, but ensure star type
	// itself returns false (not a DSO or planet).
	o := &DSOOverlay{}
	obj := &catalog.CelestialObject{Type: catalog.ObjectTypeStar}
	if o.shouldRender(obj, true, true) {
		t.Error("shouldRender(star) should return false")
	}
}
