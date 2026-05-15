package ui

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/astro"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/ephemeris"
)

// BrowserResult wraps a CelestialObject with computed transient fields
// (current Alt/Az, rise/set/transit times) so they are computed once per
// filter pass, not once per render.
type BrowserResult struct {
	Object      catalog.CelestialObject
	Alt         float64
	Az          float64
	RiseTime    *time.Time
	TransitTime *time.Time
	SetTime     *time.Time
	// Solar system extra fields (populated when CatalogSource == "solar_system").
	Elongation float64
	Phase      float64
	PhaseName  string
	Distance   float64
}

// FilterState holds the current filter and sort configuration for the browser.
type FilterState struct {
	Search        string
	Types         []catalog.ObjectType
	CatalogSource string
	MaxMagnitude  float64
	AboveHorizon  bool
	SortField     string // "name", "magnitude", "altitude", "type"
	SortAscending bool
}

// typeAbbreviation returns the short display abbreviation for an object type.
func typeAbbreviation(t catalog.ObjectType) string {
	switch t {
	case catalog.ObjectTypeGalaxy:
		return "Gal"
	case catalog.ObjectTypeNebula:
		return "Neb"
	case catalog.ObjectTypeOpenCluster:
		return "OC"
	case catalog.ObjectTypeGlobularClust:
		return "GC"
	case catalog.ObjectTypePlanetaryNeb:
		return "PNe"
	case catalog.ObjectTypeStar:
		return "Str"
	case catalog.ObjectTypeDoubleStar:
		return "Dbl"
	case catalog.ObjectTypePlanet:
		return "Plt"
	case catalog.ObjectTypeSun:
		return "Sun"
	case catalog.ObjectTypeMoon:
		return "Moo"
	case catalog.ObjectTypeSupernovaRem:
		return "SNR"
	default:
		return string(t)
	}
}

// typeFullName returns the human-readable full type name.
func typeFullName(t catalog.ObjectType) string {
	switch t {
	case catalog.ObjectTypeGalaxy:
		return "Galaxy"
	case catalog.ObjectTypeNebula:
		return "Nebula"
	case catalog.ObjectTypeOpenCluster:
		return "Open Cluster"
	case catalog.ObjectTypeGlobularClust:
		return "Globular Cluster"
	case catalog.ObjectTypePlanetaryNeb:
		return "Planetary Nebula"
	case catalog.ObjectTypeStar:
		return "Star"
	case catalog.ObjectTypeDoubleStar:
		return "Double Star"
	case catalog.ObjectTypePlanet:
		return "Planet"
	case catalog.ObjectTypeSun:
		return "Sun"
	case catalog.ObjectTypeMoon:
		return "Moon"
	case catalog.ObjectTypeSupernovaRem:
		return "Supernova Remnant"
	default:
		return string(t)
	}
}

// filterAndSort applies the given FilterState to the objects, computing
// positional data via astroSvc and looking up solar system extras from
// the ephemeris engine. It returns a sorted slice of BrowserResult.
func filterAndSort(
	objects []catalog.CelestialObject,
	fs FilterState,
	astroSvc *astro.AstroService,
	engine *ephemeris.EphemerisEngine,
) []BrowserResult {
	now := time.Now()
	lower := strings.ToLower(fs.Search)

	var results []BrowserResult
	for i := range objects {
		obj := &objects[i]

		// Text search filter.
		if lower != "" && !matchesSearchText(obj, lower) {
			continue
		}

		// Type filter.
		if len(fs.Types) > 0 && !containsType(fs.Types, obj.Type) {
			continue
		}

		// Catalog source filter.
		if fs.CatalogSource != "" && obj.CatalogSource != fs.CatalogSource {
			continue
		}

		// Magnitude filter.
		if fs.MaxMagnitude != 0 {
			if math.IsNaN(obj.Magnitude) || obj.Magnitude > fs.MaxMagnitude {
				continue
			}
		}

		// Compute alt/az.
		alt, az := astroSvc.AltAz(obj.RAJ2000, obj.DecJ2000, now)

		// Above-horizon filter.
		if fs.AboveHorizon && alt <= 0 {
			continue
		}

		// Build result.
		r := BrowserResult{
			Object: *obj,
			Alt:    alt,
			Az:     az,
		}

		// Populate rise/transit/set.
		rise, transit, set, err := astroSvc.RiseSetTimes(
			astro.Equatorial{RA: obj.RAJ2000, Dec: obj.DecJ2000}, now,
		)
		if err == nil {
			r.RiseTime = &rise
			r.TransitTime = &transit
			r.SetTime = &set
		}

		// Populate solar system extras from ephemeris engine.
		if engine != nil && obj.CatalogSource == "solar_system" {
			name := obj.Name
			if name == "" {
				name = obj.CatalogID
			}
			if info, ok := engine.GetPlanetInfo(name); ok {
				r.Elongation = info.Elongation
				r.Phase = info.Phase
				r.PhaseName = info.PhaseName
				r.Distance = info.Distance
			}
		}

		results = append(results, r)
	}

	sortResults(results, fs.SortField, fs.SortAscending)
	return results
}

func matchesSearchText(obj *catalog.CelestialObject, lower string) bool {
	if strings.Contains(strings.ToLower(obj.Name), lower) {
		return true
	}
	if strings.Contains(strings.ToLower(obj.CatalogID), lower) {
		return true
	}
	for _, alt := range obj.AlternateIDs {
		if strings.Contains(strings.ToLower(alt), lower) {
			return true
		}
	}
	if strings.Contains(strings.ToLower(obj.Constellation), lower) {
		return true
	}
	return false
}

func containsType(types []catalog.ObjectType, t catalog.ObjectType) bool {
	for _, tt := range types {
		if tt == t {
			return true
		}
	}
	return false
}

// sortResults sorts results in place. The natural/default direction for each
// field is: name A→Z, magnitude brightest first (low number), altitude
// highest first, type A→Z. When SortAscending is true it uses the natural
// direction; false reverses it.
func sortResults(results []BrowserResult, field string, ascending bool) {
	sort.SliceStable(results, func(i, j int) bool {
		var naturalLess bool
		switch field {
		case "magnitude":
			mi := results[i].Object.Magnitude
			mj := results[j].Object.Magnitude
			if math.IsNaN(mi) {
				mi = 99
			}
			if math.IsNaN(mj) {
				mj = 99
			}
			naturalLess = mi < mj // brightest (low number) first
		case "altitude":
			naturalLess = results[i].Alt > results[j].Alt // highest first
		case "type":
			ti := string(results[i].Object.Type)
			tj := string(results[j].Object.Type)
			if ti != tj {
				naturalLess = ti < tj
			} else {
				naturalLess = displayName(&results[i].Object) < displayName(&results[j].Object)
			}
		default: // "name"
			naturalLess = displayName(&results[i].Object) < displayName(&results[j].Object)
		}
		if ascending {
			return naturalLess
		}
		return !naturalLess
	})
}

func displayName(obj *catalog.CelestialObject) string {
	if obj.Name != "" {
		return obj.Name
	}
	return obj.CatalogID
}
