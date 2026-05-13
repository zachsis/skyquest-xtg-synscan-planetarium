package ephemeris

import (
	"time"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
)

// PlanetInfo extends CelestialObject with real-time observational data
// computed by the ephemeris engine.
type PlanetInfo struct {
	catalog.CelestialObject

	// AltDegrees is the current altitude above the horizon in degrees.
	AltDegrees float64
	// AzDegrees is the current azimuth in degrees (N=0, E=90).
	AzDegrees float64

	// RiseTime is the time the object rises above the horizon today.
	// Nil if circumpolar or never rises.
	RiseTime *time.Time
	// TransitTime is the time the object transits the meridian today.
	// Nil if unavailable.
	TransitTime *time.Time
	// SetTime is the time the object sets below the horizon today.
	// Nil if circumpolar or never sets.
	SetTime *time.Time

	// Elongation is the angular separation from the Sun in degrees.
	Elongation float64
	// Phase is the illumination fraction [0.0, 1.0].
	Phase float64
	// PhaseName is the Moon phase name (e.g. "Waxing Gibbous").
	// Empty for non-Moon objects.
	PhaseName string

	// Distance is the geocentric distance in AU (km for the Moon).
	Distance float64
	// AngularDiameter is the apparent angular diameter in arcseconds.
	AngularDiameter float64
}
