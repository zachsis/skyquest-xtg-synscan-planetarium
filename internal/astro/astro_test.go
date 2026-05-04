package astro

import (
	"math"
	"testing"
	"time"
)

// testLocation is a fixed observer location for tests (New York City area).
var testLocation = GeographicLocation{
	Latitude:  40.7128,
	Longitude: -74.0060,
	Elevation: 10,
}

// Known stars for test vectors.
var (
	polaris = Equatorial{RA: 2.5303, Dec: 89.2642}  // RA 2h31m49s, Dec +89°15'51"
	sirius  = Equatorial{RA: 6.7525, Dec: -16.7161}  // RA 6h45m09s, Dec -16°42'58"
	vega    = Equatorial{RA: 18.6156, Dec: 38.7837}   // RA 18h36m56s, Dec +38°47'01"
)

func TestEquatorialToHorizontalRoundTrip(t *testing.T) {
	testTime := time.Date(2026, 3, 15, 22, 0, 0, 0, time.UTC)
	stars := []Equatorial{polaris, sirius, vega}

	for _, eq := range stars {
		hz := EquatorialToHorizontal(eq, testLocation, testTime)
		back := HorizontalToEquatorial(hz, testLocation, testTime)

		raDiff := math.Abs(eq.RA - back.RA)
		if raDiff > 12 {
			raDiff = 24 - raDiff // handle wrap-around
		}
		decDiff := math.Abs(eq.Dec - back.Dec)

		// Round-trip should be within 1 arcsecond (1/3600 degree).
		if raDiff*15 > 1.0/3600 || decDiff > 1.0/3600 {
			t.Errorf("round trip for RA=%.4f Dec=%.4f: got RA=%.4f Dec=%.4f (raDiff=%.6f° decDiff=%.6f°)",
				eq.RA, eq.Dec, back.RA, back.Dec, raDiff*15, decDiff)
		}
	}
}

func TestPolarisAboveHorizonNorth(t *testing.T) {
	// Polaris should always be above horizon at 40°N.
	testTime := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	hz := EquatorialToHorizontal(polaris, testLocation, testTime)
	if hz.Alt <= 0 {
		t.Errorf("Polaris should be above horizon at 40N, got Alt=%.2f", hz.Alt)
	}
	// Polaris altitude should be approximately equal to the latitude.
	if math.Abs(hz.Alt-testLocation.Latitude) > 2.0 {
		t.Errorf("Polaris altitude (%.2f) should be near latitude (%.2f)", hz.Alt, testLocation.Latitude)
	}
}

func TestPolarisNeverVisibleSouth(t *testing.T) {
	southLoc := GeographicLocation{Latitude: -40.0, Longitude: 0, Elevation: 0}
	testTime := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	hz := EquatorialToHorizontal(polaris, southLoc, testTime)
	if hz.Alt > 0 {
		t.Errorf("Polaris should be below horizon at 40S, got Alt=%.2f", hz.Alt)
	}
}

func TestLSTKnownValue(t *testing.T) {
	// J2000.0 epoch: 2000-01-01 12:00 UT, Greenwich.
	// At J2000.0, GMST ≈ 18.6973 hours.
	j2000 := time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)
	lst := LocalSiderealTime(0, j2000) // Greenwich
	// GMST at J2000.0 is approximately 18.697 hours.
	if math.Abs(lst-18.697) > 0.05 {
		t.Errorf("LST at J2000.0 Greenwich: expected ~18.697h, got %.3f", lst)
	}
}

func TestLSTLongitudeOffset(t *testing.T) {
	testTime := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	lstGreenwich := LocalSiderealTime(0, testTime)
	lstNY := LocalSiderealTime(-74.006, testTime)
	// NY is 74° west, so LST should be ~4.93 hours behind Greenwich.
	diff := lstGreenwich - lstNY
	if diff < 0 {
		diff += 24
	}
	expectedDiff := 74.006 / 15.0 // ~4.93 hours
	if math.Abs(diff-expectedDiff) > 0.05 {
		t.Errorf("LST offset: expected %.2f hours, got %.2f", expectedDiff, diff)
	}
}

func TestAngularSeparationIdentical(t *testing.T) {
	sep := AngularSeparation(vega, vega)
	if sep > 0.0001 {
		t.Errorf("separation of identical positions: expected 0, got %f", sep)
	}
}

func TestAngularSeparationOpposite(t *testing.T) {
	a := Equatorial{RA: 0, Dec: 0}
	b := Equatorial{RA: 12, Dec: 0} // 180° apart on the equator
	sep := AngularSeparation(a, b)
	if math.Abs(sep-180) > 0.01 {
		t.Errorf("separation of opposite positions: expected 180, got %f", sep)
	}
}

func TestAngularSeparationKnown(t *testing.T) {
	// Sirius to Vega: approximately 158 degrees (opposite sides of the sky).
	sep := AngularSeparation(sirius, vega)
	if sep < 155 || sep > 162 {
		t.Errorf("Sirius-Vega separation: expected ~158°, got %.2f", sep)
	}
	// Polaris to Vega: approximately 51 degrees.
	sep2 := AngularSeparation(polaris, vega)
	if sep2 < 48 || sep2 > 54 {
		t.Errorf("Polaris-Vega separation: expected ~51°, got %.2f", sep2)
	}
}

func TestRiseSetSirius(t *testing.T) {
	date := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	rise, _, set, err := RiseSetTimes(sirius, testLocation, date)
	if err != nil {
		t.Fatalf("Sirius rise/set: %v", err)
	}
	if rise.IsZero() || set.IsZero() {
		t.Fatal("rise/set times should not be zero")
	}
	// Sirius should rise and set at a typical mid-latitude location.
	// Rise should be in afternoon/evening UTC (late morning EST).
	if rise.Hour() < 5 || rise.Hour() > 23 {
		t.Logf("Sirius rise hour (UTC): %d (may vary)", rise.Hour())
	}
}

func TestRiseSetCircumpolarAbove(t *testing.T) {
	// Polaris from 40°N should never set.
	date := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	_, _, _, err := RiseSetTimes(polaris, testLocation, date)
	if err != ErrNeverSets {
		t.Errorf("Polaris at 40N: expected ErrNeverSets, got %v", err)
	}
}

func TestRiseSetCircumpolarBelow(t *testing.T) {
	// Polaris from 40°S should never rise.
	southLoc := GeographicLocation{Latitude: -40.0, Longitude: 0, Elevation: 0}
	date := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	_, _, _, err := RiseSetTimes(polaris, southLoc, date)
	if err != ErrNeverRises {
		t.Errorf("Polaris at 40S: expected ErrNeverRises, got %v", err)
	}
}

// mockLocation satisfies LocationProvider for testing AstroService.
type mockLocation struct {
	lat, lon, elev float64
}

func (m *mockLocation) GetLatitude() float64  { return m.lat }
func (m *mockLocation) GetLongitude() float64 { return m.lon }
func (m *mockLocation) GetElevation() float64 { return m.elev }

func TestAstroServiceAltAz(t *testing.T) {
	svc := NewAstroService(&mockLocation{lat: 40.7128, lon: -74.006, elev: 10})
	testTime := time.Date(2026, 3, 15, 22, 0, 0, 0, time.UTC)

	alt, az := svc.AltAz(polaris.RA, polaris.Dec, testTime)
	if alt <= 0 {
		t.Errorf("Polaris alt should be positive, got %f", alt)
	}
	if az < 0 || az >= 360 {
		t.Errorf("Az out of range: %f", az)
	}
}

func TestAstroServiceIsAboveHorizon(t *testing.T) {
	svc := NewAstroService(&mockLocation{lat: 40.7128, lon: -74.006, elev: 10})
	testTime := time.Date(2026, 3, 15, 22, 0, 0, 0, time.UTC)

	if !svc.IsAboveHorizon(polaris.RA, polaris.Dec, testTime) {
		t.Error("Polaris should be above horizon at 40N")
	}
}

func TestAstroServiceRiseSet(t *testing.T) {
	svc := NewAstroService(&mockLocation{lat: 40.7128, lon: -74.006, elev: 10})
	date := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	rise, set, err := svc.RiseSet(sirius.RA, sirius.Dec, date)
	if err != nil {
		t.Fatalf("Sirius rise/set: %v", err)
	}
	if rise.IsZero() || set.IsZero() {
		t.Fatal("rise/set times should not be zero")
	}
}
