package ephemeris

import (
	"math"
	"time"

	"github.com/soniakeys/meeus/v3/base"
	"github.com/soniakeys/meeus/v3/coord"
	"github.com/soniakeys/meeus/v3/julian"
	"github.com/soniakeys/meeus/v3/moonillum"
	"github.com/soniakeys/meeus/v3/moonposition"
	"github.com/soniakeys/meeus/v3/nutation"
	"github.com/soniakeys/unit"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/astro"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
)

// computeMoon calculates the apparent position of the Moon at time t.
// sunInfo must already be computed for the same epoch (needed for phase).
func computeMoon(
	astroSvc *astro.AstroService,
	t time.Time,
	sun PlanetInfo,
) PlanetInfo {
	jde := julian.TimeToJD(t)

	// Geocentric ecliptic longitude, latitude, and distance (km).
	// Returned λ, β are referenced to mean equinox of date without nutation.
	λ, β, Δ := moonposition.Position(jde)

	// Apply nutation to longitude.
	Δψ, Δε := nutation.Nutation(jde)
	apparentλ := λ + Δψ

	// True obliquity of the ecliptic.
	ε := nutation.MeanObliquity(jde) + Δε
	sε, cε := ε.Sincos()

	// Convert to equatorial.
	α, δ := coord.EclToEq(apparentλ, β, sε, cε)
	raHours := α.Hour()
	decDeg := δ.Deg()

	// Horizontal coordinates.
	alt, az := astroSvc.AltAz(raHours, decDeg, t)

	// Rise / transit / set.
	eq := astro.Equatorial{RA: raHours, Dec: decDeg}
	rise, transit, set, err := astroSvc.RiseSetTimes(eq, t)
	var risePtr, transitPtr, setPtr *time.Time
	if err == nil {
		risePtr = &rise
		transitPtr = &transit
		setPtr = &set
	}

	// Phase angle using equatorial coordinates.
	// Sun distance in km: R (AU) * km_per_AU.
	sunDistKM := sun.Distance * 149597870.7
	sunRA := unit.RA(sun.RAJ2000 * math.Pi / 12)  // hours → radians
	sunDec := unit.Angle(sun.DecJ2000 * math.Pi / 180) // deg → radians
	phaseAngle := moonillum.PhaseAngleEq(α, δ, Δ, sunRA, sunDec, sunDistKM)
	illumination := base.Illuminated(phaseAngle)

	// Angular diameter from distance: 2 * atan(R_moon_km / Δ) in arcsec.
	// Lunar radius = 1737.4 km.
	angDiam := 2 * math.Atan(1737.4/Δ) * 206265 // arcsec

	// Moon phase name: use ecliptic longitude difference to determine
	// waxing vs waning.
	sunλ := sunEclipticLon(jde)
	moonSunDiff := λ.Rad() - sunλ
	// Normalize to [0, 2π).
	moonSunDiff = math.Mod(moonSunDiff, 2*math.Pi)
	if moonSunDiff < 0 {
		moonSunDiff += 2 * math.Pi
	}
	phaseName := moonPhaseName(phaseAngle.Rad(), moonSunDiff)

	obj := catalog.CelestialObject{
		Name:          "Moon",
		CatalogID:     "Moon",
		Type:          catalog.ObjectTypeMoon,
		RAJ2000:       raHours,
		DecJ2000:      decDeg,
		Magnitude:     defaultMagnitudes["Moon"],
		AngularSize:   angDiam / 60,
		CatalogSource: "solar_system",
	}

	return PlanetInfo{
		CelestialObject: obj,
		AltDegrees:      alt,
		AzDegrees:       az,
		RiseTime:        risePtr,
		TransitTime:     transitPtr,
		SetTime:         setPtr,
		Phase:           illumination,
		PhaseName:       phaseName,
		Distance:        Δ,
		AngularDiameter: angDiam,
	}
}

// sunEclipticLon returns an approximate geocentric ecliptic longitude of the
// Sun in radians, derived from the Moon's ecliptic frame. Used only for the
// waxing/waning determination so low accuracy is acceptable.
func sunEclipticLon(jde float64) float64 {
	// Use the simple low-accuracy formula from the solar package.
	T := base.J2000Century(jde)
	L0 := (280.46646 + 36000.76983*T) * math.Pi / 180
	M := (357.52911 + 35999.05029*T - 0.0001537*T*T) * math.Pi / 180
	C := (1.914602-0.004817*T-0.000014*T*T)*math.Sin(M) +
		(0.019993-0.000101*T)*math.Sin(2*M) +
		0.000289*math.Sin(3*M)
	sun := L0 + C*math.Pi/180
	return sun
}

// moonPhaseName maps phase angle (radians) and the Moon-Sun longitude
// difference (radians, in [0, 2π)) to a named lunar phase.
func moonPhaseName(phaseAngleRad, moonSunDiffRad float64) string {
	// moonSunDiff < π means waxing (Moon is east of Sun), ≥ π means waning.
	waxing := moonSunDiffRad < math.Pi

	switch {
	case phaseAngleRad < 0.18: // ~10°
		return "New Moon"
	case phaseAngleRad < 1.22 && waxing: // ~70°
		return "Waxing Crescent"
	case phaseAngleRad < 1.22 && !waxing:
		return "Waning Crescent"
	case phaseAngleRad < 1.75 && waxing: // ~100°
		return "First Quarter"
	case phaseAngleRad < 1.75 && !waxing:
		return "Last Quarter"
	case phaseAngleRad < 2.79 && waxing: // ~160°
		return "Waxing Gibbous"
	case phaseAngleRad < 2.79 && !waxing:
		return "Waning Gibbous"
	default:
		return "Full Moon"
	}
}
