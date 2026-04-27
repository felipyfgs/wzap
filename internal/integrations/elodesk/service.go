package elodesk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"golang.org/x/sync/singleflight"

	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/model"
	"wzap/internal/repo"
	"wzap/internal/wa"
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

type missingEntry struct {
	expiresAt time.Time
}

type Service struct {
	repo              Repo
	msgRepo           repo.MessageRepo
	clientFn          func(cfg *Config) Client
	messageSvc        MessageService
	cache             Cache
	jidResolver       JIDResolver
	mediaDownloader   MediaDownloader
	connector         SessionConnector
	picGetter         AvatarGetter
	numberChecker     NumberChecker
	contactNameGetter ContactNameGetter
	serverURL         string
	httpClient        *http.Client
	js                jetstream.JetStream
	cb                *cbManager
	convFlight        singleflight.Group
	importFlight      singleflight.Group
	missingConfig     sync.Map
	lastBotNotify     sync.Map
}

func NewService(ctx context.Context, repo Repo, msgRepo repo.MessageRepo, messageSvc MessageService) *Service {
	return &Service{
		repo:    repo,
		msgRepo: msgRepo,
		clientFn: func(cfg *Config) Client {
			return NewClient(cfg.URL, cfg.APIToken, &http.Client{Timeout: 30 * time.Second})
		},
		messageSvc: messageSvc,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cb:         newCircuitBreakerManager(),
		cache:      newMemoryCache(ctx),
	}
}

func (s *Service) SetJIDResolver(r JIDResolver)           { s.jidResolver = r }
func (s *Service) SetMediaDownloader(d MediaDownloader)   { s.mediaDownloader = d }
func (s *Service) SetSessionConnector(c SessionConnector) { s.connector = c }
func (s *Service) SetAvatarGetter(p AvatarGetter)         { s.picGetter = p }
func (s *Service) SetNumberChecker(n NumberChecker)       { s.numberChecker = n }
func (s *Service) SetServerURL(url string)                { s.serverURL = url }
func (s *Service) SetJetStream(js jetstream.JetStream)    { s.js = js }
func (s *Service) SetCache(c Cache)                       { s.cache = c }
func (s *Service) SetNameGetter(g ContactNameGetter)      { s.contactNameGetter = g }

func (s *Service) clearConfigCache(sessionID string) { s.missingConfig.Delete(sessionID) }

// OnEvent é o ponto de entrada da integração — chamado por
// webhook.Dispatcher quando um evento WA é publicado.
func (s *Service) OnEvent(ctx context.Context, sessionID string, event model.EventType, payload []byte) {
	if s.js != nil {
		publishCtx := ctx
		if publishCtx == nil {
			publishCtx = context.Background()
		}
		pubCtx, cancel := context.WithTimeout(publishCtx, 5*time.Second)
		defer cancel()
		if err := publishInbound(pubCtx, s.js, sessionID, event, payload); err != nil {
			logger.Warn().Str("component", "elodesk").Err(err).Str("session", sessionID).Msg("failed to publish inbound event, falling back to sync")
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
		logger.Warn().Str("component", "elodesk").Err(err).Str("session", sessionID).Str("event", string(event)).Msg("processInboundSync error")
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

	return (&eventDispatcher{svc: s}).Handle(ctx, cfg, event, payload)
}

func (s *Service) processOutbound(ctx context.Context, sessionID string, rawPayload json.RawMessage) error {
	var body dto.ElodeskWebhookPayload
	if err := json.Unmarshal(rawPayload, &body); err != nil {
		return fmt.Errorf("unmarshal outbound webhook: %w", err)
	}
	return s.HandleIncomingWebhook(ctx, sessionID, body)
}

// Configure persiste a config e reseta o cache de "config ausente".
// Se UserAccessToken estiver presente e InboxIdentifier vazio, auto-cria
// um Channel::Api inbox no elodesk (espelhando chatwoot.Service.Configure).
// Se InboxIdentifier já existir e UserAccessToken presente, atualiza
// o webhook URL do inbox existente.
func (s *Service) Configure(ctx context.Context, cfg *Config) error {
	if cfg.SignDelimiter == "" {
		cfg.SignDelimiter = "\n"
	}
	if cfg.ImportPeriod == "" {
		cfg.ImportPeriod = "7d"
	}
	if cfg.TextTimeout == 0 {
		cfg.TextTimeout = 10
	}
	if cfg.MediaTimeout == 0 {
		cfg.MediaTimeout = 60
	}
	if cfg.LargeTimeout == 0 {
		cfg.LargeTimeout = 300
	}
	if cfg.AccountID == 0 {
		cfg.AccountID = 1
	}
	cfg.Enabled = true

	whURL := s.webhookURL(cfg.SessionID)

	if cfg.UserAccessToken != "" {
		needsProvision := cfg.InboxIdentifier == "" || cfg.APIToken == ""

		if needsProvision {
			client := NewClient(cfg.URL, "", s.httpClient)
			inbox, err := client.CreateInbox(ctx, cfg.AccountID, inboxName(cfg), whURL, cfg.UserAccessToken)
			if err != nil {
				return fmt.Errorf("failed to auto-create elodesk inbox: %w", err)
			}
			cfg.InboxIdentifier = inbox.Identifier
			cfg.APIToken = inbox.ApiToken
			cfg.HMACToken = inbox.HmacToken
			cfg.ChannelID = inbox.ChannelID
		} else if cfg.ChannelID > 0 {
			client := NewClient(cfg.URL, "", s.httpClient)
			if err := client.UpdateInboxWebhook(ctx, cfg.AccountID, int(cfg.ChannelID), whURL, cfg.UserAccessToken); err != nil {
				var apiErr *APIError
				if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
					client := NewClient(cfg.URL, "", s.httpClient)
					inbox, createErr := client.CreateInbox(ctx, cfg.AccountID, inboxName(cfg), whURL, cfg.UserAccessToken)
					if createErr != nil {
						return fmt.Errorf("failed to re-provision elodesk inbox: %w", createErr)
					}
					cfg.InboxIdentifier = inbox.Identifier
					cfg.APIToken = inbox.ApiToken
					cfg.HMACToken = inbox.HmacToken
					cfg.ChannelID = inbox.ChannelID
				}
			}
		}
	}

	if err := s.repo.Upsert(ctx, cfg); err != nil {
		return err
	}
	s.clearConfigCache(cfg.SessionID)
	return nil
}

func inboxName(cfg *Config) string {
	if cfg.InboxName != "" {
		return cfg.InboxName
	}
	return "wzap"
}

func (s *Service) webhookURL(sessionID string) string {
	base := s.serverURL
	if base == "" {
		base = "http://localhost:8080"
	}
	return fmt.Sprintf("%s/elodesk/webhook/%s", base, sessionID)
}

// ImportHistoryAsync é chamado em goroutine pelo handler HTTP. Bloqueia
// imports concorrentes para a mesma sessão via singleflight.
func (s *Service) ImportHistoryAsync(ctx context.Context, sessionID, period string, customDays int) {
	key := sessionID + ":" + period
	_, _, _ = s.importFlight.Do(key, func() (any, error) {
		return nil, s.importHistory(ctx, sessionID, period, customDays)
	})
}

func (s *Service) importHistory(ctx context.Context, sessionID, period string, customDays int) error {
	cfg, err := s.repo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("config not found: %w", err)
	}
	if !cfg.Enabled {
		return nil
	}

	var since time.Time
	switch period {
	case "24h":
		since = time.Now().Add(-24 * time.Hour)
	case "7d":
		since = time.Now().Add(-7 * 24 * time.Hour)
	case "30d":
		since = time.Now().Add(-30 * 24 * time.Hour)
	case "custom":
		days := customDays
		if days <= 0 {
			days = 7
		}
		since = time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	default:
		since = time.Now().Add(-7 * 24 * time.Hour)
	}

	const batchSize = 100
	offset := 0
	for {
		msgs, err := s.msgRepo.FindUnimportedHistory(ctx, sessionID, since, batchSize, offset)
		if err != nil {
			return fmt.Errorf("list unimported history: %w", err)
		}
		if len(msgs) == 0 {
			return nil
		}
		for _, m := range msgs {
			if m.Body == "" {
				_ = s.msgRepo.MarkImported(ctx, sessionID, m.ID)
				continue
			}
			if err := s.replayImportedMessage(ctx, cfg, &m); err != nil {
				logger.Warn().Str("component", "elodesk").Err(err).Str("session", sessionID).Str("msgID", m.ID).Msg("replay imported message")
				continue
			}
			_ = s.msgRepo.MarkImported(ctx, sessionID, m.ID)
		}
		offset += len(msgs)
	}
}

type managerConnector struct {
	engine *wa.Manager
}

func NewSessionConnector(engine *wa.Manager) SessionConnector {
	return &managerConnector{engine: engine}
}

func (c *managerConnector) Connect(ctx context.Context, sessionID string) error {
	_, _, err := c.engine.Connect(ctx, sessionID)
	return err
}

func (c *managerConnector) Disconnect(ctx context.Context, sessionID string) error {
	return c.engine.Disconnect(ctx, sessionID)
}

func (c *managerConnector) Logout(ctx context.Context, sessionID string) error {
	return c.engine.Logout(ctx, sessionID)
}

func (c *managerConnector) IsConnected(sessionID string) bool {
	client, err := c.engine.GetClient(sessionID)
	if err != nil {
		return false
	}
	return client.IsConnected()
}

func (s *Service) replayImportedMessage(ctx context.Context, cfg *Config, m *model.Message) error {
	sourceID := "WAID:" + m.ID
	if exists, _ := s.msgRepo.ExistsByElodeskSrcID(ctx, cfg.SessionID, sourceID); exists {
		return nil
	}
	convID, contactSrcID, err := s.findOrCreateConversation(ctx, cfg, m.ChatJID, "")
	if err != nil {
		return err
	}
	msgType := "incoming"
	if m.FromMe {
		msgType = "outgoing"
	}
	client := s.clientFn(cfg)
	out, err := client.CreateMessage(ctx, cfg.InboxIdentifier, contactSrcID, convID, MessageReq{
		Content:     m.Body,
		MessageType: msgType,
		SourceID:    sourceID,
	})
	if err != nil {
		return err
	}
	_ = s.msgRepo.UpdateElodeskRef(ctx, cfg.SessionID, m.ID, out.ID, convID, out.SourceID)
	return nil
}
