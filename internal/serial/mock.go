package serial

import (
	"sync"
	"sync/atomic"
)

// MockPort is a mock serial port for testing.
type MockPort struct {
	mu            sync.Mutex
	connected     atomic.Bool
	lastConnected string
	recvCh        chan []byte
	errCh         chan error
	sendBuf       [][]byte
	statusCbs     []func(bool)
}

// NewMockPort creates a new MockPort.
func NewMockPort() *MockPort {
	return &MockPort{
		recvCh: make(chan []byte, 64),
		errCh:  make(chan error, 8),
	}
}

func (m *MockPort) Connect(portName string, _ PortConfig) error {
	if m.connected.Load() {
		return ErrAlreadyConnected
	}
	m.connected.Store(true)
	m.mu.Lock()
	m.lastConnected = portName
	cbs := make([]func(bool), len(m.statusCbs))
	copy(cbs, m.statusCbs)
	m.mu.Unlock()
	for _, cb := range cbs {
		cb(true)
	}
	return nil
}

func (m *MockPort) Disconnect() error {
	if !m.connected.Load() {
		return nil
	}
	m.connected.Store(false)
	m.mu.Lock()
	cbs := make([]func(bool), len(m.statusCbs))
	copy(cbs, m.statusCbs)
	m.mu.Unlock()
	for _, cb := range cbs {
		cb(false)
	}
	return nil
}

func (m *MockPort) Send(data []byte) error {
	if !m.connected.Load() {
		return ErrNotConnected
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	m.sendBuf = append(m.sendBuf, cp)
	return nil
}

func (m *MockPort) Receive() <-chan []byte {
	return m.recvCh
}

func (m *MockPort) Errors() <-chan error {
	return m.errCh
}

func (m *MockPort) IsConnected() bool {
	return m.connected.Load()
}

func (m *MockPort) LastConnectedPort() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastConnected
}

func (m *MockPort) OnStatusChange(cb func(bool)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statusCbs = append(m.statusCbs, cb)
}

// InjectResponse sends data into the receive channel (simulates incoming serial data).
func (m *MockPort) InjectResponse(data []byte) {
	m.recvCh <- data
}

// SentData returns all data that was sent via Send() and clears the buffer.
func (m *MockPort) SentData() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.sendBuf
	m.sendBuf = nil
	return out
}
