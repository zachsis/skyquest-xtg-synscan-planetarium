package ephemeris

import (
	"context"
	"testing"
	"time"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/astro"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
)

// stubLocation implements astro.LocationProvider for tests.
// Uses a mid-latitude northern location (Athens, Greece) to exercise rise/set.
type stubLocation struct{}

func (stubLocation) GetLatitude() float64  { return 37.97 }
func (stubLocation) GetLongitude() float64 { return 23.72 }
func (stubLocation) GetElevation() float64 { return 100 }

func newTestEngine(t *testing.T) *EphemerisEngine {
	t.Helper()
	svc := astro.NewAstroService(stubLocation{})
	engine, err := NewEphemerisEngine(svc)
	if err != nil {
		t.Fatalf("NewEphemerisEngine: %v", err)
	}
	return engine
}

// TestNewEphemerisEngine verifies construction succeeds and returns non-nil engine.
func TestNewEphemerisEngine(t *testing.T) {
	engine := newTestEngine(t)
	engine.Stop()
}

// TestObjectCount verifies that Objects() returns exactly 9 items:
// 1 Sun + 1 Moon + 7 planets (Mercury, Venus, Mars, Jupiter, Saturn, Uranus, Neptune).
func TestObjectCount(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Stop()

	objs := engine.Objects()
	const want = 9
	if len(objs) != want {
		t.Errorf("Objects() returned %d items, want %d", len(objs), want)
	}
}

// TestSource verifies the DynamicObjectProvider source identifier.
func TestSource(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Stop()

	if got := engine.Source(); got != "solar_system" {
		t.Errorf("Source() = %q, want %q", got, "solar_system")
	}
}

// TestObjectTypes checks that exactly 1 Sun, 1 Moon, and 7 planets are present.
func TestObjectTypes(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Stop()

	var suns, moons, planets int
	for _, obj := range engine.Objects() {
		switch obj.Type {
		case catalog.ObjectTypeSun:
			suns++
		case catalog.ObjectTypeMoon:
			moons++
		case catalog.ObjectTypePlanet:
			planets++
		default:
			t.Errorf("unexpected object type %q for %s", obj.Type, obj.Name)
		}
	}

	if suns != 1 {
		t.Errorf("got %d Sun objects, want 1", suns)
	}
	if moons != 1 {
		t.Errorf("got %d Moon objects, want 1", moons)
	}
	if planets != 7 {
		t.Errorf("got %d planet objects, want 7", planets)
	}
}

// TestObjectCatalogIDs checks that all expected solar system bodies are present.
func TestObjectCatalogIDs(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Stop()

	want := []string{
		"Sun", "Moon",
		"Mercury", "Venus", "Mars",
		"Jupiter", "Saturn", "Uranus", "Neptune",
	}

	idSet := make(map[string]bool, len(want))
	for _, obj := range engine.Objects() {
		idSet[obj.CatalogID] = true
	}

	for _, id := range want {
		if !idSet[id] {
			t.Errorf("missing expected CatalogID %q", id)
		}
	}
}

// TestCatalogSource checks all objects report source "solar_system".
func TestCatalogSource(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Stop()

	for _, obj := range engine.Objects() {
		if obj.CatalogSource != "solar_system" {
			t.Errorf("%s: CatalogSource = %q, want %q",
				obj.Name, obj.CatalogSource, "solar_system")
		}
	}
}

// TestStartStop verifies the Start/Stop lifecycle doesn't panic or deadlock.
func TestStartStop(t *testing.T) {
	engine := newTestEngine(t)

	ctx, cancel := context.WithCancel(context.Background())
	engine.Start(ctx)

	// Let the goroutine run briefly.
	time.Sleep(50 * time.Millisecond)

	cancel() // Stop via context cancellation.
	engine.Stop()
}

// TestForceUpdate verifies that ForceUpdate refreshes the stored positions.
// We check that positions are present (non-zero RA is unlikely for all bodies).
func TestForceUpdate(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Stop()

	engine.ForceUpdate()
	planets := engine.Planets()
	if len(planets) == 0 {
		t.Fatal("Planets() returned empty slice after ForceUpdate")
	}

	// At least one body should have a non-zero RA.
	var nonZero int
	for _, p := range planets {
		if p.RAJ2000 != 0 {
			nonZero++
		}
	}
	if nonZero == 0 {
		t.Error("all RAJ2000 values are zero after ForceUpdate — computation may have failed")
	}
}

// TestGetPlanetInfo verifies lookup by name.
func TestGetPlanetInfo(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Stop()

	names := []string{"Sun", "Moon", "Mercury", "Venus", "Mars",
		"Jupiter", "Saturn", "Uranus", "Neptune"}

	for _, name := range names {
		info, ok := engine.GetPlanetInfo(name)
		if !ok {
			t.Errorf("GetPlanetInfo(%q): not found", name)
			continue
		}
		if info.Name != name {
			t.Errorf("GetPlanetInfo(%q).Name = %q", name, info.Name)
		}
	}

	_, ok := engine.GetPlanetInfo("Pluto")
	if ok {
		t.Error("GetPlanetInfo(\"Pluto\") should return false")
	}
}

// TestMoonHasPhase verifies the Moon has a non-empty phase name and reasonable illumination.
func TestMoonHasPhase(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Stop()

	info, ok := engine.GetPlanetInfo("Moon")
	if !ok {
		t.Fatal("Moon not found")
	}

	if info.PhaseName == "" {
		t.Error("Moon.PhaseName is empty")
	}
	if info.Phase < 0 || info.Phase > 1 {
		t.Errorf("Moon.Phase = %v, want [0,1]", info.Phase)
	}
}

// TestSunHasNoPhaseNameAndFullIllumination checks Sun metadata invariants.
func TestSunHasNoPhaseNameAndFullIllumination(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Stop()

	sun, ok := engine.GetPlanetInfo("Sun")
	if !ok {
		t.Fatal("Sun not found")
	}
	if sun.PhaseName != "" {
		t.Errorf("Sun.PhaseName = %q, want empty", sun.PhaseName)
	}
	if sun.Phase != 1.0 {
		t.Errorf("Sun.Phase = %v, want 1.0", sun.Phase)
	}
}

// TestPlanetElongationRange checks all planets have elongation in [0, 180].
func TestPlanetElongationRange(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Stop()

	for _, p := range engine.Planets() {
		if p.Type != catalog.ObjectTypePlanet {
			continue
		}
		if p.Elongation < 0 || p.Elongation > 180 {
			t.Errorf("%s.Elongation = %.2f, want [0, 180]", p.Name, p.Elongation)
		}
	}
}

// TestAngularDiameters checks that angular diameters are positive and
// in plausible ranges for the Sun and Moon.
func TestAngularDiameters(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Stop()

	sun, _ := engine.GetPlanetInfo("Sun")
	// Sun diameter: ~31–32 arcmin = ~1860–1920 arcsec.
	if sun.AngularDiameter < 1800 || sun.AngularDiameter > 2000 {
		t.Errorf("Sun angular diameter = %.1f arcsec, want 1800–2000", sun.AngularDiameter)
	}

	moon, _ := engine.GetPlanetInfo("Moon")
	// Moon diameter: ~29–33 arcmin = ~1740–1980 arcsec.
	if moon.AngularDiameter < 1700 || moon.AngularDiameter > 2100 {
		t.Errorf("Moon angular diameter = %.1f arcsec, want 1700–2100", moon.AngularDiameter)
	}
}
