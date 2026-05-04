package synscan

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/serial"
)

// Controller manages SynScan protocol communication over a serial port.
// All methods are safe for concurrent use; the mutex serializes command execution.
type Controller struct {
	port    serial.PortInterface
	mu      sync.Mutex
	timeout time.Duration
}

// NewController creates a new SynScan controller wrapping the given serial port.
func NewController(port serial.PortInterface) *Controller {
	return &Controller{
		port:    port,
		timeout: 3 * time.Second,
	}
}

// SetTimeout overrides the default 3s response timeout. Useful for tests.
func (c *Controller) SetTimeout(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.timeout = d
}

// execute sends a command and reads the response up to the '#' terminator.
// The caller must NOT hold c.mu; this method acquires it.
func (c *Controller) execute(cmd string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.port.IsConnected() {
		return "", ErrNotConnected
	}

	c.drain()

	if err := c.port.Send([]byte(cmd)); err != nil {
		return "", fmt.Errorf("synscan send: %w", err)
	}

	return c.readResponse()
}

// drain reads and discards all pending bytes from the port's Receive() channel.
func (c *Controller) drain() {
	timeout := time.After(50 * time.Millisecond)
	for {
		select {
		case <-c.port.Receive():
			// discard
		case <-timeout:
			return
		}
	}
}

// readResponse accumulates bytes from the receive channel until a '#' terminator
// is found or the timeout expires.
func (c *Controller) readResponse() (string, error) {
	deadline := time.After(c.timeout)
	var resp bytes.Buffer
	for {
		select {
		case data := <-c.port.Receive():
			resp.Write(data)
			if idx := bytes.IndexByte(resp.Bytes(), '#'); idx >= 0 {
				return string(resp.Bytes()[:idx]), nil
			}
		case <-deadline:
			return "", ErrTimeout
		}
	}
}
