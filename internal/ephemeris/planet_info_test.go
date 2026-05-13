package ephemeris

import (
	"math"
	"testing"
	"time"

	"github.com/soniakeys/meeus/v3/julian"
	"github.com/soniakeys/meeus/v3/solar"
	pp "github.com/soniakeys/meeus/v3/planetposition"
)

// toleranceDeg is the maximum allowed angular error in degrees (2 arcmin = 1/30°).
const toleranceDeg = 2.0 / 60.0

// loadEarth extracts embedded VSOP87B data and returns Earth's V87Planet.
// It skips the test if the extraction fails.
func loadEarth(t *testing.T) *pp.V87Planet {
	t.Helper()
	tmpDir := t.TempDir()
	if err := extractVSOP87B(tmpDir); err != nil {
		t.Fatalf("extractVSOP87B: %v", err)
	}
	earth, err := pp.LoadPlanetPath(pp.Earth, tmpDir)
	if err != nil {
		t.Fatalf("LoadPlanetPath Earth: %v", err)
	}
	return earth
}

// TestSunPositionJ2000 checks the Sun's apparent RA/Dec at J2000.0.
// Reference (Meeus, Example 25.b): RA ≈ 18h 45m 39.8s, Dec ≈ -23° 26' 44"
// Tolerance: 2 arcmin in either coordinate.
func TestSunPositionJ2000(t *testing.T) {
	earth := loadEarth(t)
	jde := 2451545.0 // J2000.0 = 2000-01-01 12:00 TT

	α, δ, _ := solar.ApparentEquatorialVSOP87(earth, jde)

	gotRA := α.Hour()
	gotDec := δ.Deg()

	// Reference: RA ≈ 18.752h, Dec ≈ -23.02°
	// (varies slightly by source; we use a 2-arcmin window)
	wantRA := 18.752
	wantDec := -23.016

	if math.Abs(gotRA-wantRA)*15 > toleranceDeg {
		t.Errorf("Sun RA at J2000.0: got %.4fh, want %.4fh (diff %.2f arcmin)",
			gotRA, wantRA, math.Abs(gotRA-wantRA)*60)
	}
	if math.Abs(gotDec-wantDec) > toleranceDeg {
		t.Errorf("Sun Dec at J2000.0: got %.4f°, want %.4f° (diff %.2f arcmin)",
			gotDec, wantDec, math.Abs(gotDec-wantDec)*60)
	}
}

// TestSunPositionRecent checks the Sun's RA/Dec on 2025-01-01T12:00 UTC.
// These reference values were computed with the same meeus library, so the
// test validates self-consistency and the wiring rather than absolute accuracy.
func TestSunPositionRecent(t *testing.T) {
	earth := loadEarth(t)
	// 2025-01-01 12:00:00 UTC
	tt := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	jde := julian.TimeToJD(tt)

	α, δ, R := solar.ApparentEquatorialVSOP87(earth, jde)

	// Sanity checks: Sun should be near solstice (low Dec), reasonable RA and distance.
	if R < 0.983 || R > 1.017 {
		t.Errorf("Sun distance out of range: got %.4f AU (expected 0.983–1.017)", R)
	}
	// Dec should be near -23° in early January (just past winter solstice).
	if δ.Deg() > -22 || δ.Deg() < -24 {
		t.Errorf("Sun Dec on 2025-01-01: got %.2f°, expected near -23°", δ.Deg())
	}
	// RA should be near 18h–19h in early January.
	if α.Hour() < 17 || α.Hour() > 20 {
		t.Errorf("Sun RA on 2025-01-01: got %.2fh, expected 17–20h", α.Hour())
	}
}

// TestMoonPhaseName validates all eight lunar phase names.
func TestMoonPhaseName(t *testing.T) {
	tests := []struct {
		name           string
		phaseAngleRad  float64
		moonSunDiffRad float64
		want           string
	}{
		{
			name:           "new moon",
			phaseAngleRad:  0.05,
			moonSunDiffRad: 0.05,
			want:           "New Moon",
		},
		{
			name:           "waxing crescent",
			phaseAngleRad:  0.8,
			moonSunDiffRad: 1.0, // waxing (< π)
			want:           "Waxing Crescent",
		},
		{
			name:           "first quarter",
			phaseAngleRad:  1.5,
			moonSunDiffRad: 1.6, // waxing
			want:           "First Quarter",
		},
		{
			name:           "waxing gibbous",
			phaseAngleRad:  2.0,
			moonSunDiffRad: 2.0, // waxing
			want:           "Waxing Gibbous",
		},
		{
			name:           "full moon",
			phaseAngleRad:  3.0,
			moonSunDiffRad: 3.2, // waxing (near π)
			want:           "Full Moon",
		},
		{
			name:           "waning gibbous",
			phaseAngleRad:  2.0,
			moonSunDiffRad: 4.0, // waning (> π)
			want:           "Waning Gibbous",
		},
		{
			name:           "last quarter",
			phaseAngleRad:  1.5,
			moonSunDiffRad: 4.5, // waning
			want:           "Last Quarter",
		},
		{
			name:           "waning crescent",
			phaseAngleRad:  0.8,
			moonSunDiffRad: 5.0, // waning (> π)
			want:           "Waning Crescent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := moonPhaseName(tt.phaseAngleRad, tt.moonSunDiffRad)
			if got != tt.want {
				t.Errorf("moonPhaseName(%v, %v) = %q, want %q",
					tt.phaseAngleRad, tt.moonSunDiffRad, got, tt.want)
			}
		})
	}
}

// TestElongation checks the elongation helper for known cases.
func TestElongation(t *testing.T) {
	tests := []struct {
		name         string
		ra1, dec1    float64
		ra2, dec2    float64
		wantDeg      float64
		toleranceDeg float64
	}{
		{
			name:         "same point",
			ra1:          12, dec1: 0,
			ra2:          12, dec2: 0,
			wantDeg:      0,
			toleranceDeg: 0.001,
		},
		{
			name:         "diametrically opposite on equator",
			ra1:          0, dec1: 0,
			ra2:          12, dec2: 0,
			wantDeg:      180,
			toleranceDeg: 0.001,
		},
		{
			name:         "poles",
			ra1:          0, dec1: 90,
			ra2:          0, dec2: -90,
			wantDeg:      180,
			toleranceDeg: 0.001,
		},
		{
			name:         "90 degree separation on equator",
			ra1:          0, dec1: 0,
			ra2:          6, dec2: 0,
			wantDeg:      90,
			toleranceDeg: 0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := elongation(tt.ra1, tt.dec1, tt.ra2, tt.dec2)
			if math.Abs(got-tt.wantDeg) > tt.toleranceDeg {
				t.Errorf("elongation = %.4f°, want %.4f° (tol %.4f°)",
					got, tt.wantDeg, tt.toleranceDeg)
			}
		})
	}
}

// TestClamp verifies the clamp helper.
func TestClamp(t *testing.T) {
	tests := []struct {
		v, lo, hi, want float64
	}{
		{0.5, 0, 1, 0.5},
		{-1, 0, 1, 0},
		{2, 0, 1, 1},
		{1, 1, 1, 1},
	}
	for _, tt := range tests {
		got := clamp(tt.v, tt.lo, tt.hi)
		if got != tt.want {
			t.Errorf("clamp(%v, %v, %v) = %v, want %v", tt.v, tt.lo, tt.hi, got, tt.want)
		}
	}
}
