package chatwoot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"golang.org/x/sync/singleflight"

	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/metrics"
	"wzap/internal/model"
	"wzap/internal/repo"
)

type MessageService interface {
	SendText(ctx context.Context, sessionID string, req dto.SendTextReq) (string, error)
	SendImage(ctx context.Context, sessionID string, req dto.SendMediaReq) (string, error)
	SendVideo(ctx context.Context, sessionID string, req dto.SendMediaReq) (string, error)
	SendDocument(ctx context.Context, sessionID string, req dto.SendMediaReq) (string, error)
	SendAudio(ctx context.Context, sessionID string, req dto.SendMediaReq) (string, error)
	SendContact(ctx context.Context, sessionID string, req dto.SendContactReq) (string, error)
	SendLocation(ctx context.Context, sessionID string, req dto.SendLocationReq) (string, error)
	DeleteMessage(ctx context.Context, sessionID string, req dto.DeleteMessageReq) (string, error)
	EditMessage(ctx context.Context, sessionID string, req dto.EditMessageReq) (string, error)
	MarkRead(ctx context.Context, sessionID string, req dto.MarkReadReq) error
}
type JIDResolver interface {
	GetPNForLID(ctx context.Context, sessionID, lidJID string) string
}
type MediaDownloader interface {
	DownloadMediaByPath(ctx context.Context, sessionID, directPath string, encFileHash, fileHash, mediaKey []byte, fileLength int, mediaType string) ([]byte, error)
}
type SessionConnector interface {
	Connect(ctx context.Context, sessionID string) error
	Disconnect(ctx context.Context, sessionID string) error
	Logout(ctx context.Context, sessionID string) error
	IsConnected(sessionID string) bool
}
type AvatarGetter interface {
	GetProfilePicture(ctx context.Context, sessionID, jid string) (string, error)
}
type NumberChecker interface {
	IsOnWhatsApp(ctx context.Context, sessionID string, phones []string) (map[string]string, error)
}

type ContactNameGetter interface {
	GetContactName(ctx context.Context, sessionID, jid string) string
}

type MediaPresigner interface {
	GetPresignedURL(ctx context.Context, key string) (string, error)
}

type missingEntry struct {
	expiresAt time.Time
}

type Service struct {
	repo            Repo
	msgRepo         repo.MessageRepo
	clientFn        func(cfg *Config) Client
	messageSvc      MessageService
	cache           Cache
	jidResolver     JIDResolver
	mediaDownloader MediaDownloader
	connector       SessionConnector
	picGetter       AvatarGetter
	numberChecker   NumberChecker
	contactNameGetter ContactNameGetter
	mediaPresigner    MediaPresigner
	serverURL       string
	lastBotNotify   sync.Map
	httpClient      *http.Client
	js              jetstream.JetStream
	cb              *circuitBreakerManager
	convFlight      singleflight.Group
	importFlight    singleflight.Group
	missingConfig   sync.Map
}

func NewService(ctx context.Context, repo Repo, msgRepo repo.MessageRepo, messageSvc MessageService) *Service {
	return &Service{
		repo:    repo,
		msgRepo: msgRepo,
		clientFn: func(cfg *Config) Client {
			return NewClient(cfg.URL, cfg.AccountID, cfg.Token, &http.Client{Timeout: 30 * time.Second})
		},
		messageSvc: messageSvc,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cb:         newCircuitBreakerManager(),
		cache:      newMemoryCache(ctx),
	}
}

func (s *Service) SetJIDResolver(r JIDResolver)               { s.jidResolver = r }
func (s *Service) SetMediaDownloader(d MediaDownloader)        { s.mediaDownloader = d }
func (s *Service) SetSessionConnector(c SessionConnector)      { s.connector = c }
func (s *Service) SetAvatarGetter(p AvatarGetter)              { s.picGetter = p }
func (s *Service) SetNumberChecker(n NumberChecker)            { s.numberChecker = n }
func (s *Service) SetServerURL(url string)                     { s.serverURL = url }
func (s *Service) SetJetStream(js jetstream.JetStream)         { s.js = js }
func (s *Service) SetCache(c Cache)                            { s.cache = c }
func (s *Service) SetContactNameGetter(g ContactNameGetter)    { s.contactNameGetter = g }
func (s *Service) SetMediaPresigner(p MediaPresigner)          { s.mediaPresigner = p }
func (s *Service) ClearConfigCache(sessionID string) {
	s.missingConfig.Delete(sessionID)
}

func (s *Service) OnEvent(ctx context.Context, sessionID string, event model.EventType, payload []byte) {
	if s.js != nil {
		publishCtx := ctx
		if publishCtx == nil {
			publishCtx = context.Background()
		}
		pubCtx, cancel := context.WithTimeout(publishCtx, 5*time.Second)
		defer cancel()
		if err := publishInbound(pubCtx, s.js, sessionID, event, payload); err != nil {
			logger.Warn().Str("component", "chatwoot").Err(err).Str("session", sessionID).Msg("failed to publish inbound event, falling back to sync")
			s.processInboundSync(ctx, sessionID, event, payload)
		}
		return
	}

	s.processInboundSync(ctx, sessionID, event, payload)
}

func (s *Service) processInboundSync(ctx context.Context, sessionID string, event model.EventType, payload []byte) {
	syncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := s.processInboundEvent(syncCtx, sessionID, event, payload); err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", sessionID).Str("event", string(event)).Msg("processInboundSync error")
	}
}

func (s *Service) processInboundEvent(ctx context.Context, sessionID string, event model.EventType, payload []byte) error {
	if v, ok := s.missingConfig.Load(sessionID); ok {
		if entry, valid := v.(missingEntry); valid && time.Now().Before(entry.expiresAt) {
			return nil
		}
		s.missingConfig.Delete(sessionID)
	}

	cfg, err := s.repo.FindBySessionID(ctx, sessionID)
	if err != nil {
		s.missingConfig.Store(sessionID, missingEntry{expiresAt: time.Now().Add(5 * time.Minute)})
		return nil
	}
	if !cfg.Enabled {
		return nil
	}

	switch event {
	case model.EventMessage:
		if err := s.handleMessage(ctx, cfg, payload); err != nil {
			if isRetryableError(err) {
				return err
			}
			logger.Warn().Str("component", "chatwoot").Err(err).Str("session", sessionID).Msg("permanent error in handleMessage, dropping")
		}
	case model.EventGroupInfo:
		if err := s.handleGroupInfo(ctx, cfg, payload); err != nil {
			if isRetryableError(err) {
				return err
			}
			logger.Warn().Str("component", "chatwoot").Err(err).Str("session", sessionID).Msg("permanent error in handleGroupInfo, dropping")
		}
	case model.EventReceipt:
		s.handleReceipt(ctx, cfg, payload)
	case model.EventDeleteForMe:
		s.handleDelete(ctx, cfg, payload)
	case model.EventMessageRevoke:
		s.handleRevoke(ctx, cfg, payload)
	case model.EventMessageEdit:
		s.handleEdit(ctx, cfg, payload)
	case model.EventConnected:
		s.handleConnected(ctx, cfg, payload)
	case model.EventDisconnected:
		s.handleDisconnected(ctx, cfg, payload)
	case model.EventQR:
		s.handleQR(ctx, cfg, payload)
	case model.EventContact:
		s.handleContact(ctx, cfg, payload)
	case model.EventPushName:
		s.handlePushName(ctx, cfg, payload)
	case model.EventPicture:
		s.handlePicture(ctx, cfg, payload)
	case model.EventHistorySync:
		s.handleHistorySync(ctx, cfg, payload)
	}
	return nil
}

func (s *Service) processOutboundWebhook(ctx context.Context, sessionID string, rawPayload json.RawMessage) error {
	var body dto.ChatwootWebhookPayload
	if err := json.Unmarshal(rawPayload, &body); err != nil {
		return fmt.Errorf("unmarshal outbound webhook: %w", err)
	}
	return s.HandleIncomingWebhook(ctx, sessionID, body)
}

func (s *Service) importHistory(ctx context.Context, sessionID, period string, customDays int) {
	cfg, err := s.repo.FindBySessionID(ctx, sessionID)
	if err != nil || !cfg.Enabled {
		return
	}

	if s.msgRepo == nil {
		logger.Warn().Str("component", "chatwoot").Str("session", sessionID).Msg("msgRepo is nil, cannot import history")
		return
	}

	days := importPeriodToDays(period, customDays)
	if days <= 0 {
		logger.Warn().Str("component", "chatwoot").Str("session", sessionID).Str("period", period).Msg("invalid import period")
		return
	}

	logger.Info().Str("component", "chatwoot").Str("session", sessionID).Str("period", period).Int("days", days).Msg("Starting history import")
	metrics.CWHistoryImportProgress.WithLabelValues(sessionID).Set(0)

	rateTicker := time.NewTicker(100 * time.Millisecond)
	defer rateTicker.Stop()

	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)

	var totalProcessed int
	var failedCount int
	const chunkSize = 100

	for {
		select {
		case <-ctx.Done():
			logger.Warn().Str("component", "chatwoot").Str("session", sessionID).Err(ctx.Err()).Msg("history import context cancelled")
			return
		default:
		}

		msgs, err := s.msgRepo.FindUnimportedHistory(ctx, sessionID, since, chunkSize, failedCount)
		if err != nil {
			logger.Warn().Str("component", "chatwoot").Err(err).Str("session", sessionID).Msg("failed to fetch unimported history")
			return
		}
		if len(msgs) == 0 {
			break
		}

		for _, msg := range msgs {
			select {
			case <-ctx.Done():
				logger.Warn().Str("component", "chatwoot").Str("session", sessionID).Err(ctx.Err()).Msg("history import context cancelled during processing")
				return
			case <-rateTicker.C:
			}

			if err := s.importSingleMessage(ctx, cfg, &msg); err != nil {
				failedCount++
				logger.Warn().Str("component", "chatwoot").Err(err).Str("session", sessionID).Str("mid", msg.ID).Msg("failed to import history message")
			} else {
				if err := s.msgRepo.MarkImportedToChatwoot(ctx, sessionID, msg.ID); err != nil {
					failedCount++
					logger.Warn().Str("component", "chatwoot").Err(err).Str("session", sessionID).Str("mid", msg.ID).Msg("failed to mark message as imported")
				}
			}

			totalProcessed++
			metrics.CWHistoryImportProgress.WithLabelValues(sessionID).Set(float64(totalProcessed))
		}

		if len(msgs) < chunkSize {
			break
		}
	}

	metrics.CWHistoryImportProgress.WithLabelValues(sessionID).Set(100)
	logger.Info().Str("component", "chatwoot").Str("session", sessionID).Int("processed", totalProcessed).Msg("history import complete")
}

func (s *Service) importSingleMessage(ctx context.Context, cfg *Config, msg *model.Message) error {
	if msg == nil || msg.ID == "" {
		return nil
	}

	chatJID := msg.ChatJID
	if strings.HasSuffix(chatJID, "@lid") {
		if s.jidResolver != nil {
			if pn := s.jidResolver.GetPNForLID(ctx, cfg.SessionID, chatJID); pn != "" {
				chatJID = pn + "@s.whatsapp.net"
			}
		}
	}

	contactName := ""
	if s.contactNameGetter != nil {
		contactName = s.contactNameGetter.GetContactName(ctx, cfg.SessionID, chatJID)
	}

	convID, err := s.findOrCreateConversation(ctx, cfg, chatJID, contactName)
	if err != nil {
		return fmt.Errorf("findOrCreateConversation: %w", err)
	}

	client := s.clientFn(cfg)
	sourceID := "WAID:" + msg.ID
	messageType := "outgoing"
	if !msg.FromMe {
		messageType = "incoming"
	}

	if msg.MediaURL != "" {
		if !strings.HasPrefix(msg.MediaURL, "http") && s.mediaPresigner != nil {
			url, err := s.mediaPresigner.GetPresignedURL(ctx, msg.MediaURL)
			if err != nil {
				return fmt.Errorf("resolve media URL from key: %w", err)
			}
			msg.MediaURL = url
		}
		return s.importMediaMessage(ctx, cfg, client, convID, msg, messageType, sourceID)
	}

	if msg.Body == "" {
		return nil
	}

	content := msg.Body
	if msg.MsgType == "text" {
		content = convertWAToCWMarkdown(content)
	}

	_, err = client.CreateMessage(ctx, convID, MessageReq{
		Content:     content,
		MessageType: messageType,
		SourceID:    sourceID,
	})
	if err != nil {
		return fmt.Errorf("CreateMessage: %w", err)
	}

	return nil
}

func (s *Service) importMediaMessage(ctx context.Context, cfg *Config, client Client, convID int, msg *model.Message, messageType, sourceID string) error {
	mediaCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	req, err := httpGetWithContext(mediaCtx, s.httpClient, msg.MediaURL)
	if err != nil {
		return fmt.Errorf("download media from minio: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, req.Body)
		_ = req.Body.Close()
	}()

	data, err := io.ReadAll(io.LimitReader(req.Body, maxMediaBytes+1))
	if err != nil {
		return fmt.Errorf("read media data: %w", err)
	}
	if int64(len(data)) > maxMediaBytes {
		return fmt.Errorf("media too large: %d bytes", len(data))
	}

	mimeType := msg.MediaType
	if mimeType == "" {
		mimeType, _ = DetectMIME("", data)
	}

	filename := msg.MsgType
	ext := mimeTypeToExt(mimeType)
	if ext != "" {
		filename += ext
	}

	caption := msg.Body

	_, err = client.CreateMessageWithAttachment(ctx, convID, caption, filename, data, mimeType, messageType, sourceID, 0, nil)
	if err != nil {
		return fmt.Errorf("CreateMessageWithAttachment: %w", err)
	}

	return nil
}

func (s *Service) ImportHistoryAsync(ctx context.Context, sessionID, period string, customDays int) {
	key := "import:" + sessionID
	_, _, _ = s.importFlight.Do(key, func() (interface{}, error) {
		importCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		s.importHistory(importCtx, sessionID, period, customDays)
		return nil, nil
	})
}

func importPeriodToDays(period string, customDays int) int {
	switch period {
	case "24h":
		return 1
	case "7d":
		return 7
	case "30d":
		return 30
	case "custom":
		return customDays
	}
	return 0
}
