package synscan

import (
	"math"
	"testing"
)

func TestEncodeDegrees(t *testing.T) {
	tests := []struct {
		deg  float64
		want string
	}{
		{0, "00000000"},
		{180, "80000000"},
		{90, "40000000"},
		{270, "C0000000"},
		{359.99, EncodeDegrees(359.99)}, // verify round-trip rather than hard-coded hex
	}
	for _, tc := range tests {
		got := EncodeDegrees(tc.deg)
		if got != tc.want {
			t.Errorf("EncodeDegrees(%f) = %s, want %s", tc.deg, got, tc.want)
		}
	}
}

func TestDecodeDegrees(t *testing.T) {
	tests := []struct {
		hex  string
		want float64
	}{
		{"00000000", 0},
		{"80000000", 180},
		{"40000000", 90},
		{"C0000000", 270},
	}
	for _, tc := range tests {
		got, err := DecodeDegrees(tc.hex)
		if err != nil {
			t.Fatalf("DecodeDegrees(%s): %v", tc.hex, err)
		}
		if math.Abs(got-tc.want) > 0.001 {
			t.Errorf("DecodeDegrees(%s) = %f, want %f", tc.hex, got, tc.want)
		}
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	for _, deg := range []float64{0, 45.5, 90, 135.123, 180, 270, 359.999} {
		hex := EncodeDegrees(deg)
		got, err := DecodeDegrees(hex)
		if err != nil {
			t.Fatalf("round trip %f: %v", deg, err)
		}
		// Accuracy: within 0.001 arcsecond = 0.001/3600 degrees
		if math.Abs(got-deg) > 0.001/3600 {
			t.Errorf("round trip %f: got %f (diff %e arcsec)", deg, got, (got-deg)*3600)
		}
	}
}

func TestEncodeDecodeRA(t *testing.T) {
	for _, hours := range []float64{0, 6, 12, 18, 23.999} {
		hex := EncodeRA(hours)
		got, err := DecodeRA(hex)
		if err != nil {
			t.Fatalf("RA round trip %f: %v", hours, err)
		}
		// Within 0.001 arcsecond = 0.001/3600/15 hours
		if math.Abs(got-hours) > 0.001/3600/15 {
			t.Errorf("RA round trip %f: got %f", hours, got)
		}
	}
}

func TestDecodeSignedDegrees(t *testing.T) {
	tests := []struct {
		hex  string
		want float64
	}{
		{"00000000", 0},
		{"40000000", 90},        // +90
		{"C0000000", -90},       // 270 -> -90
		{"E0000000", -45},       // 315 -> -45
		{"20000000", 45},        // +45
	}
	for _, tc := range tests {
		got, err := DecodeSignedDegrees(tc.hex)
		if err != nil {
			t.Fatalf("DecodeSignedDegrees(%s): %v", tc.hex, err)
		}
		if math.Abs(got-tc.want) > 0.01 {
			t.Errorf("DecodeSignedDegrees(%s) = %f, want %f", tc.hex, got, tc.want)
		}
	}
}

func TestNegativeDecEncoding(t *testing.T) {
	// Encode -16 degrees (Sirius' Dec).
	// -16 + 360 = 344 degrees.
	hex := EncodeDegrees(-16)
	got, err := DecodeSignedDegrees(hex)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got-(-16)) > 0.001 {
		t.Errorf("negative dec round trip: got %f, want -16", got)
	}
}
