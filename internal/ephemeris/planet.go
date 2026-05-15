package ephemeris

import (
	"math"
	"time"

	"github.com/soniakeys/meeus/v3/coord"
	"github.com/soniakeys/meeus/v3/julian"
	"github.com/soniakeys/meeus/v3/nutation"
	pp "github.com/soniakeys/meeus/v3/planetposition"
	"github.com/soniakeys/unit"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/astro"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
)

// planetMeta holds static metadata for each of the 8 planets.
type planetMeta struct {
	// name is the display name.
	name string
	// ibody is the planetposition constant (Mercury=0 … Neptune=7).
	ibody int
	// objType is the catalog ObjectType.
	objType catalog.ObjectType
	// meanRadiusKM is the equatorial radius used for angular diameter.
	meanRadiusKM float64
}

// planetsMeta lists static metadata for the 7 observable planets (all except Earth).
// Order matches the engine's planets array index.
var planetsMeta = [7]planetMeta{
	{name: "Mercury", ibody: pp.Mercury, objType: catalog.ObjectTypePlanet, meanRadiusKM: 2439.7},
	{name: "Venus", ibody: pp.Venus, objType: catalog.ObjectTypePlanet, meanRadiusKM: 6051.8},
	{name: "Mars", ibody: pp.Mars, objType: catalog.ObjectTypePlanet, meanRadiusKM: 3396.2},
	{name: "Jupiter", ibody: pp.Jupiter, objType: catalog.ObjectTypePlanet, meanRadiusKM: 71492},
	{name: "Saturn", ibody: pp.Saturn, objType: catalog.ObjectTypePlanet, meanRadiusKM: 60268},
	{name: "Uranus", ibody: pp.Uranus, objType: catalog.ObjectTypePlanet, meanRadiusKM: 25559},
	{name: "Neptune", ibody: pp.Neptune, objType: catalog.ObjectTypePlanet, meanRadiusKM: 24764},
}

// computePlanet calculates the apparent geocentric position of a planet at t.
// planet is the loaded V87Planet object; earth is Earth's V87Planet.
// meta contains the planet's static metadata.
// sun is the already-computed Sun PlanetInfo (needed for elongation).
func computePlanet(
	planet *pp.V87Planet,
	earth *pp.V87Planet,
	meta planetMeta,
	astroSvc *astro.AstroService,
	t time.Time,
	sun PlanetInfo,
) PlanetInfo {
	jde := julian.TimeToJD(t)

	// Earth heliocentric ecliptic of date.
	L0, B0, R0 := earth.Position(jde)
	sL0, cL0 := math.Sincos(L0.Rad())
	sB0, cB0 := math.Sincos(B0.Rad())
	x0 := R0 * cB0 * cL0
	y0 := R0 * cB0 * sL0
	z0 := R0 * sB0

	// Planet heliocentric ecliptic of date — initial pass.
	L, B, R := planet.Position(jde)
	x, y, z, Δ := planetGeoRect(L, B, R, x0, y0, z0)

	// Light-time correction: τ = Δ * 0.0057755183 days.
	τ := Δ * 0.0057755183
	// Recompute planet at jde - τ.
	L, B, R = planet.Position(jde - τ)
	x, y, z, Δ = planetGeoRect(L, B, R, x0, y0, z0)

	// Geocentric ecliptic longitude and latitude.
	λ := unit.Angle(math.Atan2(y, x))
	β := unit.Angle(math.Atan(z / math.Sqrt(x*x+y*y)))

	// Nutation.
	Δψ, Δε := nutation.Nutation(jde)
	λ += Δψ

	// True obliquity.
	ε := nutation.MeanObliquity(jde) + Δε
	sε, cε := ε.Sincos()

	// Convert to equatorial.
	α, δ := coord.EclToEq(λ, β, sε, cε)
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

	// Elongation from the Sun.
	elong := elongation(raHours, decDeg, sun.RAJ2000, sun.DecJ2000)

	// Angular diameter from geocentric distance Δ (AU).
	// angDiam (arcsec) = 2 * atan(radius_km / dist_km) * 206265
	distKM := Δ * 149597870.7
	angDiam := 2 * math.Atan(meta.meanRadiusKM/distKM) * 206265

	// Approximate illumination fraction for planets (phase angle via elongation).
	// For inferior planets (inside Earth's orbit) the phase can differ significantly.
	phase := 0.5 * (1 + math.Cos(elong*math.Pi/180))

	obj := catalog.CelestialObject{
		Name:          meta.name,
		CatalogID:     meta.name,
		Type:          meta.objType,
		RAJ2000:       raHours,
		DecJ2000:      decDeg,
		Magnitude:     defaultMagnitudes[meta.name],
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
		Elongation:      elong,
		Phase:           phase,
		Distance:        Δ,
		AngularDiameter: angDiam,
	}
}

// planetGeoRect converts heliocentric ecliptic coords to geocentric rectangular
// ecliptic coords (in AU) and returns the distance Δ.
func planetGeoRect(L, B unit.Angle, R, x0, y0, z0 float64) (x, y, z, Δ float64) {
	sL, cL := math.Sincos(L.Rad())
	sB, cB := math.Sincos(B.Rad())
	x = R*cB*cL - x0
	y = R*cB*sL - y0
	z = R*sB - z0
	Δ = math.Sqrt(x*x + y*y + z*z)
	return
}

// elongation returns the angular separation in degrees between two equatorial
// positions. ra values are in decimal hours, dec values in degrees.
func elongation(ra1, dec1, ra2, dec2 float64) float64 {
	// Convert RA hours → radians.
	r1 := ra1 * math.Pi / 12
	r2 := ra2 * math.Pi / 12
	d1 := dec1 * math.Pi / 180
	d2 := dec2 * math.Pi / 180
	cosAngle := math.Sin(d1)*math.Sin(d2) + math.Cos(d1)*math.Cos(d2)*math.Cos(r1-r2)
	return math.Acos(clamp(cosAngle, -1, 1)) * 180 / math.Pi
}

// clamp restricts v to the range [lo, hi].
func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// planetBodyIndex maps the 7 non-Earth planet metadata indices (0-6) to the
// corresponding planetposition constants.
var planetBodyIndex = [7]int{
	pp.Mercury, pp.Venus, pp.Mars,
	pp.Jupiter, pp.Saturn, pp.Uranus, pp.Neptune,
}
