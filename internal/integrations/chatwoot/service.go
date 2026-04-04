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

type Service struct {
	repo      *Repository
	msgRepo   *repo.MessageRepository
	clientFn  func(cfg *ChatwootConfig) *Client
	messageSvc MessageService
	convCache sync.Map
}

func NewService(repo *Repository, msgRepo *repo.MessageRepository, messageSvc MessageService) *Service {
	return &Service{
		repo:      repo,
		msgRepo:   msgRepo,
		clientFn: func(cfg *ChatwootConfig) *Client {
			return NewClient(cfg.URL, cfg.AccountID, cfg.Token, &http.Client{Timeout: 30 * time.Second})
		},
		messageSvc: messageSvc,
	}
}

func (s *Service) OnEvent(sessionID string, event model.EventType, payload []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := s.repo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return
	}
	if !cfg.Enabled {
		return
	}

	switch event {
	case model.EventMessage:
		s.handleMessage(ctx, cfg, payload)
	case model.EventReceipt:
		s.handleReceipt(ctx, cfg, payload)
	case model.EventDeleteForMe:
		s.handleDelete(ctx, cfg, payload)
	}
}

func (s *Service) handleMessage(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		logger.Warn().Err(err).Msg("Failed to unmarshal message payload for Chatwoot")
		return
	}

	if cfg.IgnoreGroups {
		if isGroup, _ := raw["isGroup"].(bool); isGroup {
			return
		}
	}

	chatJID, _ := raw["chatJid"].(string)
	pushName, _ := raw["pushName"].(string)
	if pushName == "" {
		if pn, ok := raw["pushName"].(string); ok {
			pushName = pn
		}
	}
	fromMe, _ := raw["fromMe"].(bool)
	msgID, _ := raw["id"].(string)
	msgType, _ := raw["msgType"].(string)

	if chatJID == "" {
		return
	}

	convID, err := s.findOrCreateConversation(ctx, cfg, chatJID, pushName)
	if err != nil {
		logger.Warn().Err(err).Str("session", cfg.SessionID).Str("chatJID", chatJID).Msg("Failed to find or create Chatwoot conversation")
		return
	}

	client := s.clientFn(cfg)

	if hasMediaURL(raw) {
		s.handleMediaMessage(ctx, cfg, convID, raw)
		return
	}

	text := extractText(msgType, raw)

	if fromMe && cfg.SignMsg && pushName != "" {
		phone := extractPhone(chatJID)
		text = formatGroupContent(phone, pushName, text, fromMe)
	}

	if text != "" {
		msgReq := MessageReq{
			Content:     text,
			MessageType: "outgoing",
			SourceID:    "WAID:" + msgID,
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

func hasMediaURL(raw map[string]any) bool {
	if mediaURL, ok := raw["mediaUrl"].(string); ok && mediaURL != "" {
		return true
	}
	return false
}

func (s *Service) findOrCreateConversation(ctx context.Context, cfg *ChatwootConfig, chatJID, pushName string) (int, error) {
	cacheKey := cfg.SessionID + "+" + chatJID

	val, loading := s.convCache.LoadOrStore(cacheKey, &sync.Mutex{})
	mu := val.(*sync.Mutex)
	if loading {
		mu.Lock()
		defer mu.Unlock()
	} else {
		mu.Lock()
		defer mu.Unlock()
	}

	client := s.clientFn(cfg)
	phone := extractPhone(chatJID)

	var contacts []Contact
	if cfg.MergeBRContacts && strings.HasPrefix(phone, "55") {
		contacts1, _ := client.FilterContacts(ctx, phone)
		phoneVariant := addOrRemoveBR9thDigit(phone)
		contacts2, _ := client.FilterContacts(ctx, phoneVariant)
		contacts = append(contacts1, contacts2...)
	} else {
		var err error
		contacts, err = client.FilterContacts(ctx, phone)
		if err != nil {
			contacts = nil
		}
	}

	var contactID int
	if len(contacts) == 0 {
		name := pushName
		if name == "" {
			name = phone
		}
		contact, err := client.CreateContact(ctx, CreateContactReq{
			InboxID:     cfg.InboxID,
			Name:        name,
			PhoneNumber: "+" + phone,
		})
		if err != nil {
			return 0, fmt.Errorf("failed to create contact: %w", err)
		}
		contactID = contact.ID
	} else {
		contactID = contacts[0].ID
	}

	conversations, err := client.ListContactConversations(ctx, contactID)
	if err != nil {
		return 0, fmt.Errorf("failed to list conversations: %w", err)
	}

	for _, conv := range conversations {
		if conv.InboxID == cfg.InboxID {
			if conv.Status == "resolved" && cfg.ReopenConversation {
				if err := client.UpdateConversationStatus(ctx, conv.ID, "open"); err != nil {
					logger.Warn().Err(err).Int("convID", conv.ID).Msg("Failed to reopen conversation")
				}
			}
			if conv.Status != "resolved" {
				return conv.ID, nil
			}
		}
	}

	conv, err := client.CreateConversation(ctx, CreateConversationReq{
		InboxID:  cfg.InboxID,
		SourceID: chatJID,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create conversation: %w", err)
	}

	return conv.ID, nil
}

func (s *Service) handleReceipt(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return
	}

	msgID, _ := raw["id"].(string)
	if msgID == "" {
		return
	}

	msg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, msgID)
	if err != nil {
		return
	}

	if msg.CWConversationID == nil || *msg.CWConversationID == 0 {
		return
	}

	client := s.clientFn(cfg)
	if msg.CWSourceID != nil {
		_ = client.UpdateLastSeen(ctx, fmt.Sprintf("%d", cfg.InboxID), *msg.CWSourceID, *msg.CWConversationID)
	}
}

func (s *Service) handleDelete(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return
	}

	msgID, _ := raw["id"].(string)
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

func extractText(msgType string, raw map[string]any) string {
	switch msgType {
	case "conversation", "extendedTextMessage":
		if text, ok := raw["body"].(string); ok {
			return text
		}
	case "imageMessage", "videoMessage", "documentMessage":
		if caption, ok := raw["body"].(string); ok {
			return caption
		}
	case "audioMessage":
		if text, ok := raw["body"].(string); ok && text != "" {
			return text
		}
	case "locationMessage":
		lat, _ := raw["degreesLatitude"].(float64)
		lng, _ := raw["degreesLongitude"].(float64)
		name, _ := raw["name"].(string)
		if name != "" {
			return fmt.Sprintf("📍 %s\nhttps://www.google.com/maps?q=%f,%f", name, lat, lng)
		}
		return fmt.Sprintf("📍 Location\nhttps://www.google.com/maps?q=%f,%f", lat, lng)
	case "contactMessage":
		if vcard, ok := raw["vcard"].(string); ok {
			return vcard
		}
		if displayName, ok := raw["displayName"].(string); ok {
			return displayName
		}
	}

	if body, ok := raw["body"].(string); ok {
		return body
	}
	return ""
}

func formatGroupContent(phone, pushName, body string, fromMe bool) string {
	if fromMe {
		return body
	}
	return fmt.Sprintf("**+%s - %s:**\n\n%s", phone, pushName, body)
}

func extractPhone(jid string) string {
	jid = strings.Split(jid, "@")[0]
	jid = strings.TrimPrefix(jid, "+")
	return jid
}

func addOrRemoveBR9thDigit(phone string) string {
	if !strings.HasPrefix(phone, "55") {
		return phone
	}
	parts := strings.SplitN(phone, "", 13)
	if len(parts) < 12 {
		return phone
	}
	ddd := phone[2:4]
	number := phone[4:]
	if len(number) == 8 {
		return "55" + ddd + "9" + number
	}
	if len(number) == 9 && number[0] == '9' {
		return "55" + ddd + number[1:]
	}
	return phone
}

func (s *Service) handleMediaMessage(ctx context.Context, cfg *ChatwootConfig, convID int, raw map[string]any) {
	mediaURL, _ := raw["mediaUrl"].(string)
	if mediaURL == "" {
		return
	}

	client := s.clientFn(cfg)
	resp, err := client.httpClient.Get(mediaURL)
	if err != nil {
		logger.Warn().Err(err).Str("url", mediaURL).Msg("Failed to download media for Chatwoot")
		return
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Warn().Err(err).Str("url", mediaURL).Msg("Failed to read media body")
		return
	}

	mimeType, ext := GetMIMETypeAndExt(mediaURL, data)
	filename := "file" + ext

	msgID, _ := raw["id"].(string)
	msg, err := client.CreateMessageWithAttachment(ctx, convID, "", filename, data, mimeType)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to upload media to Chatwoot")
		return
	}

	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, msg.ID, convID, msg.SourceID)
	}
}

func (s *Service) sendAttachmentToWhatsApp(ctx context.Context, cfg *ChatwootConfig, chatJID, attachmentURL, caption, mimeType string) error {
	resp, err := http.Get(attachmentURL)
	if err != nil {
		return fmt.Errorf("download attachment: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read attachment: %w", err)
	}

	msgType := strings.Split(mimeType, "/")[0]
	mediaReq := dto.SendMediaReq{
		Phone:   chatJID,
		Caption: caption,
	}

	switch msgType {
	case "image":
		_, err = s.messageSvc.SendImage(ctx, cfg.SessionID, mediaReq)
	case "video":
		_, err = s.messageSvc.SendVideo(ctx, cfg.SessionID, mediaReq)
	case "audio":
		_, err = s.messageSvc.SendAudio(ctx, cfg.SessionID, mediaReq)
	default:
		_, err = s.messageSvc.SendDocument(ctx, cfg.SessionID, mediaReq)
	}

	return err
}

func (s *Service) HandleIncomingWebhook(ctx context.Context, sessionID string, body dto.ChatwootWebhookPayload) error {
	cfg, err := s.repo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load chatwoot config: %w", err)
	}
	if !cfg.Enabled {
		return nil
	}

	if body.Message != nil && body.Message.MessageType == "outgoing" {
		return s.handleOutgoingMessage(ctx, cfg, body)
	}

	if body.EventType == "message_updated" && body.Message != nil {
		if deleted, _ := body.Message.ContentAttributes["deleted"].(bool); deleted {
			return s.handleMessageUpdated(ctx, cfg, body)
		}
	}

	if body.EventType == "conversation_status_changed" && body.Conversation != nil {
		return s.handleConversationStatusChanged(ctx, cfg, body)
	}

	return nil
}

func (s *Service) handleOutgoingMessage(ctx context.Context, cfg *ChatwootConfig, body dto.ChatwootWebhookPayload) error {
	if body.Message == nil || body.Conversation == nil {
		return nil
	}

	sourceID := body.Message.SourceID
	if strings.HasPrefix(sourceID, "WAID:") {
		return nil
	}

	chatJID := body.Conversation.ContactInbox.SourceID
	if chatJID == "" {
		return nil
	}

	if len(body.Message.Attachments) > 0 {
		for _, att := range body.Message.Attachments {
			if err := s.sendAttachmentToWhatsApp(ctx, cfg, chatJID, att.URL, body.Message.Content, att.FileType); err != nil {
				logger.Warn().Err(err).Msg("Failed to send attachment from Chatwoot to WhatsApp")
			}
		}
		return nil
	}

	content := body.Message.Content

	_, err := s.messageSvc.SendText(ctx, cfg.SessionID, dto.SendTextReq{
		Phone: chatJID,
		Body:  content,
	})
	return err
}

func (s *Service) handleMessageUpdated(ctx context.Context, cfg *ChatwootConfig, body dto.ChatwootWebhookPayload) error {
	if body.Message == nil {
		return nil
	}

	cwMsgID := body.Message.ID
	rows, err := s.msgRepo.FindByChat(ctx, cfg.SessionID, "", 1000, 0)
	if err != nil {
		return err
	}

	for _, row := range rows {
		if row.CWMessageID != nil && *row.CWMessageID == cwMsgID {
			_, _ = s.messageSvc.DeleteMessage(ctx, cfg.SessionID, dto.DeleteMessageReq{
				Phone:     row.ChatJID,
				MessageID: row.ID,
			})
			break
		}
	}

	if body.Conversation != nil && body.Conversation.Status == "resolved" && !cfg.ReopenConversation {
		cacheKey := cfg.SessionID + "+" + body.Conversation.ContactInbox.SourceID
		s.convCache.Delete(cacheKey)
	}

	return nil
}

func (s *Service) handleConversationStatusChanged(ctx context.Context, cfg *ChatwootConfig, body dto.ChatwootWebhookPayload) error {
	if body.Conversation == nil {
		return nil
	}

	if body.Conversation.Status == "resolved" && !cfg.ReopenConversation {
		sourceID := body.Conversation.ContactInbox.SourceID
		if sourceID != "" {
			cacheKey := cfg.SessionID + "+" + sourceID
			s.convCache.Delete(cacheKey)
		}
	}

	return nil
}

func (s *Service) Configure(ctx context.Context, cfg *ChatwootConfig) error {
	if cfg.InboxName == "" {
		cfg.InboxName = "wzap"
	}

	client := s.clientFn(cfg)
	webhookURL := fmt.Sprintf("%s/chatwoot/webhook/%s", cfg.URL, cfg.SessionID)

	inboxes, err := client.ListInboxes(ctx)
	if err != nil {
		inboxes = nil
	}

	found := false
	for _, inbox := range inboxes {
		if inbox.ID == cfg.InboxID {
			found = true
			break
		}
	}

	if !found && cfg.InboxID == 0 {
		inbox, err := client.CreateInbox(ctx, cfg.InboxName, webhookURL)
		if err != nil {
			return fmt.Errorf("failed to auto-create inbox: %w", err)
		}
		cfg.InboxID = inbox.ID
	}

	cfg.Enabled = true
	return s.repo.Upsert(ctx, cfg)
}
