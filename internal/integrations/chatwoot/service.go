package chatwoot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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

type MediaPresigner interface {
	GetPresignedURL(ctx context.Context, key string) (string, error)
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
	mediaPresigner    MediaPresigner
	serverURL         string
	lastBotNotify     sync.Map
	httpClient        *http.Client
	js                jetstream.JetStream
	cb                *cbManager
	convFlight        singleflight.Group
	importFlight      singleflight.Group
	missingConfig     sync.Map
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

func (s *Service) SetJIDResolver(r JIDResolver)           { s.jidResolver = r }
func (s *Service) SetMediaDownloader(d MediaDownloader)   { s.mediaDownloader = d }
func (s *Service) SetSessionConnector(c SessionConnector) { s.connector = c }
func (s *Service) SetAvatarGetter(p AvatarGetter)         { s.picGetter = p }
func (s *Service) SetNumberChecker(n NumberChecker)       { s.numberChecker = n }
func (s *Service) SetServerURL(url string)                { s.serverURL = url }
func (s *Service) SetJetStream(js jetstream.JetStream)    { s.js = js }
func (s *Service) SetCache(c Cache)                       { s.cache = c }
func (s *Service) SetNameGetter(g ContactNameGetter)      { s.contactNameGetter = g }
func (s *Service) SetMediaPresigner(p MediaPresigner)     { s.mediaPresigner = p }
func (s *Service) clearConfigCache(sessionID string) {
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

	return (&eventDispatcher{svc: s}).Handle(ctx, cfg, event, payload)
}

func (s *Service) processOutbound(ctx context.Context, sessionID string, rawPayload json.RawMessage) error {
	var body dto.CWWebhookPayload
	if err := json.Unmarshal(rawPayload, &body); err != nil {
		return fmt.Errorf("unmarshal outbound webhook: %w", err)
	}
	return s.HandleIncomingWebhook(ctx, sessionID, body)
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

const tracerName = "wzap/chatwoot"

func startSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)
	return tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

func spanAttrs(sessionID, msgType, direction string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("messaging.system", "whatsapp"),
		attribute.String("session.id", sessionID),
		attribute.String("message.type", msgType),
		attribute.String("message.direction", direction),
	}
}

// natsHeaderCarrier adapts nats.Header to the TextMapCarrier interface.
type natsHeaderCarrier nats.Header

func (c natsHeaderCarrier) Get(key string) string {
	return nats.Header(c).Get(key)
}
func (c natsHeaderCarrier) Set(key, val string) {
	nats.Header(c).Set(key, val)
}
func (c natsHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// InjectNATSHeaders injects the current trace context into NATS message headers.
func InjectNATSHeaders(ctx context.Context, msg *nats.Msg) {
	if msg.Header == nil {
		msg.Header = make(nats.Header)
	}
	otel.GetTextMapPropagator().Inject(ctx, natsHeaderCarrier(msg.Header))
}
