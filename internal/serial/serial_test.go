package serial

import (
	"sync"
	"testing"
	"time"
)

func TestMockPortConnect(t *testing.T) {
	p := NewMockPort()
	if p.IsConnected() {
		t.Fatal("expected disconnected initially")
	}

	err := p.Connect("/dev/test", DefaultConfig())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if !p.IsConnected() {
		t.Fatal("expected connected after Connect")
	}
	if p.LastConnectedPort() != "/dev/test" {
		t.Fatalf("expected /dev/test, got %s", p.LastConnectedPort())
	}
}

func TestMockPortAlreadyConnected(t *testing.T) {
	p := NewMockPort()
	_ = p.Connect("/dev/test", DefaultConfig())
	err := p.Connect("/dev/test2", DefaultConfig())
	if err != ErrAlreadyConnected {
		t.Fatalf("expected ErrAlreadyConnected, got %v", err)
	}
}

func TestMockPortSendDisconnected(t *testing.T) {
	p := NewMockPort()
	err := p.Send([]byte("hello"))
	if err != ErrNotConnected {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
}

func TestMockPortSendReceive(t *testing.T) {
	p := NewMockPort()
	_ = p.Connect("/dev/test", DefaultConfig())

	if err := p.Send([]byte("cmd")); err != nil {
		t.Fatalf("send: %v", err)
	}

	sent := p.SentData()
	if len(sent) != 1 || string(sent[0]) != "cmd" {
		t.Fatalf("expected [cmd], got %v", sent)
	}

	go p.InjectResponse([]byte("resp#"))

	select {
	case data := <-p.Receive():
		if string(data) != "resp#" {
			t.Fatalf("expected resp#, got %s", string(data))
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for response")
	}
}

func TestMockPortDisconnect(t *testing.T) {
	p := NewMockPort()
	_ = p.Connect("/dev/test", DefaultConfig())
	_ = p.Disconnect()
	if p.IsConnected() {
		t.Fatal("expected disconnected after Disconnect")
	}
	// LastConnectedPort should persist after disconnect.
	if p.LastConnectedPort() != "/dev/test" {
		t.Fatalf("expected /dev/test, got %s", p.LastConnectedPort())
	}
}

func TestMockPortStatusCallback(t *testing.T) {
	p := NewMockPort()

	var mu sync.Mutex
	var states []bool
	p.OnStatusChange(func(connected bool) {
		mu.Lock()
		defer mu.Unlock()
		states = append(states, connected)
	})

	_ = p.Connect("/dev/test", DefaultConfig())
	_ = p.Disconnect()

	// Give callbacks a moment to fire.
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(states) != 2 || states[0] != true || states[1] != false {
		t.Fatalf("expected [true, false], got %v", states)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.BaudRate != 9600 {
		t.Fatalf("expected 9600, got %d", cfg.BaudRate)
	}
	if cfg.DataBits != 8 {
		t.Fatalf("expected 8, got %d", cfg.DataBits)
	}
	if cfg.Parity != NoParity {
		t.Fatalf("expected NoParity, got %d", cfg.Parity)
	}
	if cfg.StopBits != OneStopBit {
		t.Fatalf("expected OneStopBit, got %d", cfg.StopBits)
	}
	if cfg.ReadTimeout != 100*time.Millisecond {
		t.Fatalf("expected 100ms, got %v", cfg.ReadTimeout)
	}
}

func TestPortInterfaceCompliance(t *testing.T) {
	// Verify both types satisfy PortInterface at compile time.
	var _ PortInterface = (*Port)(nil)
	var _ PortInterface = (*MockPort)(nil)
}
