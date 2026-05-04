package astro

import (
	"math"
	"time"
)

const standardAltitude = -0.5667 // degrees, accounting for refraction

// RiseSetTimes calculates the rise, transit, and set times for an object
// at the given location on the given date (UTC). Returns ErrNeverRises
// for objects that are always below the horizon and ErrNeverSets for
// circumpolar objects.
func RiseSetTimes(eq Equatorial, loc GeographicLocation, date time.Time) (rise, transit, set time.Time, err error) {
	// Algorithm from Meeus, Chapter 15.
	lat := loc.Latitude * deg2rad
	dec := eq.Dec * deg2rad
	h0 := standardAltitude * deg2rad

	// Compute cos(H0) where H0 is the hour angle at rise/set.
	cosH0 := (math.Sin(h0) - math.Sin(lat)*math.Sin(dec)) / (math.Cos(lat) * math.Cos(dec))

	if cosH0 > 1 {
		return time.Time{}, time.Time{}, time.Time{}, ErrNeverRises
	}
	if cosH0 < -1 {
		return time.Time{}, time.Time{}, time.Time{}, ErrNeverSets
	}

	H0 := math.Acos(cosH0) * rad2deg // in degrees

	// Compute transit time.
	// Use the date's midnight UTC as the base.
	y, m, d := date.UTC().Date()
	midnight := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)

	lst0 := LocalSiderealTime(loc.Longitude, midnight) // LST at midnight
	raHours := eq.RA

	// Transit: when RA = LST, so transit_LST = RA
	// Time of transit in hours after midnight.
	transitHours := raHours - lst0
	if transitHours < 0 {
		transitHours += 24
	}
	if transitHours >= 24 {
		transitHours -= 24
	}

	// H0 in hours.
	H0hours := H0 / 15.0

	riseHours := transitHours - H0hours
	setHours := transitHours + H0hours

	// Normalize to [0, 24).
	for riseHours < 0 {
		riseHours += 24
	}
	for riseHours >= 24 {
		riseHours -= 24
	}
	for setHours < 0 {
		setHours += 24
	}
	for setHours >= 24 {
		setHours -= 24
	}

	rise = midnight.Add(time.Duration(riseHours * float64(time.Hour)))
	transit = midnight.Add(time.Duration(transitHours * float64(time.Hour)))
	set = midnight.Add(time.Duration(setHours * float64(time.Hour)))

	return rise, transit, set, nil
}
