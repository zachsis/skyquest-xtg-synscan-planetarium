package serial

import "time"

// Parity represents serial port parity settings.
type Parity int

const (
	NoParity Parity = iota
	OddParity
	EvenParity
)

// StopBits represents serial port stop bit settings.
type StopBits int

const (
	OneStopBit StopBits = iota
	TwoStopBits
)

// PortConfig holds serial port configuration parameters.
type PortConfig struct {
	BaudRate    int
	DataBits    int
	Parity      Parity
	StopBits    StopBits
	ReadTimeout time.Duration
}

// DefaultConfig returns the default serial port configuration for SynScan (9600 8N1).
func DefaultConfig() PortConfig {
	return PortConfig{
		BaudRate:    9600,
		DataBits:    8,
		Parity:      NoParity,
		StopBits:    OneStopBit,
		ReadTimeout: 100 * time.Millisecond,
	}
}
