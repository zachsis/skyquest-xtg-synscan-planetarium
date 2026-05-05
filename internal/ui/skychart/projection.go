package skychart

import "math"

// StereoRadius returns the projected radial distance for a given altitude
// and chart radius under the stereographic projection.
func StereoRadius(alt, radius float64) float64 {
	return radius * math.Cos(alt) / (1.0 + math.Sin(alt))
}

// StereoProject maps altitude/azimuth (radians) to screen X/Y.
// The zenith (alt=π/2) maps to the center (0,0); the horizon (alt=0) maps
// to the circle edge at the given radius. East is to the left when looking
// up (standard sky chart convention).
func StereoProject(alt, az, radius float64) (x, y float64) {
	r := StereoRadius(alt, radius)
	x = -r * math.Sin(az)
	y = -r * math.Cos(az)
	return x, y
}

// StereoUnproject maps screen X/Y back to altitude/azimuth (radians).
// Returns alt in [0, π/2] and az in [0, 2π).
func StereoUnproject(x, y, radius float64) (alt, az float64) {
	r := math.Sqrt(x*x + y*y)
	rNorm := r / radius
	alt = math.Asin((1.0 - rNorm*rNorm) / (1.0 + rNorm*rNorm))
	az = math.Atan2(-x, -y)
	if az < 0 {
		az += 2 * math.Pi
	}
	return alt, az
}
