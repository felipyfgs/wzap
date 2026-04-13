package wa

import (
	"context"
	"fmt"
	"sync"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	"wzap/internal/broker"
	"wzap/internal/config"
	"wzap/internal/repo"
	"wzap/internal/webhook"
)

type MediaAutoUploadFunc func(sessionID, messageID, mimeType string, downloadable whatsmeow.DownloadableMessage)
type MessagePersistFunc func(sessionID, messageID, chatJID, senderJID string, fromMe bool, msgType, body, mediaType string, timestamp int64, raw interface{})
type HistorySyncPersistFunc func(sessionID string, sync *events.HistorySync)

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

	sessionRepo           *repo.SessionRepository
	container             *sqlstore.Container
	nats                  *broker.NATS
	dispatcher            *webhook.Dispatcher
	cfg                   *config.Config
	waLog                 waLog.Logger
	OnMediaReceived       MediaAutoUploadFunc
	OnMessageReceived     MessagePersistFunc
	OnHistorySyncReceived HistorySyncPersistFunc
}

func (m *Manager) SetMediaAutoUpload(fn MediaAutoUploadFunc) {
	m.OnMediaReceived = fn
}

func (m *Manager) SetMessagePersist(fn MessagePersistFunc) {
	m.OnMessageReceived = fn
}

func (m *Manager) SetHistorySyncPersist(fn HistorySyncPersistFunc) {
	m.OnHistorySyncReceived = fn
}

func (m *Manager) GetPNForLID(ctx context.Context, sessionID, lidJID string) string {
	m.mu.RLock()
	client, exists := m.clients[sessionID]
	m.mu.RUnlock()
	if !exists || client.Store == nil {
		return ""
	}
	lid, err := types.ParseJID(lidJID)
	if err != nil {
		return ""
	}
	pn, err := client.Store.LIDs.GetPNForLID(ctx, lid)
	if err != nil || pn.IsEmpty() {
		return ""
	}
	return pn.User
}

func (m *Manager) UpdateSessionName(sessionID, name string) {
	m.mu.Lock()
	m.sessionNames[sessionID] = name
	m.mu.Unlock()
}

func (m *Manager) getSessionName(ctx context.Context, sessionID string) string {
	m.mu.RLock()
	name, ok := m.sessionNames[sessionID]
	m.mu.RUnlock()

	if ok {
		return name
	}

	// Buscar do banco se não estiver em cache
	if m.sessionRepo != nil {
		session, err := m.sessionRepo.FindByID(ctx, sessionID)
		if err == nil {
			m.mu.Lock()
			m.sessionNames[sessionID] = session.Name
			m.mu.Unlock()
			return session.Name
		}
	}
	return ""
}

func (m *Manager) DownloadMediaByPath(ctx context.Context, sessionID, directPath string, encFileHash, fileHash, mediaKey []byte, fileLength int, mediaType string) ([]byte, error) {
	client, err := m.GetClient(sessionID)
	if err != nil {
		return nil, err
	}

	var wmMediaType whatsmeow.MediaType
	switch mediaType {
	case "image", "sticker":
		wmMediaType = whatsmeow.MediaImage
	case "audio":
		wmMediaType = whatsmeow.MediaAudio
	case "video":
		wmMediaType = whatsmeow.MediaVideo
	case "document":
		wmMediaType = whatsmeow.MediaDocument
	default:
		return nil, fmt.Errorf("unknown media type: %s", mediaType)
	}

	return client.DownloadMediaWithPath(ctx, directPath, encFileHash, fileHash, mediaKey, fileLength, wmMediaType, "")
}

func (m *Manager) GetProfilePicture(ctx context.Context, sessionID, jid string) (string, error) {
	client, err := m.GetClient(sessionID)
	if err != nil {
		return "", err
	}

	parsedJID, err := types.ParseJID(jid)
	if err != nil {
		return "", fmt.Errorf("failed to parse JID: %w", err)
	}

	pic, err := client.GetProfilePictureInfo(ctx, parsedJID, &whatsmeow.GetProfilePictureParams{Preview: false})
	if err != nil {
		return "", fmt.Errorf("failed to get profile picture: %w", err)
	}
	if pic == nil {
		return "", nil
	}
	return pic.URL, nil
}

func (m *Manager) IsOnWhatsApp(ctx context.Context, sessionID string, phones []string) (map[string]string, error) {
	client, err := m.GetClient(sessionID)
	if err != nil {
		return nil, err
	}
	if !client.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	resp, err := client.IsOnWhatsApp(ctx, phones)
	if err != nil {
		return nil, fmt.Errorf("failed to check numbers on WhatsApp: %w", err)
	}

	result := make(map[string]string, len(resp))
	for _, r := range resp {
		if r.IsIn {
			result[r.Query] = r.JID.User + "@s.whatsapp.net"
		}
	}
	return result, nil
}
