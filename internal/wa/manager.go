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

type Manager struct {
	clients map[string]*whatsmeow.Client
	mu      sync.RWMutex

	ctx                context.Context
	sessionRepo        *repo.SessionRepository
	container          *sqlstore.Container
	nats               *broker.Nats
	dispatcher         *webhook.Dispatcher
	cfg                *config.Config
	waLog              waLog.Logger
	OnMediaReceived    MediaAutoUploadFunc
	OnMessageReceived  MessagePersistFunc
}

func (m *Manager) SetMediaAutoUpload(fn MediaAutoUploadFunc) {
	m.OnMediaReceived = fn
}

func (m *Manager) SetMessagePersist(fn MessagePersistFunc) {
	m.OnMessageReceived = fn
}
