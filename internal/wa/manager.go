package wa

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

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

type MediaAutoUploadFunc func(sessionID, messageID, chatJID, senderJID, mimeType string, fromMe bool, timestamp time.Time, downloadable whatsmeow.DownloadableMessage)
type MediaRetryFunc func(sessionID, messageID, chatJID, senderJID string, fromMe bool, mimeType string, timestamp time.Time, directPath string, encFileHash, fileHash, mediaKey []byte, fileLength int)
type MessagePersistFunc func(sessionID, messageID, chatJID, senderJID string, fromMe bool, msgType, body, mediaType string, timestamp int64, raw any)
type HistorySyncPersistFunc func(sessionID string, sync *events.HistorySync)
type StatusReceivedFunc func(sessionID, messageID, chatJID, senderJID string, fromMe bool, msgType, body, mediaType string, timestamp int64, raw any)
type ShouldIgnoreStatusFunc func(sessionID string) bool

type mediaRetryCacheEntry struct {
	sessionID   string
	chatJID     string
	senderJID   string
	fromMe      bool
	mimeType    string
	timestamp   time.Time
	encFileHash []byte
	fileHash    []byte
	mediaKey    []byte
	fileLength  int
	expiresAt   time.Time
}

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
	OnMediaRetry          MediaRetryFunc
	OnMessageReceived     MessagePersistFunc
	OnHistorySyncReceived HistorySyncPersistFunc
	OnStatusReceived      StatusReceivedFunc
	OnStatusMediaReceived MediaAutoUploadFunc
	ShouldIgnoreStatus    ShouldIgnoreStatusFunc
	mediaRetryCache       sync.Map
	stopGC                chan struct{}
}

func (m *Manager) StartCacheGC() {
	m.stopGC = make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-m.stopGC:
				return
			case <-ticker.C:
				now := time.Now()
				m.mediaRetryCache.Range(func(key, value any) bool {
					if entry, ok := value.(mediaRetryCacheEntry); ok && now.After(entry.expiresAt) {
						m.mediaRetryCache.Delete(key)
					}
					return true
				})
			}
		}
	}()
}

func (m *Manager) StopCacheGC() {
	if m.stopGC != nil {
		close(m.stopGC)
	}
}

func (m *Manager) SetMediaAutoUpload(fn MediaAutoUploadFunc) {
	m.OnMediaReceived = fn
}

func (m *Manager) SetMediaRetry(fn MediaRetryFunc) {
	m.OnMediaRetry = fn
}

func (m *Manager) RequestMediaRetry(ctx context.Context, sessionID, messageID, chatJID, senderJID string, fromMe bool, mimeType string, timestamp time.Time, encFileHash, fileHash, mediaKey []byte, fileLength int) error {
	client, err := m.GetClient(sessionID)
	if err != nil {
		return err
	}

	parsedChatJID, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("failed to parse chat JID: %w", err)
	}

	var parsedSenderJID types.JID
	if senderJID != "" {
		parsedSenderJID, _ = types.ParseJID(senderJID)
	}

	msgInfo := types.MessageInfo{
		MessageSource: types.MessageSource{
			Chat:     parsedChatJID,
			Sender:   parsedSenderJID,
			IsFromMe: fromMe,
			IsGroup:  parsedChatJID.Server == "g.us",
		},
		ID:        types.MessageID(messageID),
		Timestamp: timestamp,
	}

	m.mediaRetryCache.Store(messageID, mediaRetryCacheEntry{
		sessionID:   sessionID,
		chatJID:     chatJID,
		senderJID:   senderJID,
		fromMe:      fromMe,
		mimeType:    mimeType,
		timestamp:   timestamp,
		encFileHash: encFileHash,
		fileHash:    fileHash,
		mediaKey:    mediaKey,
		fileLength:  fileLength,
		expiresAt:   time.Now().Add(10 * time.Minute),
	})

	if err := client.SendMediaRetryReceipt(ctx, &msgInfo, mediaKey); err != nil {
		m.mediaRetryCache.Delete(messageID)
		return fmt.Errorf("failed to send media retry receipt: %w", err)
	}

	return nil
}

func (m *Manager) SetMessagePersist(fn MessagePersistFunc) {
	m.OnMessageReceived = fn
}

func (m *Manager) SetHistorySyncPersist(fn HistorySyncPersistFunc) {
	m.OnHistorySyncReceived = fn
}

func (m *Manager) SetStatusReceived(fn StatusReceivedFunc) {
	m.OnStatusReceived = fn
}

func (m *Manager) SetStatusMediaAutoUpload(fn MediaAutoUploadFunc) {
	m.OnStatusMediaReceived = fn
}

func (m *Manager) SetShouldIgnoreStatus(fn ShouldIgnoreStatusFunc) {
	m.ShouldIgnoreStatus = fn
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
	switch {
	case strings.HasPrefix(mediaType, "image/"), mediaType == "image", mediaType == "sticker":
		wmMediaType = whatsmeow.MediaImage
	case strings.HasPrefix(mediaType, "audio/"), mediaType == "audio":
		wmMediaType = whatsmeow.MediaAudio
	case strings.HasPrefix(mediaType, "video/"), mediaType == "video":
		wmMediaType = whatsmeow.MediaVideo
	case strings.HasPrefix(mediaType, "application/"), mediaType == "document":
		wmMediaType = whatsmeow.MediaDocument
	default:
		wmMediaType = whatsmeow.MediaDocument
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

func (m *Manager) GetContactName(ctx context.Context, sessionID, jid string) string {
	client, err := m.GetClient(sessionID)
	if err != nil {
		return ""
	}

	parsedJID, err := types.ParseJID(jid)
	if err != nil {
		return ""
	}

	contact, err := client.Store.Contacts.GetContact(ctx, parsedJID)
	if err != nil {
		return ""
	}

	if contact.FullName != "" {
		return contact.FullName
	}
	if contact.FirstName != "" {
		return contact.FirstName
	}
	if contact.PushName != "" {
		return contact.PushName
	}
	return ""
}
