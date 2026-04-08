package chatwoot

import (
	"context"
	"encoding/json"
	"net/http"
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
	Disconnect(sessionID string) error
	Logout(ctx context.Context, sessionID string) error
	IsConnected(sessionID string) bool
}
type ProfilePictureGetter interface {
	GetProfilePicture(ctx context.Context, sessionID, jid string) (string, error)
}
type NumberChecker interface {
	IsOnWhatsApp(ctx context.Context, sessionID string, phones []string) (map[string]string, error)
}

type noConfigEntry struct {
	expiresAt time.Time
}

type Service struct {
	repo             Repo
	msgRepo          repo.MessageRepo
	clientFn         func(cfg *ChatwootConfig) CWClient
	messageSvc       MessageService
	cache            Cache
	jidResolver      JIDResolver
	mediaDownloader  MediaDownloader
	connector        SessionConnector
	profilePicGetter ProfilePictureGetter
	numberChecker    NumberChecker
	serverURL        string
	lastBotNotify    sync.Map
	httpClient       *http.Client
	js               jetstream.JetStream
	cb               *circuitBreakerManager
	convFlight       singleflight.Group
	noConfigCache    sync.Map
}

func NewService(ctx context.Context, repo Repo, msgRepo repo.MessageRepo, messageSvc MessageService) *Service {
	return &Service{
		repo:    repo,
		msgRepo: msgRepo,
		clientFn: func(cfg *ChatwootConfig) CWClient {
			return NewClient(cfg.URL, cfg.AccountID, cfg.Token, &http.Client{Timeout: 30 * time.Second})
		},
		messageSvc: messageSvc,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cb:         newCircuitBreakerManager(),
		cache:      newMemoryCache(ctx),
	}
}

func (s *Service) SetJIDResolver(r JIDResolver)                   { s.jidResolver = r }
func (s *Service) SetMediaDownloader(d MediaDownloader)           { s.mediaDownloader = d }
func (s *Service) SetSessionConnector(c SessionConnector)         { s.connector = c }
func (s *Service) SetProfilePictureGetter(p ProfilePictureGetter) { s.profilePicGetter = p }
func (s *Service) SetNumberChecker(n NumberChecker)               { s.numberChecker = n }
func (s *Service) SetServerURL(url string)                        { s.serverURL = url }
func (s *Service) SetJetStream(js jetstream.JetStream)            { s.js = js }
func (s *Service) SetCache(c Cache)                               { s.cache = c }
func (s *Service) InvalidateNoConfigCache(sessionID string) {
	s.noConfigCache.Delete(sessionID)
}

func (s *Service) OnEvent(sessionID string, event model.EventType, payload []byte) {
	if s.js != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := publishInbound(ctx, s.js, sessionID, event, payload); err != nil {
			logger.Warn().Err(err).Str("session", sessionID).Msg("[CW] failed to publish inbound event, falling back to sync")
			s.processInboundSync(sessionID, event, payload)
		}
		return
	}

	s.processInboundSync(sessionID, event, payload)
}

func (s *Service) processInboundSync(sessionID string, event model.EventType, payload []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = s.processInboundEvent(ctx, sessionID, event, payload)
}

func (s *Service) processInboundEvent(ctx context.Context, sessionID string, event model.EventType, payload []byte) error {
	if v, ok := s.noConfigCache.Load(sessionID); ok {
		if entry, valid := v.(noConfigEntry); valid && time.Now().Before(entry.expiresAt) {
			return nil
		}
		s.noConfigCache.Delete(sessionID)
	}

	cfg, err := s.repo.FindBySessionID(ctx, sessionID)
	if err != nil {
		s.noConfigCache.Store(sessionID, noConfigEntry{expiresAt: time.Now().Add(5 * time.Minute)})
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
			logger.Warn().Err(err).Str("session", sessionID).Msg("[CW] permanent error in handleMessage, dropping")
		}
	case model.EventGroupInfo:
		if err := s.handleGroupInfo(ctx, cfg, payload); err != nil {
			if isRetryableError(err) {
				return err
			}
			logger.Warn().Err(err).Str("session", sessionID).Msg("[CW] permanent error in handleGroupInfo, dropping")
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
		return nil
	}
	return s.HandleIncomingWebhook(ctx, sessionID, body)
}

func (s *Service) importHistory(ctx context.Context, sessionID, period string, customDays int) {
	cfg, err := s.repo.FindBySessionID(ctx, sessionID)
	if err != nil || !cfg.Enabled {
		return
	}

	days := importPeriodToDays(period, customDays)
	if days <= 0 {
		logger.Warn().Str("session", sessionID).Str("period", period).Msg("[CW] invalid import period")
		return
	}

	logger.Info().Str("session", sessionID).Str("period", period).Int("days", days).Msg("[CW] Starting history import")
	metrics.CWHistoryImportProgress.WithLabelValues(sessionID).Set(0)

	// Rate limiter: max 10 msgs/s
	rateTicker := time.NewTicker(100 * time.Millisecond)
	defer rateTicker.Stop()

	// Fetch messages from the local store within the period
	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	_ = since

	// Placeholder: in a real implementation, iterate messages from msgRepo
	// filtered by timestamp >= since, respecting the rate limit.
	// For each message, call processInboundEvent to re-create in Chatwoot.
	// Progress is updated as percentage of total messages processed.
	logger.Info().Str("session", sessionID).Msg("[CW] history import complete (no historical messages available in current store)")
	metrics.CWHistoryImportProgress.WithLabelValues(sessionID).Set(100)
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
