package slew

import (
	"sync"
	"testing"
	"time"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/serial"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/synscan"
)

// setupScripted creates a controller whose mock auto-responds to commands.
// script maps first-byte to response. Returns the service and a stop func.
func setupScripted(t *testing.T, script map[byte]string) (GoToService, func()) {
	t.Helper()
	mock := serial.NewMockPort()
	if err := mock.Connect("/dev/test", serial.DefaultConfig()); err != nil {
		t.Fatal(err)
	}
	ctrl := synscan.NewController(mock)
	ctrl.SetTimeout(500 * time.Millisecond)

	stopCh := make(chan struct{})
	// responseDelay must exceed drain timeout (50ms).
	mock.ScriptResponses(script, 80*time.Millisecond, stopCh)

	svc := NewGoToService(ctrl)
	return svc, func() { close(stopCh) }
}

func TestSlewToCoordinatesValidation(t *testing.T) {
	svc, stop := setupScripted(t, map[byte]string{})
	defer stop()

	if err := svc.SlewToCoordinates(-1, 0); err == nil {
		t.Error("expected error for negative RA")
	}
	if err := svc.SlewToCoordinates(25, 0); err == nil {
		t.Error("expected error for RA >= 24")
	}
	if err := svc.SlewToCoordinates(12, -91); err == nil {
		t.Error("expected error for Dec < -90")
	}
	if err := svc.SlewToCoordinates(12, 91); err == nil {
		t.Error("expected error for Dec > 90")
	}
}

func TestSlewToCoordinatesSuccess(t *testing.T) {
	// 'r' = GoTo ack, 'L' = first call "1" (slewing), then "0" (done).
	callCount := 0
	var mu sync.Mutex

	mock := serial.NewMockPort()
	if err := mock.Connect("/dev/test", serial.DefaultConfig()); err != nil {
		t.Fatal(err)
	}
	ctrl := synscan.NewController(mock)
	ctrl.SetTimeout(500 * time.Millisecond)

	stopCh := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				sent := mock.SentData()
				for _, cmd := range sent {
					if len(cmd) == 0 {
						continue
					}
					var resp string
					switch cmd[0] {
					case 'r', 'b', 'S', 'M':
						resp = "#"
					case 'L':
						mu.Lock()
						callCount++
						if callCount >= 2 {
							resp = "0#" // done
						} else {
							resp = "1#" // still slewing
						}
						mu.Unlock()
					default:
						resp = "#"
					}
					r := resp
					go func() {
						time.Sleep(80 * time.Millisecond)
						mock.InjectResponse([]byte(r))
					}()
				}
			}
		}
	}()
	defer close(stopCh)

	svc := NewGoToService(ctrl)

	done := make(chan SlewEvent, 1)
	svc.OnSlew(func(e SlewEvent) {
		if e.Type == SlewComplete {
			done <- e
		}
	})

	if err := svc.SlewToCoordinates(6.0, 45.0); err != nil {
		t.Fatalf("SlewToCoordinates: %v", err)
	}
	if svc.SlewState() != SlewSlewing {
		t.Errorf("expected SlewSlewing, got %v", svc.SlewState())
	}

	select {
	case e := <-done:
		if e.TargetRA != 6.0 || e.TargetDec != 45.0 {
			t.Errorf("event coords: RA=%f Dec=%f", e.TargetRA, e.TargetDec)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for SlewComplete")
	}
	if svc.SlewState() != SlewComplete {
		t.Errorf("expected SlewComplete, got %v", svc.SlewState())
	}
}

func TestSlewToObject(t *testing.T) {
	callCount := 0
	var mu sync.Mutex

	mock := serial.NewMockPort()
	_ = mock.Connect("/dev/test", serial.DefaultConfig())
	ctrl := synscan.NewController(mock)
	ctrl.SetTimeout(500 * time.Millisecond)

	stopCh := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				for _, cmd := range mock.SentData() {
					if len(cmd) == 0 {
						continue
					}
					var resp string
					if cmd[0] == 'L' {
						mu.Lock()
						callCount++
						if callCount >= 2 {
							resp = "0#"
						} else {
							resp = "1#"
						}
						mu.Unlock()
					} else {
						resp = "#"
					}
					r := resp
					go func() {
						time.Sleep(80 * time.Millisecond)
						mock.InjectResponse([]byte(r))
					}()
				}
			}
		}
	}()
	defer close(stopCh)

	svc := NewGoToService(ctrl)
	done := make(chan SlewEvent, 1)
	svc.OnSlew(func(e SlewEvent) {
		if e.Type == SlewComplete {
			done <- e
		}
	})

	if err := svc.SlewToObject("Andromeda Galaxy", "M31", 0.712, 41.269); err != nil {
		t.Fatalf("SlewToObject: %v", err)
	}

	select {
	case e := <-done:
		if e.TargetName != "Andromeda Galaxy" || e.CatalogID != "M31" {
			t.Errorf("event: name=%q catalogID=%q", e.TargetName, e.CatalogID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestConcurrentSlewRejected(t *testing.T) {
	svc, stop := setupScripted(t, map[byte]string{'r': "#"})
	defer stop()

	if err := svc.SlewToCoordinates(6.0, 45.0); err != nil {
		t.Fatalf("first slew: %v", err)
	}
	// Second slew while first is active.
	if err := svc.SlewToCoordinates(12.0, 0.0); err == nil {
		t.Error("expected error for concurrent slew")
	}
}

func TestCancelSlew(t *testing.T) {
	svc, stop := setupScripted(t, map[byte]string{'r': "#", 'M': "#"})
	defer stop()

	done := make(chan struct{}, 1)
	svc.OnSlew(func(e SlewEvent) {
		if e.Type == SlewCancelled {
			done <- struct{}{}
		}
	})

	if err := svc.SlewToCoordinates(6.0, 45.0); err != nil {
		t.Fatalf("slew: %v", err)
	}
	time.Sleep(150 * time.Millisecond) // let slew start

	if err := svc.CancelSlew(); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for SlewCancelled")
	}
	if svc.SlewState() != SlewCancelled {
		t.Errorf("expected SlewCancelled, got %v", svc.SlewState())
	}
}

func TestOnSlewUnsubscribe(t *testing.T) {
	svc, stop := setupScripted(t, map[byte]string{'r': "#"})
	defer stop()

	count := 0
	unsub := svc.OnSlew(func(e SlewEvent) { count++ })
	unsub()

	_ = svc.SlewToCoordinates(6.0, 45.0)
	time.Sleep(300 * time.Millisecond)

	if count != 0 {
		t.Errorf("unsubscribed callback fired %d times", count)
	}
}

func TestSyncPosition(t *testing.T) {
	svc, stop := setupScripted(t, map[byte]string{'S': "#"})
	defer stop()

	if err := svc.SyncPosition(6.0, 45.0); err != nil {
		t.Fatalf("SyncPosition: %v", err)
	}
}
