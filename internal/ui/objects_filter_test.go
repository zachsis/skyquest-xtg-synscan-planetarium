package ui

import (
	"math"
	"testing"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
)

func makeObj(name, catalogID, source string, t catalog.ObjectType, mag float64, const_ string) catalog.CelestialObject {
	return catalog.CelestialObject{
		Name:          name,
		CatalogID:     catalogID,
		Type:          t,
		Magnitude:     mag,
		Constellation: const_,
		CatalogSource: source,
	}
}

func TestMatchesSearchText(t *testing.T) {
	obj := catalog.CelestialObject{
		Name:          "Andromeda Galaxy",
		CatalogID:     "M31",
		AlternateIDs:  []string{"NGC 224"},
		Constellation: "And",
	}

	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{"match name", "andromeda", true},
		{"match catalogID", "m31", true},
		{"match alternateID", "ngc 224", true},
		{"match constellation", "and", true},
		{"no match", "orion", false},
		{"case insensitive name", "andromeda", true},
		{"partial match", "galax", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesSearchText(&obj, tt.query)
			if got != tt.want {
				t.Errorf("matchesSearchText(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestContainsType(t *testing.T) {
	types := []catalog.ObjectType{
		catalog.ObjectTypeGalaxy,
		catalog.ObjectTypeNebula,
	}
	tests := []struct {
		t    catalog.ObjectType
		want bool
	}{
		{catalog.ObjectTypeGalaxy, true},
		{catalog.ObjectTypeNebula, true},
		{catalog.ObjectTypeStar, false},
		{catalog.ObjectTypePlanet, false},
	}
	for _, tt := range tests {
		got := containsType(types, tt.t)
		if got != tt.want {
			t.Errorf("containsType(%v) = %v, want %v", tt.t, got, tt.want)
		}
	}
}

func TestTypeAbbreviation(t *testing.T) {
	tests := []struct {
		t    catalog.ObjectType
		want string
	}{
		{catalog.ObjectTypeGalaxy, "Gal"},
		{catalog.ObjectTypeNebula, "Neb"},
		{catalog.ObjectTypeOpenCluster, "OC"},
		{catalog.ObjectTypeGlobularClust, "GC"},
		{catalog.ObjectTypePlanetaryNeb, "PNe"},
		{catalog.ObjectTypeStar, "Str"},
		{catalog.ObjectTypeDoubleStar, "Dbl"},
		{catalog.ObjectTypePlanet, "Plt"},
		{catalog.ObjectTypeSun, "Sun"},
		{catalog.ObjectTypeMoon, "Moo"},
		{catalog.ObjectTypeSupernovaRem, "SNR"},
	}
	for _, tt := range tests {
		got := typeAbbreviation(tt.t)
		if got != tt.want {
			t.Errorf("typeAbbreviation(%v) = %q, want %q", tt.t, got, tt.want)
		}
	}
}

func TestTypeFullName(t *testing.T) {
	tests := []struct {
		t    catalog.ObjectType
		want string
	}{
		{catalog.ObjectTypeGalaxy, "Galaxy"},
		{catalog.ObjectTypeNebula, "Nebula"},
		{catalog.ObjectTypeOpenCluster, "Open Cluster"},
		{catalog.ObjectTypeGlobularClust, "Globular Cluster"},
		{catalog.ObjectTypePlanetaryNeb, "Planetary Nebula"},
		{catalog.ObjectTypeStar, "Star"},
		{catalog.ObjectTypeDoubleStar, "Double Star"},
		{catalog.ObjectTypePlanet, "Planet"},
		{catalog.ObjectTypeSun, "Sun"},
		{catalog.ObjectTypeMoon, "Moon"},
		{catalog.ObjectTypeSupernovaRem, "Supernova Remnant"},
	}
	for _, tt := range tests {
		got := typeFullName(tt.t)
		if got != tt.want {
			t.Errorf("typeFullName(%v) = %q, want %q", tt.t, got, tt.want)
		}
	}
}

func TestSortResults_ByName(t *testing.T) {
	results := []BrowserResult{
		{Object: makeObj("Zebra Nebula", "Z1", "", catalog.ObjectTypeNebula, 5, "")},
		{Object: makeObj("Andromeda Galaxy", "M31", "", catalog.ObjectTypeGalaxy, 3.4, "")},
		{Object: makeObj("Milky Way", "MW", "", catalog.ObjectTypeGalaxy, 1, "")},
	}

	sortResults(results, "name", true)
	if results[0].Object.Name != "Andromeda Galaxy" {
		t.Errorf("expected first to be Andromeda Galaxy, got %s", results[0].Object.Name)
	}
	if results[2].Object.Name != "Zebra Nebula" {
		t.Errorf("expected last to be Zebra Nebula, got %s", results[2].Object.Name)
	}

	// Descending reverses.
	sortResults(results, "name", false)
	if results[0].Object.Name != "Zebra Nebula" {
		t.Errorf("desc: expected first to be Zebra Nebula, got %s", results[0].Object.Name)
	}
}

func TestSortResults_ByMagnitude(t *testing.T) {
	results := []BrowserResult{
		{Object: makeObj("A", "A", "", catalog.ObjectTypeStar, 5.0, ""), Alt: 0},
		{Object: makeObj("B", "B", "", catalog.ObjectTypeStar, 1.0, ""), Alt: 0},
		{Object: makeObj("C", "C", "", catalog.ObjectTypeStar, math.NaN(), ""), Alt: 0},
	}

	sortResults(results, "magnitude", true)
	if results[0].Object.Name != "B" {
		t.Errorf("expected brightest (1.0) first, got %s", results[0].Object.Name)
	}
	if results[2].Object.Name != "C" {
		t.Errorf("expected NaN last, got %s", results[2].Object.Name)
	}
}

func TestSortResults_ByAltitude(t *testing.T) {
	results := []BrowserResult{
		{Object: makeObj("Low", "L", "", catalog.ObjectTypeStar, 5, ""), Alt: 10.0},
		{Object: makeObj("High", "H", "", catalog.ObjectTypeStar, 5, ""), Alt: 75.0},
		{Object: makeObj("Mid", "M", "", catalog.ObjectTypeStar, 5, ""), Alt: 40.0},
	}

	sortResults(results, "altitude", true)
	if results[0].Object.Name != "High" {
		t.Errorf("asc altitude: expected High first, got %s", results[0].Object.Name)
	}

	sortResults(results, "altitude", false)
	if results[0].Object.Name != "Low" {
		t.Errorf("desc altitude: expected Low first, got %s", results[0].Object.Name)
	}
}

func TestDisplayName(t *testing.T) {
	tests := []struct {
		name      string
		catalogID string
		want      string
	}{
		{"Andromeda Galaxy", "M31", "Andromeda Galaxy"},
		{"", "M31", "M31"},
	}
	for _, tt := range tests {
		obj := catalog.CelestialObject{Name: tt.name, CatalogID: tt.catalogID}
		got := displayName(&obj)
		if got != tt.want {
			t.Errorf("displayName(%q, %q) = %q, want %q", tt.name, tt.catalogID, got, tt.want)
		}
	}
}
