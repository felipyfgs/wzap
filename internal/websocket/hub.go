package websocket

import (
	"encoding/json"
	"sync"

	"wzap/internal/logger"

	ws "github.com/gofiber/contrib/websocket"
)

type Hub struct {
	mu          sync.RWMutex
	connections map[string]map[*ws.Conn]struct{} // sessionID -> set of connections
}

func NewHub() *Hub {
	return &Hub{
		connections: make(map[string]map[*ws.Conn]struct{}),
	}
}

func (h *Hub) Register(sessionID string, conn *ws.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.connections[sessionID] == nil {
		h.connections[sessionID] = make(map[*ws.Conn]struct{})
	}
	h.connections[sessionID][conn] = struct{}{}
	logger.Debug().Str("component", "ws").Str("session", sessionID).Msg("WebSocket client connected")
}

func (h *Hub) Unregister(sessionID string, conn *ws.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, ok := h.connections[sessionID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.connections, sessionID)
		}
	}
	logger.Debug().Str("component", "ws").Str("session", sessionID).Msg("WebSocket client disconnected")
}

func (h *Hub) Broadcast(sessionID string, payload []byte) {
	h.mu.RLock()
	snapshot := make(map[*ws.Conn]struct{}, len(h.connections[sessionID]))
	for conn := range h.connections[sessionID] {
		snapshot[conn] = struct{}{}
	}
	h.mu.RUnlock()

	for conn := range snapshot {
		if err := conn.WriteMessage(ws.TextMessage, payload); err != nil {
			logger.Warn().Str("component", "ws").Err(err).Str("session", sessionID).Msg("WebSocket write failed, closing connection")
			h.Unregister(sessionID, conn)
			_ = conn.Close()
		}
	}
}

func (h *Hub) BroadcastAll(payload []byte) {
	h.mu.RLock()
	allConns := make(map[*ws.Conn]struct{})
	for _, conns := range h.connections {
		for conn := range conns {
			allConns[conn] = struct{}{}
		}
	}
	h.mu.RUnlock()

	for conn := range allConns {
		if err := conn.WriteMessage(ws.TextMessage, payload); err != nil {
			_ = conn.Close()
		}
	}
}

func (h *Hub) BroadcastJSON(sessionID string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		logger.Error().Str("component", "ws").Err(err).Msg("Failed to marshal WebSocket payload")
		return
	}
	h.Broadcast(sessionID, payload)
}
