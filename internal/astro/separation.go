package astro

import "math"

// AngularSeparation returns the angular separation in degrees between
// two equatorial positions using the Vincenty formula.
func AngularSeparation(a, b Equatorial) float64 {
	ra1, dec1 := a.RA*15*math.Pi/180, a.Dec*math.Pi/180
	ra2, dec2 := b.RA*15*math.Pi/180, b.Dec*math.Pi/180
	dRA := ra2 - ra1

	num := math.Sqrt(
		math.Pow(math.Cos(dec2)*math.Sin(dRA), 2) +
			math.Pow(math.Cos(dec1)*math.Sin(dec2)-math.Sin(dec1)*math.Cos(dec2)*math.Cos(dRA), 2))
	den := math.Sin(dec1)*math.Sin(dec2) + math.Cos(dec1)*math.Cos(dec2)*math.Cos(dRA)

	return math.Atan2(num, den) * 180 / math.Pi
}
