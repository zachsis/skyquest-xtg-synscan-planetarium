package synscan

import "fmt"

// GotoRADec slews the telescope to the given RA (hours) and Dec (degrees) using
// the precise 32-bit GoTo command.
func (c *Controller) GotoRADec(ra, dec float64) error {
	cmd := fmt.Sprintf("r%s,%s", EncodeRA(ra), EncodeDegrees(dec))
	_, err := c.execute(cmd)
	return err
}

// GotoAltAz slews the telescope to the given Alt and Az (degrees) using
// the precise 32-bit GoTo command.
func (c *Controller) GotoAltAz(alt, az float64) error {
	cmd := fmt.Sprintf("b%s,%s", EncodeDegrees(alt), EncodeDegrees(az))
	_, err := c.execute(cmd)
	return err
}

// GetRADec reads the telescope's current RA (hours) and Dec (degrees)
// using the precise 32-bit position command.
func (c *Controller) GetRADec() (ra, dec float64, err error) {
	resp, err := c.execute("e")
	if err != nil {
		return 0, 0, err
	}
	if len(resp) < 17 || resp[8] != ',' {
		return 0, 0, fmt.Errorf("%w: expected 17 chars with comma, got %q", ErrMalformedResponse, resp)
	}
	ra, err = DecodeRA(resp[:8])
	if err != nil {
		return 0, 0, fmt.Errorf("%w: RA: %v", ErrMalformedResponse, err)
	}
	dec, err = DecodeSignedDegrees(resp[9:17])
	if err != nil {
		return 0, 0, fmt.Errorf("%w: Dec: %v", ErrMalformedResponse, err)
	}
	return ra, dec, nil
}

// GetAltAz reads the telescope's current Alt and Az (degrees)
// using the precise 32-bit position command.
func (c *Controller) GetAltAz() (alt, az float64, err error) {
	resp, err := c.execute("z")
	if err != nil {
		return 0, 0, err
	}
	if len(resp) < 17 || resp[8] != ',' {
		return 0, 0, fmt.Errorf("%w: expected 17 chars with comma, got %q", ErrMalformedResponse, resp)
	}
	alt, err = DecodeSignedDegrees(resp[:8])
	if err != nil {
		return 0, 0, fmt.Errorf("%w: Alt: %v", ErrMalformedResponse, err)
	}
	az, err = DecodeDegrees(resp[9:17])
	if err != nil {
		return 0, 0, fmt.Errorf("%w: Az: %v", ErrMalformedResponse, err)
	}
	return alt, az, nil
}

// CancelGoto stops the current GoTo slew.
func (c *Controller) CancelGoto() error {
	_, err := c.execute("M")
	return err
}

// IsGotoInProgress returns true if the telescope is currently slewing.
func (c *Controller) IsGotoInProgress() (bool, error) {
	resp, err := c.execute("L")
	if err != nil {
		return false, err
	}
	if len(resp) < 1 {
		return false, fmt.Errorf("%w: empty IsGoto response", ErrMalformedResponse)
	}
	return resp[0] == '1', nil
}

// IsAligned returns true if the telescope is aligned.
func (c *Controller) IsAligned() (bool, error) {
	resp, err := c.execute("J")
	if err != nil {
		return false, err
	}
	if len(resp) < 1 {
		return false, fmt.Errorf("%w: empty IsAligned response", ErrMalformedResponse)
	}
	// The controller returns a raw byte: 0x01 = aligned, 0x00 = not aligned.
	return resp[0] == 0x01 || resp[0] == '1', nil
}

// SyncRADec syncs the telescope's position to the given RA (hours) and Dec (degrees).
// This tells the controller "the current position is actually (ra, dec)".
func (c *Controller) SyncRADec(ra, dec float64) error {
	cmd := fmt.Sprintf("S%s,%s", EncodeRA(ra), EncodeDegrees(dec))
	_, err := c.execute(cmd)
	return err
}

// SyncAltAz syncs the telescope's position to the given Alt and Az (degrees).
func (c *Controller) SyncAltAz(alt, az float64) error {
	cmd := fmt.Sprintf("s%s,%s", EncodeDegrees(alt), EncodeDegrees(az))
	_, err := c.execute(cmd)
	return err
}

// GetTrackingMode returns the telescope's current tracking mode.
func (c *Controller) GetTrackingMode() (TrackingMode, error) {
	resp, err := c.execute("t")
	if err != nil {
		return TrackingOff, err
	}
	if len(resp) < 1 {
		return TrackingOff, fmt.Errorf("%w: empty tracking response", ErrMalformedResponse)
	}
	return TrackingMode(resp[0]), nil
}

// SetTrackingMode sets the telescope's tracking mode.
func (c *Controller) SetTrackingMode(mode TrackingMode) error {
	cmd := fmt.Sprintf("T%c", byte(mode))
	_, err := c.execute(cmd)
	return err
}

// Echo sends a byte and verifies the controller echoes it back.
func (c *Controller) Echo(b byte) (byte, error) {
	cmd := fmt.Sprintf("K%c", b)
	resp, err := c.execute(cmd)
	if err != nil {
		return 0, err
	}
	if len(resp) < 1 {
		return 0, fmt.Errorf("%w: empty echo response", ErrMalformedResponse)
	}
	return resp[0], nil
}

// GetModel returns the telescope's model identifier.
func (c *Controller) GetModel() (string, error) {
	resp, err := c.execute("m")
	if err != nil {
		return "", err
	}
	return resp, nil
}
