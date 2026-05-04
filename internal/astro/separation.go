package astro

import "math"

// AngularSeparation returns the angular separation in degrees between
// two equatorial positions using the Vincenty formula.
func AngularSeparation(a, b Equatorial) float64 {
	ra1, dec1 := a.RA*15*deg2rad, a.Dec*deg2rad
	ra2, dec2 := b.RA*15*deg2rad, b.Dec*deg2rad
	dRA := ra2 - ra1

	num := math.Sqrt(
		math.Pow(math.Cos(dec2)*math.Sin(dRA), 2) +
			math.Pow(math.Cos(dec1)*math.Sin(dec2)-math.Sin(dec1)*math.Cos(dec2)*math.Cos(dRA), 2))
	den := math.Sin(dec1)*math.Sin(dec2) + math.Cos(dec1)*math.Cos(dec2)*math.Cos(dRA)

	return math.Atan2(num, den) * rad2deg
}
