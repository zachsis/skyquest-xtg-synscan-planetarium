// Package server implements the Stellarium Telescope Protocol TCP server
// and optionally the LX200 ASCII protocol.
package server

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

// Stellarium Telescope Protocol message sizes (bytes).
const (
	sizeCurrentPosition = 24
	sizeGoto            = 20

	msgTypeCurrentPosition uint16 = 0
	msgTypeGoto            uint16 = 0

	maxClients = 5
)

// msgCurrentPosition is the server→client binary message (24 bytes).
type msgCurrentPosition struct {
	Length uint16 // always sizeCurrentPosition
	Type   uint16 // always msgTypeCurrentPosition
	Time   int64  // microseconds since Unix epoch
	RA     uint32 // encoded RA
	Dec    int32  // encoded Dec
	Status int32  // always 0
}

// msgGoto is the client→server binary message (20 bytes).
type msgGoto struct {
	Length uint16
	Type   uint16
	Time   int64
	RA     uint32
	Dec    int32
}

// encodeRA converts RA hours [0, 24) to the Stellarium uint32 wire format.
func encodeRA(raHours float64) uint32 {
	return uint32(raHours / 24.0 * 0x100000000)
}

// decodeRA converts the Stellarium uint32 wire format back to RA hours.
func decodeRA(raw uint32) float64 {
	return float64(raw) / 0x100000000 * 24.0
}

// encodeDec converts declination degrees [-90, 90] to the Stellarium int32 wire format.
func encodeDec(decDegrees float64) int32 {
	return int32(decDegrees / 90.0 * 0x40000000)
}

// decodeDec converts the Stellarium int32 wire format back to declination degrees.
func decodeDec(raw int32) float64 {
	return float64(raw) / 0x40000000 * 90.0
}

// marshalCurrentPosition encodes a position update into the 24-byte wire format.
func marshalCurrentPosition(ra, dec float64) []byte {
	buf := make([]byte, sizeCurrentPosition)
	binary.LittleEndian.PutUint16(buf[0:2], sizeCurrentPosition)
	binary.LittleEndian.PutUint16(buf[2:4], msgTypeCurrentPosition)
	us := time.Now().UnixMicro()
	binary.LittleEndian.PutUint64(buf[4:12], uint64(us))
	binary.LittleEndian.PutUint32(buf[12:16], encodeRA(ra))
	binary.LittleEndian.PutUint32(buf[16:20], uint32(encodeDec(dec)))
	binary.LittleEndian.PutUint32(buf[20:24], 0) // status
	return buf
}

// readGoto reads a single client→server GoTo message from r using length-prefix
// framing. It returns the decoded RA (hours) and Dec (degrees).
func readGoto(r io.Reader) (ra, dec float64, err error) {
	// Read the 2-byte length prefix first.
	var lenBuf [2]byte
	if _, err = io.ReadFull(r, lenBuf[:]); err != nil {
		return 0, 0, fmt.Errorf("protocol: read length: %w", err)
	}
	msgLen := binary.LittleEndian.Uint16(lenBuf[:])
	if msgLen < 2 {
		return 0, 0, fmt.Errorf("protocol: invalid message length %d", msgLen)
	}

	// Read the remaining msgLen-2 bytes.
	rest := make([]byte, msgLen-2)
	if _, err = io.ReadFull(r, rest); err != nil {
		return 0, 0, fmt.Errorf("protocol: read body: %w", err)
	}

	// We need at least 18 more bytes for a valid GoTo message (type + time + ra + dec).
	if len(rest) < 18 {
		return 0, 0, fmt.Errorf(
			"protocol: goto message too short: %d bytes", msgLen,
		)
	}

	msgType := binary.LittleEndian.Uint16(rest[0:2])
	if msgType != msgTypeGoto {
		return 0, 0, fmt.Errorf("protocol: unexpected message type %d", msgType)
	}

	// Bytes 2-9: int64 timestamp (ignored).
	rawRA := binary.LittleEndian.Uint32(rest[10:14])
	rawDec := int32(binary.LittleEndian.Uint32(rest[14:18]))

	return decodeRA(rawRA), decodeDec(rawDec), nil
}
