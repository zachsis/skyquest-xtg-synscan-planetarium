package astro

import (
	"math"
	"time"
)

const (
	deg2rad = math.Pi / 180
	rad2deg = 180 / math.Pi
)

// EquatorialToHorizontal converts equatorial coordinates to horizontal
// for the given observer location and UTC time.
// Uses the standard spherical trigonometry formulas (Meeus Chapter 13).
func EquatorialToHorizontal(eq Equatorial, loc GeographicLocation, t time.Time) Horizontal {
	lst := LocalSiderealTime(loc.Longitude, t)

	// Hour angle in hours, then convert to radians.
	ha := (lst - eq.RA) * 15 * deg2rad // hours -> degrees -> radians
	dec := eq.Dec * deg2rad
	lat := loc.Latitude * deg2rad

	// Altitude.
	sinAlt := math.Sin(dec)*math.Sin(lat) + math.Cos(dec)*math.Cos(lat)*math.Cos(ha)
	alt := math.Asin(sinAlt)

	// Azimuth.
	cosAz := (math.Sin(dec) - math.Sin(lat)*sinAlt) / (math.Cos(lat) * math.Cos(alt))
	// Clamp for numerical stability.
	if cosAz > 1 {
		cosAz = 1
	}
	if cosAz < -1 {
		cosAz = -1
	}
	az := math.Acos(cosAz)
	if math.Sin(ha) > 0 {
		az = 2*math.Pi - az
	}

	altDeg := alt * rad2deg
	azDeg := az * rad2deg
	if azDeg < 0 {
		azDeg += 360
	}

	return Horizontal{
		Alt: altDeg,
		Az:  azDeg,
	}
}

// HorizontalToEquatorial converts horizontal coordinates to equatorial
// for the given observer location and UTC time.
func HorizontalToEquatorial(hz Horizontal, loc GeographicLocation, t time.Time) Equatorial {
	lst := LocalSiderealTime(loc.Longitude, t)

	alt := hz.Alt * deg2rad
	az := hz.Az * deg2rad
	lat := loc.Latitude * deg2rad

	// Declination.
	sinDec := math.Sin(alt)*math.Sin(lat) + math.Cos(alt)*math.Cos(lat)*math.Cos(az)
	dec := math.Asin(sinDec)

	// Hour angle.
	cosHA := (math.Sin(alt) - math.Sin(lat)*sinDec) / (math.Cos(lat) * math.Cos(dec))
	if cosHA > 1 {
		cosHA = 1
	}
	if cosHA < -1 {
		cosHA = -1
	}
	ha := math.Acos(cosHA)
	if math.Sin(az) > 0 {
		ha = 2*math.Pi - ha
	}

	// RA = LST - HA.
	raHours := lst - ha*rad2deg/15
	for raHours < 0 {
		raHours += 24
	}
	for raHours >= 24 {
		raHours -= 24
	}

	return Equatorial{
		RA:  raHours,
		Dec: dec * rad2deg,
	}
}
