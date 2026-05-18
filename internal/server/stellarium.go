package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/slew"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/telescope"
)

// client holds a single TCP connection and its outbound message channel.
type client struct {
	conn  net.Conn
	outCh chan []byte // buffered capacity 2; drop if full (non-blocking broadcast)
}

// StellariumServer is a TCP server that speaks the Stellarium Telescope Protocol.
// It broadcasts the current telescope position to all connected clients and
// accepts GoTo commands from them.
type StellariumServer struct {
	port     int
	interval time.Duration

	slewSvc  slew.GoToService
	posProv  telescope.PositionProvider
	unsub    func() // unsubscribe from position updates

	listener net.Listener
	clients  []*client
	mu       sync.Mutex

	cancel context.CancelFunc
	wg     sync.WaitGroup

	posmu     sync.RWMutex
	lastRA    float64
	lastDec   float64
	connected bool // whether the telescope itself is connected

	onClientChange func(count int) // optional UI callback
}

// NewStellariumServer creates a StellariumServer. Call Start to bind and serve.
func NewStellariumServer(
	port int,
	interval time.Duration,
	slewSvc slew.GoToService,
) *StellariumServer {
	return &StellariumServer{
		port:     port,
		interval: interval,
		slewSvc:  slewSvc,
	}
}

// SetPositionProvider subscribes the server to telescope state updates.
// Call this before Start.
func (s *StellariumServer) SetPositionProvider(pp telescope.PositionProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.unsub != nil {
		s.unsub()
		s.unsub = nil
	}
	if pp == nil {
		s.posProv = nil
		return
	}
	s.posProv = pp

	// Seed the last known position from the current state.
	st := pp.CurrentState()
	s.posmu.Lock()
	s.lastRA = st.RA
	s.lastDec = st.Dec
	s.connected = st.Connected
	s.posmu.Unlock()

	s.unsub = pp.Subscribe(func(st telescope.TelescopeState) {
		s.posmu.Lock()
		s.lastRA = st.RA
		s.lastDec = st.Dec
		s.connected = st.Connected
		s.posmu.Unlock()
	})
}

// SetOnClientChange sets a callback that is invoked (on an internal goroutine)
// whenever the number of connected clients changes. Safe to call before Start.
func (s *StellariumServer) SetOnClientChange(fn func(count int)) {
	s.mu.Lock()
	s.onClientChange = fn
	s.mu.Unlock()
}

// IsRunning reports whether the server is currently listening.
func (s *StellariumServer) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listener != nil
}

// ClientCount returns the number of currently connected clients.
func (s *StellariumServer) ClientCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.clients)
}

// Start binds the TCP listener and begins accepting connections.
// It is a no-op if the server is already running.
func (s *StellariumServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener != nil {
		return nil // already running
	}
	if s.port < 1024 || s.port > 65535 {
		return fmt.Errorf(
			"stellarium: port %d out of valid range [1024, 65535]", s.port,
		)
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("stellarium: listen on :%d: %w", s.port, err)
	}
	s.listener = ln

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	s.wg.Add(2)
	go s.acceptLoop(ctx, ln)
	go s.broadcastLoop(ctx)

	log.Printf("stellarium: listening on :%d", s.port)
	return nil
}

// Stop closes the listener and disconnects all clients gracefully.
// It blocks until all goroutines have exited.
func (s *StellariumServer) Stop() {
	s.mu.Lock()
	if s.listener == nil {
		s.mu.Unlock()
		return
	}
	cancel := s.cancel
	ln := s.listener
	s.listener = nil
	s.cancel = nil
	s.mu.Unlock()

	cancel()
	ln.Close()
	s.wg.Wait()

	// Close and drain any remaining client connections.
	s.mu.Lock()
	for _, c := range s.clients {
		c.conn.Close()
	}
	s.clients = nil
	s.mu.Unlock()

	log.Printf("stellarium: stopped")
}

// Port returns the configured port.
func (s *StellariumServer) Port() int { return s.port }

// acceptLoop waits for new TCP connections and spawns a read goroutine for each.
func (s *StellariumServer) acceptLoop(ctx context.Context, ln net.Listener) {
	defer s.wg.Done()
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("stellarium: accept error: %v", err)
				return
			}
		}

		s.mu.Lock()
		clientCount := len(s.clients)
		if clientCount >= maxClients {
			s.mu.Unlock()
			log.Printf(
				"stellarium: max clients (%d) reached, rejecting %s",
				maxClients, conn.RemoteAddr(),
			)
			conn.Close()
			continue
		}
		c := &client{
			conn:  conn,
			outCh: make(chan []byte, 2),
		}
		s.clients = append(s.clients, c)
		newCount := len(s.clients)
		cb := s.onClientChange
		s.mu.Unlock()

		log.Printf(
			"stellarium: client connected: %s (%d total)",
			conn.RemoteAddr(), newCount,
		)
		if cb != nil {
			go cb(newCount)
		}

		s.wg.Add(2)
		go s.readClient(ctx, c)
		go s.writeClient(ctx, c)
	}
}

// broadcastLoop sends current position to all clients at the configured interval.
func (s *StellariumServer) broadcastLoop(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.posmu.RLock()
			ra := s.lastRA
			dec := s.lastDec
			s.posmu.RUnlock()

			buf := marshalCurrentPosition(ra, dec)

			s.mu.Lock()
			for _, c := range s.clients {
				// Non-blocking: drop the message if the channel is full.
				select {
				case c.outCh <- buf:
				default:
				}
			}
			s.mu.Unlock()
		}
	}
}

// readClient reads GoTo commands from a single client connection.
func (s *StellariumServer) readClient(ctx context.Context, c *client) {
	defer s.wg.Done()
	defer s.removeClient(c)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Set a read deadline so we can check ctx periodically.
		c.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		ra, dec, err := readGoto(c.conn)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // check ctx and retry
			}
			// Connection closed or protocol error — remove client.
			return
		}

		if s.slewSvc != nil {
			if err := s.slewSvc.SlewToCoordinates(ra, dec); err != nil {
				log.Printf(
					"stellarium: slew to RA=%.4f Dec=%.4f failed: %v",
					ra, dec, err,
				)
			} else {
				log.Printf(
					"stellarium: goto RA=%.4f Dec=%.4f from %s",
					ra, dec, c.conn.RemoteAddr(),
				)
			}
		}
	}
}

// writeClient drains the outbound channel and writes to the connection.
func (s *StellariumServer) writeClient(ctx context.Context, c *client) {
	defer s.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case buf, ok := <-c.outCh:
			if !ok {
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			if _, err := c.conn.Write(buf); err != nil {
				return
			}
		}
	}
}

// removeClient removes c from the client list and fires the callback.
func (s *StellariumServer) removeClient(c *client) {
	c.conn.Close()

	s.mu.Lock()
	newClients := make([]*client, 0, len(s.clients))
	for _, existing := range s.clients {
		if existing != c {
			newClients = append(newClients, existing)
		}
	}
	s.clients = newClients
	count := len(s.clients)
	cb := s.onClientChange
	s.mu.Unlock()

	log.Printf(
		"stellarium: client disconnected: %s (%d remaining)",
		c.conn.RemoteAddr(), count,
	)
	if cb != nil {
		go cb(count)
	}
}
