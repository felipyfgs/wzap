package chatwoot

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/model"
	"wzap/internal/repo"
)

type MessageService interface {
	SendText(ctx context.Context, sessionID string, req dto.SendTextReq) (string, error)
	SendImage(ctx context.Context, sessionID string, req dto.SendMediaReq) (string, error)
	SendVideo(ctx context.Context, sessionID string, req dto.SendMediaReq) (string, error)
	SendDocument(ctx context.Context, sessionID string, req dto.SendMediaReq) (string, error)
	SendAudio(ctx context.Context, sessionID string, req dto.SendMediaReq) (string, error)
	DeleteMessage(ctx context.Context, sessionID string, req dto.DeleteMessageReq) (string, error)
}

type JIDResolver interface {
	GetPNForLID(ctx context.Context, sessionID, lidJID string) string
}

type Service struct {
	repo        Repo
	msgRepo     repo.MessageRepo
	clientFn    func(cfg *ChatwootConfig) CWClient
	messageSvc  MessageService
	convCache   sync.Map
	jidResolver JIDResolver
	serverURL   string
	httpClient  *http.Client
}

func NewService(repo Repo, msgRepo repo.MessageRepo, messageSvc MessageService) *Service {
	return &Service{
		repo:    repo,
		msgRepo: msgRepo,
		clientFn: func(cfg *ChatwootConfig) CWClient {
			return NewClient(cfg.URL, cfg.AccountID, cfg.Token, &http.Client{Timeout: 30 * time.Second})
		},
		messageSvc: messageSvc,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *Service) SetJIDResolver(r JIDResolver) {
	s.jidResolver = r
}

func (s *Service) SetServerURL(url string) {
	s.serverURL = url
}

func (s *Service) OnEvent(sessionID string, event model.EventType, payload []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger.Debug().Str("session", sessionID).Str("event", string(event)).Msg("[CW] OnEvent received")

	cfg, err := s.repo.FindBySessionID(ctx, sessionID)
	if err != nil {
		logger.Debug().Str("session", sessionID).Err(err).Msg("[CW] config not found, skipping")
		return
	}
	if !cfg.Enabled {
		logger.Debug().Str("session", sessionID).Msg("[CW] integration disabled, skipping")
		return
	}

	switch event {
	case model.EventMessage:
		s.handleMessage(ctx, cfg, payload)
	case model.EventReceipt:
		s.handleReceipt(ctx, cfg, payload)
	case model.EventDeleteForMe:
		s.handleDelete(ctx, cfg, payload)
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
	}
}

func (s *Service) handleMessage(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
	data, err := parseMessagePayload(payload)
	if err != nil {
		logger.Warn().Err(err).Msg("[CW] Failed to parse message payload")
		return
	}

	logger.Debug().Str("chat", data.Info.Chat).Str("id", data.Info.ID).Bool("fromMe", data.Info.IsFromMe).Msg("[CW] handleMessage")

	chatJID := data.Info.Chat
	if chatJID == "" {
		logger.Warn().Msg("[CW] chatJID empty, skipping")
		return
	}

	if strings.HasSuffix(chatJID, "@lid") && s.jidResolver != nil {
		if pn := s.jidResolver.GetPNForLID(ctx, cfg.SessionID, chatJID); pn != "" {
			logger.Debug().Str("lid", chatJID).Str("pn", pn).Msg("[CW] resolved LID to PN")
			chatJID = pn + "@s.whatsapp.net"
		}
	}

	if shouldIgnoreJID(chatJID, cfg.IgnoreGroups, cfg.IgnoreJIDs) {
		logger.Debug().Str("chat", chatJID).Msg("[CW] JID ignored, skipping")
		return
	}

	pushName := data.Info.PushName
	fromMe := data.Info.IsFromMe
	msgID := data.Info.ID

	convID, err := s.findOrCreateConversation(ctx, cfg, chatJID, pushName)
	if err != nil {
		logger.Warn().Err(err).Str("session", cfg.SessionID).Str("chatJID", chatJID).Msg("Failed to find or create Chatwoot conversation")
		return
	}

	client := s.clientFn(cfg)

	mediaURL := extractMediaURL(data.Message)
	if mediaURL != "" {
		s.handleMediaMessage(ctx, cfg, convID, msgID, mediaURL, data.Message)
		return
	}

	text := extractTextFromMessage(data.Message)
	logger.Debug().Str("text", text).Interface("msg", data.Message).Msg("[CW] extracted text")
	text = convertWAToCWMarkdown(text)

	if !fromMe && data.Info.IsGroup && cfg.SignMsg && pushName != "" {
		phone := extractPhone(chatJID)
		text = formatGroupContent(phone, pushName, text, fromMe)
	}

	if text != "" {
		messageType := "outgoing"
		if !fromMe {
			messageType = "incoming"
		}

		msgReq := MessageReq{
			Content:     text,
			MessageType: messageType,
			SourceID:    "WAID:" + msgID,
		}

		stanzaID := extractStanzaID(data.Message)
		if stanzaID != "" {
			origMsg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, stanzaID)
			if err == nil && origMsg.CWMessageID != nil && *origMsg.CWMessageID != 0 {
				msgReq.InReplyTo = *origMsg.CWMessageID
			}
		}

		msg, err := client.CreateMessage(ctx, convID, msgReq)
		if err != nil {
			logger.Warn().Err(err).Str("session", cfg.SessionID).Msg("Failed to create Chatwoot message")
			return
		}
		if msgID != "" {
			_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, msg.ID, convID, msg.SourceID)
		}
	}
}

func (s *Service) handleReceipt(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
	data, err := parseReceiptPayload(payload)
	if err != nil {
		return
	}

	if len(data.MessageIDs) == 0 {
		return
	}

	client := s.clientFn(cfg)
	for _, msgID := range data.MessageIDs {
		if msgID == "" {
			continue
		}

		msg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, msgID)
		if err != nil {
			continue
		}

		if msg.CWConversationID == nil || *msg.CWConversationID == 0 {
			continue
		}

		if msg.CWSourceID != nil {
			_ = client.UpdateLastSeen(ctx, fmt.Sprintf("%d", cfg.InboxID), *msg.CWSourceID, *msg.CWConversationID)
		}
	}
}

func (s *Service) handleDelete(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
	data, err := parseDeletePayload(payload)
	if err != nil {
		return
	}

	msgID := data.MessageID
	if msgID == "" {
		return
	}

	msg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, msgID)
	if err != nil {
		return
	}

	if msg.CWMessageID == nil || msg.CWConversationID == nil {
		return
	}

	client := s.clientFn(cfg)
	if err := client.DeleteMessage(ctx, *msg.CWConversationID, *msg.CWMessageID); err != nil {
		logger.Warn().Err(err).Msg("Failed to delete Chatwoot message")
	}
}

func (s *Service) handleConnected(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
	convID, err := s.findOrCreateBotConversation(ctx, cfg)
	if err != nil {
		logger.Warn().Err(err).Str("session", cfg.SessionID).Msg("Failed to find or create bot conversation for connected event")
		return
	}

	client := s.clientFn(cfg)
	_, _ = client.CreateMessage(ctx, convID, MessageReq{
		Content:     "✅ Session connected",
		MessageType: "outgoing",
	})
}

func (s *Service) handleDisconnected(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
	convID, err := s.findOrCreateBotConversation(ctx, cfg)
	if err != nil {
		logger.Warn().Err(err).Str("session", cfg.SessionID).Msg("Failed to find or create bot conversation for disconnected event")
		return
	}

	client := s.clientFn(cfg)
	_, _ = client.CreateMessage(ctx, convID, MessageReq{
		Content:     "⚠️ Session disconnected",
		MessageType: "outgoing",
	})
}

func (s *Service) handleQR(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
	var data struct {
		Codes []string `json:"Codes"`
	}
	if err := parseEnvelopeData(payload, &data); err != nil {
		return
	}

	if len(data.Codes) == 0 {
		return
	}

	qrContent := data.Codes[0]

	convID, err := s.findOrCreateBotConversation(ctx, cfg)
	if err != nil {
		logger.Warn().Err(err).Str("session", cfg.SessionID).Msg("Failed to find or create bot conversation for QR event")
		return
	}

	client := s.clientFn(cfg)
	qrPNG, err := generateQRCodePNG(qrContent)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to generate QR code")
		return
	}

	_, _ = client.CreateMessageWithAttachment(ctx, convID, "Scan QR code to connect", "qrcode.png", qrPNG, "image/png")
}

func (s *Service) handleContact(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
	var data struct {
		JID       string `json:"JID"`
		FirstName string `json:"FirstName"`
		FullName  string `json:"FullName"`
	}
	if err := parseEnvelopeData(payload, &data); err != nil {
		return
	}

	if data.JID == "" {
		return
	}

	phone := extractPhone(data.JID)
	name := data.FullName
	if name == "" {
		name = data.FirstName
	}
	if name == "" {
		return
	}

	client := s.clientFn(cfg)
	contacts, _ := client.FilterContacts(ctx, phone)
	if len(contacts) == 0 {
		return
	}

	_ = client.UpdateContact(ctx, contacts[0].ID, UpdateContactReq{Name: name})
}

func (s *Service) handlePushName(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
	var data struct {
		JID      string `json:"JID"`
		PushName string `json:"PushName"`
	}
	if err := parseEnvelopeData(payload, &data); err != nil {
		return
	}

	if data.JID == "" || data.PushName == "" {
		return
	}

	phone := extractPhone(data.JID)

	client := s.clientFn(cfg)
	contacts, _ := client.FilterContacts(ctx, phone)
	if len(contacts) == 0 {
		return
	}

	_ = client.UpdateContact(ctx, contacts[0].ID, UpdateContactReq{Name: data.PushName})
}

func (s *Service) handlePicture(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
	var data struct {
		JID       string `json:"JID"`
		PictureID string `json:"ID"`
		URL       string `json:"URL"`
		IsGroup   bool   `json:"IsGroup"`
	}
	if err := parseEnvelopeData(payload, &data); err != nil {
		return
	}

	if data.JID == "" || data.URL == "" {
		return
	}

	phone := extractPhone(data.JID)

	client := s.clientFn(cfg)
	contacts, _ := client.FilterContacts(ctx, phone)
	if len(contacts) == 0 {
		return
	}

	_ = client.UpdateContact(ctx, contacts[0].ID, UpdateContactReq{
		AdditionalAttributes: map[string]any{"avatar_url": data.URL},
	})
}
