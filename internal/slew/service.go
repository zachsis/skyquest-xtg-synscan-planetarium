package slew

import (
	"fmt"
	"sync"
	"time"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/synscan"
)

// SlewState represents the current state of the slew service.
type SlewState int

const (
	SlewIdle      SlewState = iota
	SlewSlewing
	SlewComplete
	SlewCancelled
	SlewFailed
)

// SlewEvent carries context about a slew state transition.
type SlewEvent struct {
	Type       SlewState
	TargetName string
	CatalogID  string
	TargetRA   float64
	TargetDec  float64
	Timestamp  time.Time
}

// GoToService is the interface for all telescope slew operations.
type GoToService interface {
	SlewToCoordinates(ra, dec float64) error
	SlewToObject(name, catalogID string, ra, dec float64) error
	SyncPosition(ra, dec float64) error
	CancelSlew() error
	SlewState() SlewState
	OnSlew(func(SlewEvent)) (unsubscribe func())
}

type goToServiceImpl struct {
	ctrl      *synscan.Controller
	mu        sync.Mutex
	state     SlewState
	cancelCh  chan struct{}
	callbacks []func(SlewEvent)
	cbMu      sync.RWMutex
}

// NewGoToService creates a new GoToService wrapping the given SynScan controller.
func NewGoToService(ctrl *synscan.Controller) GoToService {
	return &goToServiceImpl{
		ctrl:  ctrl,
		state: SlewIdle,
	}
}

func (s *goToServiceImpl) SlewToCoordinates(ra, dec float64) error {
	return s.slewTo("", "", ra, dec)
}

func (s *goToServiceImpl) SlewToObject(name, catalogID string, ra, dec float64) error {
	return s.slewTo(name, catalogID, ra, dec)
}

func (s *goToServiceImpl) slewTo(name, catalogID string, ra, dec float64) error {
	if ra < 0 || ra >= 24 {
		return fmt.Errorf("slew: RA %f out of range [0, 24)", ra)
	}
	if dec < -90 || dec > 90 {
		return fmt.Errorf("slew: Dec %f out of range [-90, 90]", dec)
	}

	s.mu.Lock()
	if s.state == SlewSlewing {
		s.mu.Unlock()
		return fmt.Errorf("slew: already in progress")
	}
	s.state = SlewSlewing
	cancelCh := make(chan struct{})
	s.cancelCh = cancelCh
	s.mu.Unlock()

	if err := s.ctrl.GotoRADec(ra, dec); err != nil {
		s.mu.Lock()
		s.state = SlewFailed
		s.mu.Unlock()
		s.fire(SlewEvent{Type: SlewFailed, TargetName: name, CatalogID: catalogID, TargetRA: ra, TargetDec: dec, Timestamp: time.Now()})
		return fmt.Errorf("slew: %w", err)
	}

	s.fire(SlewEvent{Type: SlewSlewing, TargetName: name, CatalogID: catalogID, TargetRA: ra, TargetDec: dec, Timestamp: time.Now()})

	go s.pollCompletion(name, catalogID, ra, dec, cancelCh)
	return nil
}

func (s *goToServiceImpl) pollCompletion(name, catalogID string, ra, dec float64, cancelCh chan struct{}) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-cancelCh:
			return
		case <-ticker.C:
			inProgress, err := s.ctrl.IsGotoInProgress()
			if err != nil {
				s.mu.Lock()
				s.state = SlewFailed
				s.mu.Unlock()
				s.fire(SlewEvent{Type: SlewFailed, TargetName: name, CatalogID: catalogID, TargetRA: ra, TargetDec: dec, Timestamp: time.Now()})
				return
			}
			if !inProgress {
				s.mu.Lock()
				s.state = SlewComplete
				s.mu.Unlock()
				s.fire(SlewEvent{Type: SlewComplete, TargetName: name, CatalogID: catalogID, TargetRA: ra, TargetDec: dec, Timestamp: time.Now()})
				return
			}
		}
	}
}

func (s *goToServiceImpl) SyncPosition(ra, dec float64) error {
	return s.ctrl.SyncRADec(ra, dec)
}

func (s *goToServiceImpl) CancelSlew() error {
	s.mu.Lock()
	if s.state != SlewSlewing {
		s.mu.Unlock()
		return nil
	}
	if s.cancelCh != nil {
		close(s.cancelCh)
		s.cancelCh = nil
	}
	s.state = SlewCancelled
	s.mu.Unlock()

	if err := s.ctrl.CancelGoto(); err != nil {
		return fmt.Errorf("cancel slew: %w", err)
	}
	s.fire(SlewEvent{Type: SlewCancelled, Timestamp: time.Now()})
	return nil
}

func (s *goToServiceImpl) SlewState() SlewState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *goToServiceImpl) OnSlew(cb func(SlewEvent)) (unsubscribe func()) {
	s.cbMu.Lock()
	s.callbacks = append(s.callbacks, cb)
	idx := len(s.callbacks) - 1
	s.cbMu.Unlock()

	return func() {
		s.cbMu.Lock()
		defer s.cbMu.Unlock()
		s.callbacks[idx] = nil
	}
}

func (s *goToServiceImpl) fire(evt SlewEvent) {
	s.cbMu.RLock()
	cbs := make([]func(SlewEvent), len(s.callbacks))
	copy(cbs, s.callbacks)
	s.cbMu.RUnlock()
	for _, cb := range cbs {
		if cb != nil {
			cb(evt)
		}
	}
}
