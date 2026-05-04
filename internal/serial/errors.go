package serial

import "errors"

var (
	ErrNotConnected     = errors.New("serial: port not connected")
	ErrAlreadyConnected = errors.New("serial: port already connected")
	ErrPortNotFound     = errors.New("serial: port not found")
	ErrPortClosed       = errors.New("serial: port closed")
)
