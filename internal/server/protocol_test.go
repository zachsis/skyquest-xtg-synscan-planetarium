package server

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func TestEncodeDecodeRA(t *testing.T) {
	tests := []struct {
		name    string
		raHours float64
	}{
		{"zero", 0.0},
		{"midnight", 12.0},
		{"max", 23.9999},
		{"pi-ish", 6.123456},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := encodeRA(tt.raHours)
			got := decodeRA(raw)
			// Round-trip tolerance: ~0.001 hours (~3.6 seconds of arc).
			if math.Abs(got-tt.raHours) > 0.001 {
				t.Errorf(
					"RA round-trip: input=%.6f encoded=%d decoded=%.6f",
					tt.raHours, raw, got,
				)
			}
		})
	}
}

func TestEncodeDecodeDec(t *testing.T) {
	tests := []struct {
		name      string
		decDegree float64
	}{
		{"zero", 0.0},
		{"north pole", 90.0},
		{"south pole", -90.0},
		{"positive", 45.678},
		{"negative", -23.456},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := encodeDec(tt.decDegree)
			got := decodeDec(raw)
			// Round-trip tolerance: ~0.001 degrees.
			if math.Abs(got-tt.decDegree) > 0.001 {
				t.Errorf(
					"Dec round-trip: input=%.6f encoded=%d decoded=%.6f",
					tt.decDegree, raw, got,
				)
			}
		})
	}
}

func TestMarshalCurrentPosition_Length(t *testing.T) {
	buf := marshalCurrentPosition(12.345, -30.0)
	if len(buf) != sizeCurrentPosition {
		t.Fatalf("expected %d bytes, got %d", sizeCurrentPosition, len(buf))
	}
	// Verify length field in the packet.
	msgLen := binary.LittleEndian.Uint16(buf[0:2])
	if msgLen != sizeCurrentPosition {
		t.Errorf("length field: want %d, got %d", sizeCurrentPosition, msgLen)
	}
	// Verify type field.
	msgType := binary.LittleEndian.Uint16(buf[2:4])
	if msgType != msgTypeCurrentPosition {
		t.Errorf("type field: want %d, got %d", msgTypeCurrentPosition, msgType)
	}
	// Verify status field is 0.
	status := binary.LittleEndian.Uint32(buf[20:24])
	if status != 0 {
		t.Errorf("status field: want 0, got %d", status)
	}
}

func TestMarshalCurrentPosition_Roundtrip(t *testing.T) {
	tests := []struct {
		ra  float64
		dec float64
	}{
		{0.0, 0.0},
		{6.0, 45.0},
		{23.5, -60.0},
		{12.0, 90.0},
		{18.75, -90.0},
	}
	for _, tt := range tests {
		buf := marshalCurrentPosition(tt.ra, tt.dec)

		rawRA := binary.LittleEndian.Uint32(buf[12:16])
		rawDec := int32(binary.LittleEndian.Uint32(buf[16:20]))

		gotRA := decodeRA(rawRA)
		gotDec := decodeDec(rawDec)

		if math.Abs(gotRA-tt.ra) > 0.001 {
			t.Errorf("RA roundtrip fail: in=%.4f out=%.4f", tt.ra, gotRA)
		}
		if math.Abs(gotDec-tt.dec) > 0.001 {
			t.Errorf("Dec roundtrip fail: in=%.4f out=%.4f", tt.dec, gotDec)
		}
	}
}

func TestReadGoto_Valid(t *testing.T) {
	// Build a valid 20-byte GoTo message.
	buf := make([]byte, 20)
	binary.LittleEndian.PutUint16(buf[0:2], 20) // length
	binary.LittleEndian.PutUint16(buf[2:4], 0)  // type
	binary.LittleEndian.PutUint64(buf[4:12], 0) // timestamp
	binary.LittleEndian.PutUint32(buf[12:16], encodeRA(10.5))
	binary.LittleEndian.PutUint32(buf[16:20], uint32(encodeDec(22.3)))

	// readGoto reads length first (2 bytes), then the rest.
	r := bytes.NewReader(buf)
	ra, dec, err := readGoto(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(ra-10.5) > 0.001 {
		t.Errorf("RA: want %.4f, got %.4f", 10.5, ra)
	}
	if math.Abs(dec-22.3) > 0.001 {
		t.Errorf("Dec: want %.4f, got %.4f", 22.3, dec)
	}
}

func TestReadGoto_TruncatedMessage(t *testing.T) {
	// Message says length=20 but only provides 10 bytes total.
	buf := make([]byte, 10)
	binary.LittleEndian.PutUint16(buf[0:2], 20)
	r := bytes.NewReader(buf)
	_, _, err := readGoto(r)
	if err == nil {
		t.Error("expected error for truncated message, got nil")
	}
}

func TestReadGoto_InvalidLength(t *testing.T) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf[0:2], 1) // length < 2
	r := bytes.NewReader(buf)
	_, _, err := readGoto(r)
	if err == nil {
		t.Error("expected error for invalid length, got nil")
	}
}
