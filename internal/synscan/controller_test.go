package synscan

import (
	"testing"
	"time"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/serial"
)

// scriptedMock sets up a MockPort that automatically responds to commands.
type scriptedMock struct {
	*serial.MockPort
	responses []string
	idx       int
}

func newScriptedMock(responses ...string) *scriptedMock {
	return &scriptedMock{
		MockPort:  serial.NewMockPort(),
		responses: responses,
	}
}

func (s *scriptedMock) feedNext() {
	if s.idx < len(s.responses) {
		s.InjectResponse([]byte(s.responses[s.idx]))
		s.idx++
	}
}

func TestEcho(t *testing.T) {
	mock := newScriptedMock()
	_ = mock.Connect("/dev/test", serial.DefaultConfig())
	ctrl := NewController(mock.MockPort)

	go func() {
		time.Sleep(80 * time.Millisecond) // Must exceed drain() timeout (50ms)
		mock.InjectResponse([]byte("x#"))
	}()

	b, err := ctrl.Echo('x')
	if err != nil {
		t.Fatalf("echo: %v", err)
	}
	if b != 'x' {
		t.Fatalf("expected 'x', got %c", b)
	}
}

func TestGetRADec(t *testing.T) {
	mock := serial.NewMockPort()
	_ = mock.Connect("/dev/test", serial.DefaultConfig())
	ctrl := NewController(mock)

	go func() {
		time.Sleep(80 * time.Millisecond)
		// RA=6h (90 deg = 40000000), Dec=+45 (20000000)
		mock.InjectResponse([]byte("40000000,20000000#"))
	}()

	ra, dec, err := ctrl.GetRADec()
	if err != nil {
		t.Fatalf("GetRADec: %v", err)
	}
	if ra < 5.99 || ra > 6.01 {
		t.Errorf("RA: expected ~6h, got %f", ra)
	}
	if dec < 44.99 || dec > 45.01 {
		t.Errorf("Dec: expected ~45, got %f", dec)
	}
}

func TestGetRADecNegativeDec(t *testing.T) {
	mock := serial.NewMockPort()
	_ = mock.Connect("/dev/test", serial.DefaultConfig())
	ctrl := NewController(mock)

	// Encode -16.716 using our encoder for a consistent round-trip.
	decHex := EncodeDegrees(-16.716)

	go func() {
		time.Sleep(80 * time.Millisecond)
		mock.InjectResponse([]byte("40000000," + decHex + "#"))
	}()

	_, dec, err := ctrl.GetRADec()
	if err != nil {
		t.Fatalf("GetRADec: %v", err)
	}
	if dec > -16.5 || dec < -17.0 {
		t.Errorf("Dec: expected ~-16.7, got %f", dec)
	}
}

func TestGetAltAz(t *testing.T) {
	mock := serial.NewMockPort()
	_ = mock.Connect("/dev/test", serial.DefaultConfig())
	ctrl := NewController(mock)

	go func() {
		time.Sleep(80 * time.Millisecond)
		// Alt=45 (20000000), Az=180 (80000000)
		mock.InjectResponse([]byte("20000000,80000000#"))
	}()

	alt, az, err := ctrl.GetAltAz()
	if err != nil {
		t.Fatalf("GetAltAz: %v", err)
	}
	if alt < 44.99 || alt > 45.01 {
		t.Errorf("Alt: expected ~45, got %f", alt)
	}
	if az < 179.99 || az > 180.01 {
		t.Errorf("Az: expected ~180, got %f", az)
	}
}

func TestIsGotoInProgress(t *testing.T) {
	mock := serial.NewMockPort()
	_ = mock.Connect("/dev/test", serial.DefaultConfig())
	ctrl := NewController(mock)

	go func() {
		time.Sleep(80 * time.Millisecond)
		mock.InjectResponse([]byte("0#"))
	}()

	slewing, err := ctrl.IsGotoInProgress()
	if err != nil {
		t.Fatalf("IsGotoInProgress: %v", err)
	}
	if slewing {
		t.Error("expected not slewing")
	}
}

func TestIsAligned(t *testing.T) {
	mock := serial.NewMockPort()
	_ = mock.Connect("/dev/test", serial.DefaultConfig())
	ctrl := NewController(mock)

	go func() {
		time.Sleep(80 * time.Millisecond)
		mock.InjectResponse([]byte{0x01, '#'})
	}()

	aligned, err := ctrl.IsAligned()
	if err != nil {
		t.Fatalf("IsAligned: %v", err)
	}
	if !aligned {
		t.Error("expected aligned")
	}
}

func TestGetTrackingMode(t *testing.T) {
	mock := serial.NewMockPort()
	_ = mock.Connect("/dev/test", serial.DefaultConfig())
	ctrl := NewController(mock)

	go func() {
		time.Sleep(80 * time.Millisecond)
		mock.InjectResponse([]byte{byte(TrackingSidereal), '#'})
	}()

	mode, err := ctrl.GetTrackingMode()
	if err != nil {
		t.Fatalf("GetTrackingMode: %v", err)
	}
	if mode != TrackingSidereal {
		t.Errorf("expected Sidereal, got %v", mode)
	}
}

func TestGotoRADec(t *testing.T) {
	mock := serial.NewMockPort()
	_ = mock.Connect("/dev/test", serial.DefaultConfig())
	ctrl := NewController(mock)

	go func() {
		time.Sleep(80 * time.Millisecond)
		mock.InjectResponse([]byte("#"))
	}()

	err := ctrl.GotoRADec(6.0, 45.0)
	if err != nil {
		t.Fatalf("GotoRADec: %v", err)
	}

	sent := mock.SentData()
	if len(sent) == 0 {
		t.Fatal("no data sent")
	}
	cmd := string(sent[0])
	// Should start with 'r' and contain two 8-char hex values separated by comma.
	if cmd[0] != 'r' || len(cmd) != 18 || cmd[9] != ',' {
		t.Errorf("unexpected command format: %q", cmd)
	}
}

func TestCancelGoto(t *testing.T) {
	mock := serial.NewMockPort()
	_ = mock.Connect("/dev/test", serial.DefaultConfig())
	ctrl := NewController(mock)

	go func() {
		time.Sleep(80 * time.Millisecond)
		mock.InjectResponse([]byte("#"))
	}()

	err := ctrl.CancelGoto()
	if err != nil {
		t.Fatalf("CancelGoto: %v", err)
	}

	sent := mock.SentData()
	if len(sent) == 0 || string(sent[0]) != "M" {
		t.Errorf("expected 'M' command, got %v", sent)
	}
}

func TestSetTrackingMode(t *testing.T) {
	mock := serial.NewMockPort()
	_ = mock.Connect("/dev/test", serial.DefaultConfig())
	ctrl := NewController(mock)

	go func() {
		time.Sleep(80 * time.Millisecond)
		mock.InjectResponse([]byte("#"))
	}()

	err := ctrl.SetTrackingMode(TrackingSidereal)
	if err != nil {
		t.Fatalf("SetTrackingMode: %v", err)
	}
}

func TestSyncRADec(t *testing.T) {
	mock := serial.NewMockPort()
	_ = mock.Connect("/dev/test", serial.DefaultConfig())
	ctrl := NewController(mock)

	go func() {
		time.Sleep(80 * time.Millisecond)
		mock.InjectResponse([]byte("#"))
	}()

	err := ctrl.SyncRADec(6.0, 45.0)
	if err != nil {
		t.Fatalf("SyncRADec: %v", err)
	}

	sent := mock.SentData()
	if len(sent) == 0 || sent[0][0] != 'S' {
		t.Errorf("expected 'S' command, got %v", sent)
	}
}

func TestNotConnected(t *testing.T) {
	mock := serial.NewMockPort()
	ctrl := NewController(mock)

	_, err := ctrl.Echo('x')
	if err != ErrNotConnected {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
}

func TestTimeout(t *testing.T) {
	mock := serial.NewMockPort()
	_ = mock.Connect("/dev/test", serial.DefaultConfig())
	ctrl := NewController(mock)
	ctrl.timeout = 100 * time.Millisecond // Speed up for test.

	_, err := ctrl.Echo('x')
	if err != ErrTimeout {
		t.Fatalf("expected ErrTimeout, got %v", err)
	}
}

func TestMalformedResponse(t *testing.T) {
	mock := serial.NewMockPort()
	_ = mock.Connect("/dev/test", serial.DefaultConfig())
	ctrl := NewController(mock)

	go func() {
		time.Sleep(80 * time.Millisecond)
		mock.InjectResponse([]byte("SHORT#"))
	}()

	_, _, err := ctrl.GetRADec()
	if err == nil {
		t.Fatal("expected error for short response")
	}
}

func TestGetModel(t *testing.T) {
	mock := serial.NewMockPort()
	_ = mock.Connect("/dev/test", serial.DefaultConfig())
	ctrl := NewController(mock)

	go func() {
		time.Sleep(80 * time.Millisecond)
		mock.InjectResponse([]byte("\x14#")) // Model byte 0x14
	}()

	model, err := ctrl.GetModel()
	if err != nil {
		t.Fatalf("GetModel: %v", err)
	}
	if len(model) != 1 || model[0] != 0x14 {
		t.Errorf("expected model byte 0x14, got %q", model)
	}
}

func TestChunkedResponse(t *testing.T) {
	mock := serial.NewMockPort()
	_ = mock.Connect("/dev/test", serial.DefaultConfig())
	ctrl := NewController(mock)

	// Simulate response arriving in two chunks.
	go func() {
		time.Sleep(80 * time.Millisecond)
		mock.InjectResponse([]byte("40000000,"))
		time.Sleep(80 * time.Millisecond)
		mock.InjectResponse([]byte("20000000#"))
	}()

	ra, dec, err := ctrl.GetRADec()
	if err != nil {
		t.Fatalf("chunked GetRADec: %v", err)
	}
	if ra < 5.99 || ra > 6.01 {
		t.Errorf("RA: expected ~6h, got %f", ra)
	}
	if dec < 44.99 || dec > 45.01 {
		t.Errorf("Dec: expected ~45, got %f", dec)
	}
}
