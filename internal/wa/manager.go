package wa

import (
	"context"
	"sync"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"

	"wzap/internal/broker"
	"wzap/internal/config"
	"wzap/internal/repo"
	"wzap/internal/webhook"
)

type MediaAutoUploadFunc func(sessionID, messageID, mimeType string, downloadable whatsmeow.DownloadableMessage)
type MessagePersistFunc func(sessionID, messageID, chatJID, senderJID string, fromMe bool, msgType, body, mediaType string, timestamp int64, raw interface{})

type ClientInfo struct {
	PushName     string
	BusinessName string
	Platform     string
}

func (m *Manager) GetClientInfo(sessionID string) *ClientInfo {
	m.mu.RLock()
	client, exists := m.clients[sessionID]
	m.mu.RUnlock()

	if !exists || client.Store == nil {
		return nil
	}

	info := &ClientInfo{
		PushName:     client.Store.PushName,
		BusinessName: client.Store.BusinessName,
		Platform:     client.Store.Platform,
	}

	if info.PushName == "" && info.BusinessName == "" && info.Platform == "" {
		return nil
	}

	return info
}

type Manager struct {
	clients      map[string]*whatsmeow.Client
	sessionNames map[string]string // cache de sessionID -> name
	mu           sync.RWMutex

	ctx               context.Context
	sessionRepo       *repo.SessionRepository
	container         *sqlstore.Container
	nats              *broker.NATS
	dispatcher        *webhook.Dispatcher
	cfg               *config.Config
	waLog             waLog.Logger
	OnMediaReceived   MediaAutoUploadFunc
	OnMessageReceived MessagePersistFunc
}

func (m *Manager) SetMediaAutoUpload(fn MediaAutoUploadFunc) {
	m.OnMediaReceived = fn
}

func (m *Manager) SetMessagePersist(fn MessagePersistFunc) {
	m.OnMessageReceived = fn
}

func (m *Manager) UpdateSessionName(sessionID, name string) {
	m.mu.Lock()
	m.sessionNames[sessionID] = name
	m.mu.Unlock()
}

func (m *Manager) getSessionName(sessionID string) string {
	m.mu.RLock()
	name, ok := m.sessionNames[sessionID]
	m.mu.RUnlock()

	if ok {
		return name
	}

	// Buscar do banco se não estiver em cache
	if m.sessionRepo != nil {
		session, err := m.sessionRepo.FindByID(m.ctx, sessionID)
		if err == nil {
			m.mu.Lock()
			m.sessionNames[sessionID] = session.Name
			m.mu.Unlock()
			return session.Name
		}
	}
	return ""
}
