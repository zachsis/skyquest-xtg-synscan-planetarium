package synscan

import "errors"

var (
	ErrTimeout           = errors.New("synscan: response timeout")
	ErrMalformedResponse = errors.New("synscan: malformed response")
	ErrNotConnected      = errors.New("synscan: serial port not connected")
)
