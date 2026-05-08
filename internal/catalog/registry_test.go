package catalog

import (
	"math"
	"testing"
)

func loadedRegistry(t *testing.T) *CatalogRegistry {
	t.Helper()
	r := NewCatalogRegistry()
	if err := r.LoadAll(); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	return r
}

func TestRegistryLoadAll(t *testing.T) {
	r := loadedRegistry(t)
	if r.Len() < 200 {
		t.Errorf("registry has %d objects, want >= 200", r.Len())
	}
}

func TestGetByIDMessier(t *testing.T) {
	r := loadedRegistry(t)
	obj, ok := r.GetByID("M31")
	if !ok {
		t.Fatal("M31 not found")
	}
	if obj.Name != "Andromeda Galaxy" {
		t.Errorf("M31 name: got %q", obj.Name)
	}
	// RA should be ~0.712h, Dec ~+41.27°.
	if math.Abs(obj.RAJ2000-0.712) > 0.01 {
		t.Errorf("M31 RA: got %g, want ~0.712", obj.RAJ2000)
	}
	if math.Abs(obj.DecJ2000-41.27) > 0.1 {
		t.Errorf("M31 Dec: got %g, want ~41.27", obj.DecJ2000)
	}
	if obj.CatalogSource != "messier" {
		t.Errorf("M31 source: got %q", obj.CatalogSource)
	}
}

func TestGetByIDAlternate(t *testing.T) {
	r := loadedRegistry(t)
	// M31 should also be findable as NGC 224.
	obj, ok := r.GetByID("NGC 224")
	if !ok {
		t.Fatal("NGC 224 (M31 alternate ID) not found")
	}
	if obj.CatalogID != "M31" {
		t.Errorf("expected M31 via NGC 224 lookup, got %q", obj.CatalogID)
	}
}

func TestGetByIDMissing(t *testing.T) {
	r := loadedRegistry(t)
	_, ok := r.GetByID("XYZ 9999")
	if ok {
		t.Error("expected not found for nonexistent ID")
	}
}

func TestSearchByNameOrion(t *testing.T) {
	r := loadedRegistry(t)
	results := r.SearchByName("orion")
	if len(results) == 0 {
		t.Fatal("SearchByName('orion') returned no results")
	}
	// Should include M42 (Orion Nebula) and Betelgeuse.
	found42, foundBetel := false, false
	for _, obj := range results {
		if obj.CatalogID == "M42" {
			found42 = true
		}
		if obj.Name == "Betelgeuse" {
			foundBetel = true
		}
	}
	if !found42 {
		t.Error("M42 (Orion Nebula) not in 'orion' search results")
	}
	if !foundBetel {
		t.Error("Betelgeuse not in 'orion' search results")
	}
}

func TestSearchByNameCaseInsensitive(t *testing.T) {
	r := loadedRegistry(t)
	lower := r.SearchByName("andromeda")
	upper := r.SearchByName("ANDROMEDA")
	if len(lower) != len(upper) {
		t.Errorf("case-sensitive results differ: lower=%d upper=%d", len(lower), len(upper))
	}
}

func TestFilterByTypeGlobular(t *testing.T) {
	r := loadedRegistry(t)
	results := r.FilterByType(ObjectTypeGlobularClust)
	if len(results) < 10 {
		t.Errorf("FilterByType(globular): got %d, want >= 10", len(results))
	}
	for _, obj := range results {
		if obj.Type != ObjectTypeGlobularClust {
			t.Errorf("unexpected type %q for %s", obj.Type, obj.CatalogID)
		}
	}
}

func TestFilterByConstellation(t *testing.T) {
	r := loadedRegistry(t)
	results := r.FilterByConstellation("Sgr")
	if len(results) == 0 {
		t.Error("FilterByConstellation('Sgr'): no results")
	}
	for _, obj := range results {
		if obj.Constellation != "Sgr" {
			t.Errorf("unexpected constellation %q for %s", obj.Constellation, obj.CatalogID)
		}
	}
}

func TestFilterByMagnitude(t *testing.T) {
	r := loadedRegistry(t)
	results := r.FilterByMagnitude(6.0)
	if len(results) == 0 {
		t.Error("FilterByMagnitude(6.0): no results")
	}
	for _, obj := range results {
		if math.IsNaN(obj.Magnitude) {
			t.Errorf("%s has NaN magnitude in magnitude-filtered results", obj.CatalogID)
		}
		if obj.Magnitude > 6.0 {
			t.Errorf("%s magnitude %.1f exceeds filter cutoff 6.0", obj.CatalogID, obj.Magnitude)
		}
	}
}

func TestFilterMultipleCriteria(t *testing.T) {
	r := loadedRegistry(t)
	results := r.Filter(FilterCriteria{
		Types:         []ObjectType{ObjectTypeGalaxy},
		Constellation: "Vir",
		MaxMagnitude:  12.0,
	})
	for _, obj := range results {
		if obj.Type != ObjectTypeGalaxy {
			t.Errorf("unexpected type %q", obj.Type)
		}
		if obj.Constellation != "Vir" {
			t.Errorf("unexpected constellation %q", obj.Constellation)
		}
		if !math.IsNaN(obj.Magnitude) && obj.Magnitude > 12.0 {
			t.Errorf("magnitude %g exceeds cutoff", obj.Magnitude)
		}
	}
}

func TestObjectsInRegion(t *testing.T) {
	r := loadedRegistry(t)
	// Orion region: RA 4-7h, Dec -15 to +25.
	results := r.ObjectsInRegion(4.0, 7.0, -15.0, 25.0, 0)
	if len(results) == 0 {
		t.Error("ObjectsInRegion(Orion area): no results")
	}
	// M42 (Orion Nebula) RA~5.59h, Dec~-5.4° should be included.
	found := false
	for _, obj := range results {
		if obj.CatalogID == "M42" {
			found = true
		}
	}
	if !found {
		t.Error("M42 not found in Orion region query")
	}
}

func TestObjectsInRegionWraparound(t *testing.T) {
	r := loadedRegistry(t)
	// Region wrapping around 0h: 23h-1h RA. M31 is at ~0.71h, should be included.
	results := r.ObjectsInRegion(23.0, 1.0, 25.0, 55.0, 0)
	found := false
	for _, obj := range results {
		if obj.CatalogID == "M31" {
			found = true
		}
	}
	if !found {
		t.Error("M31 not found in RA wrap-around region query")
	}

	// Verify objects at RA=12h are NOT included.
	for _, obj := range results {
		if obj.RAJ2000 > 1.0 && obj.RAJ2000 < 23.0 {
			t.Errorf("object %s at RA=%.2f should not be in wrap-around region", obj.CatalogID, obj.RAJ2000)
		}
	}
}

func TestObjectsInRegionMagnitudeFilter(t *testing.T) {
	r := loadedRegistry(t)
	all := r.ObjectsInRegion(0, 24, -90, 90, 0)
	bright := r.ObjectsInRegion(0, 24, -90, 90, 5.0)
	if len(bright) >= len(all) {
		t.Error("magnitude filter should reduce result count")
	}
	for _, obj := range bright {
		if !math.IsNaN(obj.Magnitude) && obj.Magnitude > 5.0 {
			t.Errorf("%s mag %g exceeds filter 5.0", obj.CatalogID, obj.Magnitude)
		}
	}
}

func TestDynamicObjectProvider(t *testing.T) {
	r := NewCatalogRegistry()
	if err := r.LoadAll(); err != nil {
		t.Fatal(err)
	}

	provider := &mockDynamic{objects: []CelestialObject{
		{Name: "Jupiter", CatalogID: "Jupiter", Type: ObjectTypePlanet,
			RAJ2000: 5.0, DecJ2000: 20.0, CatalogSource: "solar_system"},
	}}
	r.RegisterDynamic(provider)

	obj, ok := r.GetByID("Jupiter")
	if !ok {
		t.Fatal("Jupiter not found via dynamic provider")
	}
	if obj.Type != ObjectTypePlanet {
		t.Errorf("Jupiter type: got %q", obj.Type)
	}

	planets := r.FilterByType(ObjectTypePlanet)
	if len(planets) < 1 {
		t.Error("FilterByType(planet): no results from dynamic provider")
	}
}

type mockDynamic struct {
	objects []CelestialObject
}

func (m *mockDynamic) Objects() []CelestialObject { return m.objects }
func (m *mockDynamic) Source() string             { return "solar_system" }

func TestConstellationName(t *testing.T) {
	tests := []struct{ code, want string }{
		{"And", "Andromeda"},
		{"Ori", "Orion"},
		{"UMa", "Ursa Major"},
		{"Sgr", "Sagittarius"},
		{"Sco", "Scorpius"},
		{"UNKNOWN", "UNKNOWN"}, // returns input unchanged
	}
	for _, tc := range tests {
		if got := ConstellationName(tc.code); got != tc.want {
			t.Errorf("ConstellationName(%q) = %q, want %q", tc.code, got, tc.want)
		}
	}
}

func TestConstellationNamesCount(t *testing.T) {
	if len(constellationNames) != 88 {
		t.Errorf("constellationNames has %d entries, want 88", len(constellationNames))
	}
}

func TestNamedStarsCount(t *testing.T) {
	if len(namedStarsCatalog) < 100 {
		t.Errorf("namedStarsCatalog: got %d entries, want >= 100", len(namedStarsCatalog))
	}
}

func TestNamedStarsSirius(t *testing.T) {
	var sirius *CelestialObject
	for i := range namedStarsCatalog {
		if namedStarsCatalog[i].Name == "Sirius" {
			sirius = &namedStarsCatalog[i]
			break
		}
	}
	if sirius == nil {
		t.Fatal("Sirius not in named stars catalog")
	}
	if math.Abs(sirius.RAJ2000-6.7525) > 0.001 {
		t.Errorf("Sirius RA: got %g, want ~6.7525", sirius.RAJ2000)
	}
	if math.Abs(sirius.DecJ2000-(-16.7161)) > 0.001 {
		t.Errorf("Sirius Dec: got %g, want ~-16.7161", sirius.DecJ2000)
	}
	if math.Abs(sirius.Magnitude-(-1.46)) > 0.01 {
		t.Errorf("Sirius mag: got %g, want -1.46", sirius.Magnitude)
	}
}

func TestMessierSpotCheck(t *testing.T) {
	// Spot-check 10 Messier objects against SEDS reference positions.
	tests := []struct {
		id         string
		raExpected float64 // decimal hours
		decExpected float64
		tol        float64 // degrees
	}{
		{"M1", 5.5755, 22.0145, 0.02},   // Crab Nebula
		{"M31", 0.7122, 41.2692, 0.02},  // Andromeda Galaxy
		{"M42", 5.5883, -5.3911, 0.02},  // Orion Nebula
		{"M45", 3.7914, 24.1167, 0.05},  // Pleiades
		{"M13", 16.6947, 36.4608, 0.02}, // Hercules Cluster
		{"M57", 18.8931, 33.0289, 0.02}, // Ring Nebula
		{"M87", 12.5136, 12.3911, 0.02}, // Virgo A
		{"M51", 13.4978, 47.1956, 0.02}, // Whirlpool Galaxy
		{"M8", 18.0617, -24.3833, 0.05}, // Lagoon Nebula
		{"M97", 11.2483, 55.0189, 0.02}, // Owl Nebula
	}
	for _, tc := range tests {
		obj, ok := r_global.GetByID(tc.id)
		if !ok {
			t.Errorf("%s not found", tc.id)
			continue
		}
		// RA tolerance in hours = angular tolerance / 15.
		raTolH := tc.tol / 15.0
		if math.Abs(obj.RAJ2000-tc.raExpected) > raTolH {
			t.Errorf("%s RA: got %.4f, want %.4f (tol %.4f h)", tc.id, obj.RAJ2000, tc.raExpected, raTolH)
		}
		if math.Abs(obj.DecJ2000-tc.decExpected) > tc.tol {
			t.Errorf("%s Dec: got %.4f, want %.4f (tol %.4f°)", tc.id, obj.DecJ2000, tc.decExpected, tc.tol)
		}
	}
}

// r_global is a package-level registry used by spot-check tests to avoid
// re-loading catalogs for each sub-test.
var r_global = func() *CatalogRegistry {
	r := NewCatalogRegistry()
	_ = r.LoadAll()
	return r
}()
