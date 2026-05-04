package synscan

import (
	"fmt"
	"math"
	"strconv"
)

const fullRevolution = 0x100000000 // 2^32

// EncodeDegrees encodes a degree value (0-360) to an 8-character uppercase hex string
// representing a fraction of a full revolution.
func EncodeDegrees(deg float64) string {
	deg = math.Mod(deg, 360.0)
	if deg < 0 {
		deg += 360.0
	}
	val := uint32(deg / 360.0 * fullRevolution)
	return fmt.Sprintf("%08X", val)
}

// DecodeDegrees decodes an 8-character hex string to degrees (0-360).
func DecodeDegrees(hex string) (float64, error) {
	val, err := strconv.ParseUint(hex, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("decode position: %w", err)
	}
	return float64(val) / fullRevolution * 360.0, nil
}

// DecodeSignedDegrees decodes an 8-character hex string to signed degrees (-180 to +180).
// Values > 180 are treated as negative (wrapping convention for declination).
func DecodeSignedDegrees(hex string) (float64, error) {
	deg, err := DecodeDegrees(hex)
	if err != nil {
		return 0, err
	}
	if deg > 180 {
		deg -= 360
	}
	return deg, nil
}

// EncodeRA encodes right ascension in hours (0-24) to an 8-character hex string.
func EncodeRA(hours float64) string {
	return EncodeDegrees(hours / 24.0 * 360.0)
}

// DecodeRA decodes an 8-character hex string to right ascension in hours (0-24).
func DecodeRA(hex string) (float64, error) {
	deg, err := DecodeDegrees(hex)
	if err != nil {
		return 0, err
	}
	return deg / 360.0 * 24.0, nil
}
