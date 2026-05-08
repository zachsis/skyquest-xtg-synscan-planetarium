package catalog

import "math"

// ObjectType categorizes celestial objects for filtering and rendering.
type ObjectType string

const (
	ObjectTypeStar          ObjectType = "star"
	ObjectTypeDoubleStar    ObjectType = "double_star"
	ObjectTypeGalaxy        ObjectType = "galaxy"
	ObjectTypeNebula        ObjectType = "nebula"
	ObjectTypePlanetaryNeb  ObjectType = "planetary_nebula"
	ObjectTypeOpenCluster   ObjectType = "open_cluster"
	ObjectTypeGlobularClust ObjectType = "globular_cluster"
	ObjectTypeSupernovaRem  ObjectType = "supernova_remnant"
	ObjectTypePlanet        ObjectType = "planet"
	ObjectTypeMoon          ObjectType = "moon"
	ObjectTypeSun           ObjectType = "sun"
)

// CelestialObject is the unified type for all catalog entries.
type CelestialObject struct {
	Name          string     // Primary display name
	CatalogID     string     // Canonical designation: "M31", "NGC 224", "HIP 32349"
	AlternateIDs  []string   // Other designations
	Type          ObjectType
	RAJ2000       float64    // Right ascension in decimal hours [0, 24)
	DecJ2000      float64    // Declination in decimal degrees [-90, +90]
	Magnitude     float64    // Visual magnitude (math.NaN() if unknown)
	AngularSize   float64    // Apparent size in arcminutes (0 if point source)
	Constellation string     // IAU 3-letter abbreviation
	Description   string     // Brief description
	CatalogSource string     // "messier", "ngc", "named_stars", "solar_system"
}

// HasMagnitude returns true if the object has a known magnitude.
func (o *CelestialObject) HasMagnitude() bool {
	return !math.IsNaN(o.Magnitude)
}

// DisplayName returns Name if set, otherwise CatalogID.
func (o *CelestialObject) DisplayName() string {
	if o.Name != "" {
		return o.Name
	}
	return o.CatalogID
}
