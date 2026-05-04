package astro

import "errors"

// Equatorial represents equatorial coordinates (J2000).
type Equatorial struct {
	RA  float64 // Right Ascension in hours [0, 24)
	Dec float64 // Declination in degrees [-90, +90]
}

// Horizontal represents horizontal (alt-azimuth) coordinates.
type Horizontal struct {
	Alt float64 // Altitude in degrees [-90, +90]
	Az  float64 // Azimuth in degrees [0, 360), 0=North, 90=East
}

// GeographicLocation represents an observer's position on Earth.
type GeographicLocation struct {
	Latitude  float64 // Degrees [-90, +90], positive = North
	Longitude float64 // Degrees [-180, +180], positive = East
	Elevation float64 // Meters above sea level
}

var (
	ErrNeverRises = errors.New("astro: object never rises at this location")
	ErrNeverSets  = errors.New("astro: object never sets at this location")
)
