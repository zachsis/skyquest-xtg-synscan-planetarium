package telescope

import (
	"sync"
	"time"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/synscan"
)

// TelescopeState holds a snapshot of the telescope's current status.
type TelescopeState struct {
	RA          float64
	Dec         float64
	Alt         float64
	Az          float64
	TrackingMode synscan.TrackingMode
	IsSlewing   bool
	IsAligned   bool
	Connected   bool
	PortName    string
	LastUpdate  time.Time
}

// PositionProvider is implemented by the live status panel and consumed
// by sky chart, Stellarium server, and GoTo panel.
type PositionProvider interface {
	CurrentState() TelescopeState
	Subscribe(func(TelescopeState)) (unsubscribe func())
}

// StatePublisher is a thread-safe PositionProvider that can be updated.
type StatePublisher struct {
	mu        sync.RWMutex
	state     TelescopeState
	cbMu      sync.RWMutex
	callbacks []func(TelescopeState)
}

// NewStatePublisher creates a new StatePublisher.
func NewStatePublisher() *StatePublisher {
	return &StatePublisher{}
}

// Update sets the current state and notifies all subscribers.
func (p *StatePublisher) Update(s TelescopeState) {
	p.mu.Lock()
	p.state = s
	p.mu.Unlock()

	p.cbMu.RLock()
	cbs := make([]func(TelescopeState), len(p.callbacks))
	copy(cbs, p.callbacks)
	p.cbMu.RUnlock()

	for _, cb := range cbs {
		if cb != nil {
			go cb(s)
		}
	}
}

// CurrentState returns the most recent telescope state.
func (p *StatePublisher) CurrentState() TelescopeState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// Subscribe registers a callback for state updates. Returns an unsubscribe func.
func (p *StatePublisher) Subscribe(cb func(TelescopeState)) (unsubscribe func()) {
	p.cbMu.Lock()
	p.callbacks = append(p.callbacks, cb)
	idx := len(p.callbacks) - 1
	p.cbMu.Unlock()

	return func() {
		p.cbMu.Lock()
		defer p.cbMu.Unlock()
		p.callbacks[idx] = nil
	}
}
