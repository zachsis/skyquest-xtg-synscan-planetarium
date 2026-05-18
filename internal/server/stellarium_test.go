package server

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/slew"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/telescope"
)

// --- test doubles ---

type fakePositionProvider struct {
	mu        sync.RWMutex
	state     telescope.TelescopeState
	callbacks []func(telescope.TelescopeState)
}

func newFakeProvider(ra, dec float64) *fakePositionProvider {
	return &fakePositionProvider{
		state: telescope.TelescopeState{RA: ra, Dec: dec, Connected: true},
	}
}

func (f *fakePositionProvider) CurrentState() telescope.TelescopeState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state
}

func (f *fakePositionProvider) Subscribe(
	cb func(telescope.TelescopeState),
) func() {
	f.mu.Lock()
	idx := len(f.callbacks)
	f.callbacks = append(f.callbacks, cb)
	f.mu.Unlock()
	return func() {
		f.mu.Lock()
		f.callbacks[idx] = nil
		f.mu.Unlock()
	}
}

func (f *fakePositionProvider) Publish(ra, dec float64) {
	st := telescope.TelescopeState{RA: ra, Dec: dec, Connected: true}
	f.mu.Lock()
	f.state = st
	cbs := make([]func(telescope.TelescopeState), len(f.callbacks))
	copy(cbs, f.callbacks)
	f.mu.Unlock()
	for _, cb := range cbs {
		if cb != nil {
			cb(st)
		}
	}
}

// fakeSlewService implements slew.GoToService for testing.
type fakeSlewService struct {
	mu      sync.Mutex
	lastRA  float64
	lastDec float64
	calls   int
}

var _ slew.GoToService = (*fakeSlewService)(nil)

func (f *fakeSlewService) SlewToCoordinates(ra, dec float64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastRA = ra
	f.lastDec = dec
	f.calls++
	return nil
}

func (f *fakeSlewService) SlewToObject(
	name, catalogID string, ra, dec float64,
) error {
	return f.SlewToCoordinates(ra, dec)
}

func (f *fakeSlewService) SyncPosition(ra, dec float64) error { return nil }
func (f *fakeSlewService) CancelSlew() error                  { return nil }

func (f *fakeSlewService) SlewState() slew.SlewState { return slew.SlewIdle }

func (f *fakeSlewService) OnSlew(cb func(slew.SlewEvent)) func() {
	return func() {}
}

// --- helpers ---

// freePort finds a free TCP port on localhost.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

// dialRetry retries dialing until success or timeout.
func dialRetry(t *testing.T, addr string) net.Conn {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			return conn
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("dialRetry: could not connect to %s", addr)
	return nil
}

// sendGoto writes a 20-byte GoTo message to conn.
func sendGoto(t *testing.T, conn net.Conn, ra, dec float64) {
	t.Helper()
	buf := make([]byte, 20)
	binary.LittleEndian.PutUint16(buf[0:2], 20)
	binary.LittleEndian.PutUint16(buf[2:4], 0)
	binary.LittleEndian.PutUint64(buf[4:12], 0)
	binary.LittleEndian.PutUint32(buf[12:16], encodeRA(ra))
	binary.LittleEndian.PutUint32(buf[16:20], uint32(encodeDec(dec)))
	conn.SetWriteDeadline(time.Now().Add(time.Second))
	if _, err := conn.Write(buf); err != nil {
		t.Fatalf("sendGoto write: %v", err)
	}
}

// readCurrentPosition reads a 24-byte position message from conn.
func readCurrentPosition(t *testing.T, conn net.Conn) (ra, dec float64) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, sizeCurrentPosition)
	if _, err := readFull(conn, buf); err != nil {
		t.Fatalf("readCurrentPosition: %v", err)
	}
	msgLen := binary.LittleEndian.Uint16(buf[0:2])
	if msgLen != sizeCurrentPosition {
		t.Fatalf("expected length %d, got %d", sizeCurrentPosition, msgLen)
	}
	rawRA := binary.LittleEndian.Uint32(buf[12:16])
	rawDec := int32(binary.LittleEndian.Uint32(buf[16:20]))
	return decodeRA(rawRA), decodeDec(rawDec)
}

// readFull reads exactly len(buf) bytes from conn.
func readFull(conn net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := conn.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// --- tests ---

func TestStellariumServer_StartStop(t *testing.T) {
	port := freePort(t)
	srv := NewStellariumServer(port, 100*time.Millisecond, nil)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !srv.IsRunning() {
		t.Error("expected server to be running after Start")
	}
	srv.Stop()
	if srv.IsRunning() {
		t.Error("expected server to be stopped after Stop")
	}
}

func TestStellariumServer_DoubleStartNoOp(t *testing.T) {
	port := freePort(t)
	srv := NewStellariumServer(port, 100*time.Millisecond, nil)
	defer srv.Stop()
	if err := srv.Start(); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	// Second Start must be a no-op (no error, no panic).
	if err := srv.Start(); err != nil {
		t.Fatalf("second Start: %v", err)
	}
}

func TestStellariumServer_InvalidPort(t *testing.T) {
	srv := NewStellariumServer(80, 100*time.Millisecond, nil)
	err := srv.Start()
	if err == nil {
		srv.Stop()
		t.Fatal("expected error for privileged port 80, got nil")
	}
}

func TestStellariumServer_BroadcastsPosition(t *testing.T) {
	port := freePort(t)
	prov := newFakeProvider(6.0, 30.0)
	srv := NewStellariumServer(port, 100*time.Millisecond, nil)
	srv.SetPositionProvider(prov)

	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	conn := dialRetry(t, addr)
	defer conn.Close()

	ra, dec := readCurrentPosition(t, conn)
	if math.Abs(ra-6.0) > 0.01 {
		t.Errorf("RA: want ~6.0, got %.4f", ra)
	}
	if math.Abs(dec-30.0) > 0.01 {
		t.Errorf("Dec: want ~30.0, got %.4f", dec)
	}
}

func TestStellariumServer_PositionUpdates(t *testing.T) {
	port := freePort(t)
	prov := newFakeProvider(0.0, 0.0)
	srv := NewStellariumServer(port, 50*time.Millisecond, nil)
	srv.SetPositionProvider(prov)

	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()

	// Update position.
	prov.Publish(18.5, -45.0)

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	conn := dialRetry(t, addr)
	defer conn.Close()

	// Read several broadcasts and check that the updated value eventually appears.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		ra, dec := readCurrentPosition(t, conn)
		if math.Abs(ra-18.5) < 0.01 && math.Abs(dec+45.0) < 0.01 {
			return // success
		}
	}
	t.Error("never received updated RA/Dec from position provider")
}

func TestStellariumServer_GotoForwardedToSlew(t *testing.T) {
	port := freePort(t)
	fakeSvc := &fakeSlewService{}
	srv := NewStellariumServer(port, 200*time.Millisecond, fakeSvc)

	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	conn := dialRetry(t, addr)
	defer conn.Close()

	sendGoto(t, conn, 10.5, 22.3)

	// Wait for the slew service to receive the command.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		fakeSvc.mu.Lock()
		calls := fakeSvc.calls
		fakeSvc.mu.Unlock()
		if calls > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	fakeSvc.mu.Lock()
	gotRA := fakeSvc.lastRA
	gotDec := fakeSvc.lastDec
	fakeSvc.mu.Unlock()

	if math.Abs(gotRA-10.5) > 0.01 {
		t.Errorf("slew RA: want ~10.5, got %.4f", gotRA)
	}
	if math.Abs(gotDec-22.3) > 0.01 {
		t.Errorf("slew Dec: want ~22.3, got %.4f", gotDec)
	}
}

func TestStellariumServer_ClientCountCallback(t *testing.T) {
	port := freePort(t)
	srv := NewStellariumServer(port, 500*time.Millisecond, nil)

	var (
		lastCount int32
	)
	srv.SetOnClientChange(func(n int) {
		atomic.StoreInt32(&lastCount, int32(n))
	})

	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()

	addr := fmt.Sprintf("127.0.0.1:%d", port)

	conn1 := dialRetry(t, addr)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&lastCount) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt32(&lastCount) != 1 {
		t.Errorf(
			"after 1 connect: want count=1, got %d",
			atomic.LoadInt32(&lastCount),
		)
	}

	conn2 := dialRetry(t, addr)
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&lastCount) == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt32(&lastCount) != 2 {
		t.Errorf(
			"after 2 connects: want count=2, got %d",
			atomic.LoadInt32(&lastCount),
		)
	}

	conn1.Close()
	conn2.Close()
	// Server detects disconnect on next read timeout (~2s).
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&lastCount) == 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if atomic.LoadInt32(&lastCount) != 0 {
		t.Errorf(
			"after both disconnect: want count=0, got %d",
			atomic.LoadInt32(&lastCount),
		)
	}
}

func TestStellariumServer_MaxClientsRejected(t *testing.T) {
	port := freePort(t)
	srv := NewStellariumServer(port, 500*time.Millisecond, nil)

	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()

	addr := fmt.Sprintf("127.0.0.1:%d", port)

	conns := make([]net.Conn, maxClients)
	for i := 0; i < maxClients; i++ {
		conns[i] = dialRetry(t, addr)
	}
	defer func() {
		for _, c := range conns {
			if c != nil {
				c.Close()
			}
		}
	}()

	// Wait for all to be registered.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if srv.ClientCount() == maxClients {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Connect one more; the server should close it immediately.
	extra, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("extra dial: %v", err)
	}
	defer extra.Close()

	// The extra connection should be closed by the server quickly.
	extra.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 1)
	_, readErr := extra.Read(buf)
	if readErr == nil {
		t.Error("extra client: expected connection to be closed by server")
	}
}
