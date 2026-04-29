package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"wzap/internal/logger"

	ws "github.com/gofiber/contrib/websocket"
)

// Hub fan-outs server-side events to every WebSocket peer subscribed to a
// session. The previous implementation called `conn.WriteMessage` directly
// from each broadcaster goroutine — fasthttp/websocket is **not** safe for
// concurrent writes, so this caused both data races and frame corruption
// under load. Each connection now has a dedicated writer goroutine that owns
// the socket end-to-end; broadcasters simply enqueue payloads.
//
// Slow consumers are evicted instead of stalling the whole fan-out:
//   - the per-conn send channel is buffered (sendBuffer)
//   - if the buffer is full when we try to enqueue, the conn is dropped
//
// The writer also runs the application-level ping ticker so idle sockets
// don't get reaped by NAT/proxies and the read pump can detect dead peers
// via the read deadline.

const (
	writeTimeout = 10 * time.Second
	pongTimeout  = 60 * time.Second
	pingInterval = (pongTimeout * 9) / 10
	sendBuffer   = 64
)

// peer wraps a connection with its dedicated outbound channel. Writes are
// serialised through the writer goroutine — no other code path touches the
// socket directly for output.
type peer struct {
	conn      *ws.Conn
	send      chan []byte
	closeOnce sync.Once
	done      chan struct{}
}

func newPeer(conn *ws.Conn) *peer {
	return &peer{
		conn: conn,
		send: make(chan []byte, sendBuffer),
		done: make(chan struct{}),
	}
}

// closeSend signals the writer goroutine to terminate. Safe to call multiple
// times. The channel is closed exactly once so writers see io.EOF on send.
func (p *peer) closeSend() {
	p.closeOnce.Do(func() {
		close(p.done)
	})
}

type Hub struct {
	mu          sync.RWMutex
	connections map[string]map[*peer]struct{} // sessionID → peers
}

func NewHub() *Hub {
	return &Hub{
		connections: make(map[string]map[*peer]struct{}),
	}
}

// Register attaches `conn` to `sessionID` and returns a function the caller
// MUST invoke when the read loop exits — typically via `defer`. The function
// drains the writer, removes the peer from the routing table and closes the
// underlying connection.
//
// The caller is also expected to drive the read pump (this hub does not own
// the read side because the read loop is what the upgrader hands to fiber).
// Use `Configure` on the conn beforehand to wire the read deadline + pong
// handler.
func (h *Hub) Register(sessionID string, conn *ws.Conn) func() {
	p := newPeer(conn)

	h.mu.Lock()
	if h.connections[sessionID] == nil {
		h.connections[sessionID] = make(map[*peer]struct{})
	}
	h.connections[sessionID][p] = struct{}{}
	h.mu.Unlock()

	go h.writePump(sessionID, p)

	logger.Debug().Str("component", "ws").Str("session", sessionID).Msg("WebSocket client connected")

	return func() {
		h.removePeer(sessionID, p)
	}
}

// Configure applies the read deadline + pong handler to a freshly-upgraded
// conn. Call this BEFORE entering the read loop. Without it, idle peers are
// not reaped and the read deadline never advances.
func Configure(conn *ws.Conn) {
	_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
		return nil
	})
}

func (h *Hub) removePeer(sessionID string, p *peer) {
	h.mu.Lock()
	if conns, ok := h.connections[sessionID]; ok {
		if _, exists := conns[p]; exists {
			delete(conns, p)
			if len(conns) == 0 {
				delete(h.connections, sessionID)
			}
		}
	}
	h.mu.Unlock()
	p.closeSend()
	_ = p.conn.Close()
	logger.Debug().Str("component", "ws").Str("session", sessionID).Msg("WebSocket client disconnected")
}

// Broadcast enqueues `payload` for delivery to every peer currently joined to
// `sessionID`. Slow peers (full send buffer) are evicted to keep the hub
// responsive — a healthy client will always have an empty buffer because the
// writer pumps it as fast as the network allows.
func (h *Hub) Broadcast(sessionID string, payload []byte) {
	h.mu.RLock()
	peers := make([]*peer, 0, len(h.connections[sessionID]))
	for p := range h.connections[sessionID] {
		peers = append(peers, p)
	}
	h.mu.RUnlock()

	for _, p := range peers {
		h.deliver(sessionID, p, payload)
	}
}

// BroadcastAll enqueues `payload` to every connected peer across all sessions.
// Used for system-wide announcements; routine traffic should target a session.
func (h *Hub) BroadcastAll(payload []byte) {
	h.mu.RLock()
	type entry struct {
		sessionID string
		peer      *peer
	}
	peers := make([]entry, 0)
	for sessionID, conns := range h.connections {
		for p := range conns {
			peers = append(peers, entry{sessionID: sessionID, peer: p})
		}
	}
	h.mu.RUnlock()

	for _, e := range peers {
		h.deliver(e.sessionID, e.peer, payload)
	}
}

// deliver tries to enqueue without blocking. If the send buffer is full the
// peer is too slow to keep up — drop it so the next broadcast doesn't pile up
// on the same dead-end conn. Eviction is async (via removePeer in a goroutine)
// because we still hold no locks here and want the caller to keep moving.
func (h *Hub) deliver(sessionID string, p *peer, payload []byte) {
	select {
	case p.send <- payload:
	case <-p.done:
		// peer is already shutting down — drop silently.
	default:
		logger.Warn().Str("component", "ws").Str("session", sessionID).Msg("WebSocket peer slow, dropping connection")
		go h.removePeer(sessionID, p)
	}
}

// BroadcastJSON marshals `data` once and fans it out to all peers in the
// session. Marshal errors are logged but never bubble — realtime is
// best-effort and mustn't block the caller.
func (h *Hub) BroadcastJSON(sessionID string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		logger.Error().Str("component", "ws").Err(err).Msg("Failed to marshal WebSocket payload")
		return
	}
	h.Broadcast(sessionID, payload)
}

// writePump owns the socket on the write side. It serialises:
//   - text frames coming from the broadcast queue
//   - keepalive pings on a fixed interval
//
// Exits when the peer is closed (channel done) or any write fails. The deferred
// removePeer guarantees the routing table is cleaned up even if the goroutine
// panics or the conn dies between iterations.
func (h *Hub) writePump(sessionID string, p *peer) {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		h.removePeer(sessionID, p)
	}()

	for {
		select {
		case payload := <-p.send:
			_ = p.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := p.conn.WriteMessage(ws.TextMessage, payload); err != nil {
				logger.Warn().Str("component", "ws").Err(err).Str("session", sessionID).Msg("WebSocket write failed")
				return
			}
		case <-ticker.C:
			_ = p.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := p.conn.WriteMessage(ws.PingMessage, nil); err != nil {
				logger.Debug().Str("component", "ws").Err(err).Str("session", sessionID).Msg("WebSocket ping failed")
				return
			}
		case <-p.done:
			return
		}
	}
}
