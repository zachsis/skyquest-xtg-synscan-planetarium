package serial

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	hwserial "go.bug.st/serial"
)

// PortInterface defines the contract for serial port implementations.
// Both Port and MockPort satisfy this interface.
type PortInterface interface {
	Connect(portName string, config PortConfig) error
	Disconnect() error
	Send(data []byte) error
	Receive() <-chan []byte
	Errors() <-chan error
	IsConnected() bool
	LastConnectedPort() string
	OnStatusChange(func(bool))
}

// Port manages a serial port connection with channel-based I/O.
type Port struct {
	mu            sync.Mutex
	port          hwserial.Port
	config        PortConfig
	portName      string
	lastConnected string
	recvCh        chan []byte
	errCh         chan error
	stopCh        chan struct{}
	connected     atomic.Bool
	statusCbs     []func(bool)
}

// NewPort creates a new Port with default configuration.
func NewPort() *Port {
	return &Port{
		recvCh: make(chan []byte, 64),
		errCh:  make(chan error, 8),
	}
}

// Connect opens the serial port with the given configuration.
func (p *Port) Connect(portName string, cfg PortConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.connected.Load() {
		return ErrAlreadyConnected
	}

	parity := hwserial.NoParity
	switch cfg.Parity {
	case OddParity:
		parity = hwserial.OddParity
	case EvenParity:
		parity = hwserial.EvenParity
	}

	stopBits := hwserial.OneStopBit
	if cfg.StopBits == TwoStopBits {
		stopBits = hwserial.TwoStopBits
	}

	mode := &hwserial.Mode{
		BaudRate: cfg.BaudRate,
		DataBits: cfg.DataBits,
		Parity:   parity,
		StopBits: stopBits,
	}

	port, err := hwserial.Open(portName, mode)
	if err != nil {
		return fmt.Errorf("serial open %s: %w", portName, err)
	}

	if err := port.SetReadTimeout(cfg.ReadTimeout); err != nil {
		port.Close()
		return fmt.Errorf("serial set timeout: %w", err)
	}

	p.port = port
	p.portName = portName
	p.config = cfg
	p.lastConnected = portName
	p.stopCh = make(chan struct{})
	p.connected.Store(true)

	go p.readLoop()

	cbs := make([]func(bool), len(p.statusCbs))
	copy(cbs, p.statusCbs)
	go func() {
		for _, cb := range cbs {
			cb(true)
		}
	}()

	return nil
}

// Disconnect closes the serial port and stops the read goroutine.
func (p *Port) Disconnect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.connected.Load() {
		return nil
	}

	close(p.stopCh)
	p.connected.Store(false)

	var closeErr error
	if p.port != nil {
		closeErr = p.port.Close()
		p.port = nil
	}

	cbs := make([]func(bool), len(p.statusCbs))
	copy(cbs, p.statusCbs)
	go func() {
		for _, cb := range cbs {
			cb(false)
		}
	}()

	return closeErr
}

// Send writes data to the serial port.
func (p *Port) Send(data []byte) error {
	if !p.connected.Load() {
		return ErrNotConnected
	}

	p.mu.Lock()
	port := p.port
	p.mu.Unlock()

	if port == nil {
		return ErrNotConnected
	}

	_, err := port.Write(data)
	if err != nil {
		return fmt.Errorf("serial write: %w", err)
	}
	return nil
}

// Receive returns a read-only channel that emits received byte slices.
func (p *Port) Receive() <-chan []byte {
	return p.recvCh
}

// Errors returns a read-only channel for asynchronous errors.
func (p *Port) Errors() <-chan error {
	return p.errCh
}

// IsConnected reports current connection status.
func (p *Port) IsConnected() bool {
	return p.connected.Load()
}

// LastConnectedPort returns the port name from the last successful connection.
func (p *Port) LastConnectedPort() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastConnected
}

// OnStatusChange registers a callback for connection state changes.
func (p *Port) OnStatusChange(cb func(bool)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.statusCbs = append(p.statusCbs, cb)
}

func (p *Port) readLoop() {
	buf := make([]byte, 4096)
	for {
		select {
		case <-p.stopCh:
			return
		default:
		}

		p.mu.Lock()
		port := p.port
		p.mu.Unlock()

		if port == nil {
			return
		}

		n, err := port.Read(buf)
		if err != nil {
			// Check if we were asked to stop before reporting the error.
			select {
			case <-p.stopCh:
				return
			default:
			}
			p.handleDisconnect(err)
			return
		}
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			select {
			case p.recvCh <- data:
			case <-p.stopCh:
				return
			}
		}
	}
}

func (p *Port) handleDisconnect(readErr error) {
	p.mu.Lock()
	wasConnected := p.connected.Load()
	if wasConnected {
		p.connected.Store(false)
		if p.port != nil {
			p.port.Close()
			p.port = nil
		}
	}
	cbs := make([]func(bool), len(p.statusCbs))
	copy(cbs, p.statusCbs)
	p.mu.Unlock()

	if wasConnected {
		// Notify error channel with a timeout to avoid blocking indefinitely.
		select {
		case p.errCh <- fmt.Errorf("serial read: %w", ErrPortClosed):
		case <-time.After(time.Second):
		}

		for _, cb := range cbs {
			cb(false)
		}
	}
}
