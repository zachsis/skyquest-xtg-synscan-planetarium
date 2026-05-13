package ephemeris

import (
	"math"
	"time"

	"github.com/soniakeys/meeus/v3/julian"
	pp "github.com/soniakeys/meeus/v3/planetposition"
	"github.com/soniakeys/meeus/v3/solar"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/astro"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
)

// computeSun calculates the apparent position of the Sun at time t.
// earth is the VSOP87B Earth V87Planet object.
// Returns a PlanetInfo with RA/Dec, alt/az, rise/set, and angular diameter.
func computeSun(
	earth *pp.V87Planet,
	astroSvc *astro.AstroService,
	t time.Time,
) PlanetInfo {
	jde := julian.TimeToJD(t)

	// Apparent equatorial coordinates via full VSOP87.
	// α is in hours, δ in degrees, R in AU.
	α, δ, R := solar.ApparentEquatorialVSOP87(earth, jde)
	raHours := α.Hour()
	decDeg := δ.Deg()

	// Horizontal coordinates.
	alt, az := astroSvc.AltAz(raHours, decDeg, t)

	// Rise / transit / set times.
	eq := astro.Equatorial{RA: raHours, Dec: decDeg}
	rise, transit, set, err := astroSvc.RiseSetTimes(eq, t)
	var risePtr, transitPtr, setPtr *time.Time
	if err == nil {
		risePtr = &rise
		transitPtr = &transit
		setPtr = &set
	}

	// Angular diameter from distance R (AU): 2 * atan(R_sun_km / dist_km).
	// Solar radius = 696000 km; 1 AU = 149597870.7 km.
	distKM := R * 149597870.7
	angDiam := 2 * math.Atan(696000/distKM) * 180 / math.Pi * 3600 // arcsec

	obj := catalog.CelestialObject{
		Name:          "Sun",
		CatalogID:     "Sun",
		Type:          catalog.ObjectTypeSun,
		RAJ2000:       raHours,
		DecJ2000:      decDeg,
		Magnitude:     defaultMagnitudes["Sun"],
		AngularSize:   angDiam / 60, // store in arcmin
		CatalogSource: "solar_system",
	}

	return PlanetInfo{
		CelestialObject: obj,
		AltDegrees:      alt,
		AzDegrees:       az,
		RiseTime:        risePtr,
		TransitTime:     transitPtr,
		SetTime:         setPtr,
		Phase:           1.0,
		Distance:        R,
		AngularDiameter: angDiam,
	}
}
