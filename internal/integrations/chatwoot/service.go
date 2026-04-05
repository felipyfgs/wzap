package chatwoot

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"wzap/internal/dto"
	"wzap/internal/logger"
	"wzap/internal/model"
	"wzap/internal/repo"
)

// ── Typed event payload structs matching the canonical envelope ──

type waMessageInfo struct {
	Chat     string `json:"Chat"`
	Sender   string `json:"Sender"`
	IsFromMe bool   `json:"IsFromMe"`
	IsGroup  bool   `json:"IsGroup"`
	ID       string `json:"ID"`
	PushName string `json:"PushName"`
}

type waMessagePayload struct {
	Info    waMessageInfo          `json:"Info"`
	Message map[string]interface{} `json:"Message"`
}

type waReceiptPayload struct {
	Type       string   `json:"Type"`
	MessageIDs []string `json:"MessageIDs"`
	Chat       string   `json:"Chat"`
	Sender     string   `json:"Sender"`
	Timestamp  int64    `json:"Timestamp"`
}

type waDeletePayload struct {
	Chat      string `json:"Chat"`
	Sender    string `json:"Sender"`
	MessageID string `json:"MessageID"`
	Timestamp int64  `json:"Timestamp"`
}

func parseEnvelopeData(payload []byte, target interface{}) error {
	envelope, err := model.ParseEventEnvelope(payload)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event envelope: %w", err)
	}
	if err := json.Unmarshal(envelope.Data, target); err != nil {
		return fmt.Errorf("failed to unmarshal envelope data: %w", err)
	}
	return nil
}

func parseMessagePayload(payload []byte) (*waMessagePayload, error) {
	var data waMessagePayload
	if err := parseEnvelopeData(payload, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func parseReceiptPayload(payload []byte) (*waReceiptPayload, error) {
	var data waReceiptPayload
	if err := parseEnvelopeData(payload, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func parseDeletePayload(payload []byte) (*waDeletePayload, error) {
	var data waDeletePayload
	if err := parseEnvelopeData(payload, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

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

func extractMediaURL(msg map[string]interface{}) string {
	if msg == nil {
		return ""
	}

	mediaURLIf, ok := msg["url"]
	if !ok {
		return ""
	}

	mediaURL, ok := mediaURLIf.(string)
	if !ok {
		return ""
	}
	return mediaURL
}

func extractTextFromMessage(msg map[string]interface{}) string {
	if msg == nil {
		return ""
	}

	if conversation := getStringField(msg, "conversation"); conversation != "" {
		return conversation
	}

	if extText := getMapField(msg, "extendedTextMessage"); extText != nil {
		if text := getStringField(extText, "text"); text != "" {
			return text
		}
	}

	if imgMsg := getMapField(msg, "imageMessage"); imgMsg != nil {
		return getStringField(imgMsg, "caption")
	}

	if vidMsg := getMapField(msg, "videoMessage"); vidMsg != nil {
		return getStringField(vidMsg, "caption")
	}

	if docMsg := getMapField(msg, "documentMessage"); docMsg != nil {
		caption := getStringField(docMsg, "caption")
		filename := getStringField(docMsg, "fileName")
		if caption != "" {
			return caption
		}
		return filename
	}

	if locMsg := getMapField(msg, "locationMessage"); locMsg != nil {
		lat := getFloatField(locMsg, "degreesLatitude")
		lng := getFloatField(locMsg, "degreesLongitude")
		name := getStringField(locMsg, "name")
		if name != "" {
			return fmt.Sprintf("📍 %s\nhttps://www.google.com/maps?q=%f,%f", name, lat, lng)
		}
		return fmt.Sprintf("📍 Location\nhttps://www.google.com/maps?q=%f,%f", lat, lng)
	}

	if contactMsg := getMapField(msg, "contactMessage"); contactMsg != nil {
		if vcard := getStringField(contactMsg, "vcard"); vcard != "" {
			return vcard
		}
		return getStringField(contactMsg, "displayName")
	}

	return ""
}

func getStringField(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloatField(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

func getMapField(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key]; ok {
		if m2, ok := v.(map[string]interface{}); ok {
			return m2
		}
	}
	return nil
}

func extractStanzaID(msg map[string]interface{}) string {
	if msg == nil {
		return ""
	}

	extText := getMapField(msg, "extendedTextMessage")
	if extText != nil {
		contextInfo := getMapField(extText, "contextInfo")
		if contextInfo != nil {
			return getStringField(contextInfo, "stanzaId")
		}
	}

	return ""
}

func convertWAToCWMarkdown(s string) string {
	s = waBoldToCW.ReplaceAllString(s, "**${1}**")
	s = waItalicToCW.ReplaceAllString(s, "*${1}*")
	s = waStrikeToCW.ReplaceAllString(s, "~~${1}~~")
	return s
}

func convertCWToWAMarkdown(s string) string {
	s = cwBoldToWA.ReplaceAllString(s, "\x00BOLD\x00${1}\x00/BOLD\x00")
	s = cwStrikeToWA.ReplaceAllString(s, "\x00STRIKE\x00${1}\x00/STRIKE\x00")
	s = cwItalicToWA.ReplaceAllString(s, "_${1}_")
	s = strings.ReplaceAll(s, "\x00BOLD\x00", "*")
	s = strings.ReplaceAll(s, "\x00/BOLD\x00", "*")
	s = strings.ReplaceAll(s, "\x00STRIKE\x00", "~")
	s = strings.ReplaceAll(s, "\x00/STRIKE\x00", "~")
	return s
}

var (
	waBoldToCW   = regexp.MustCompile(`\*([^*\n]+?)\*`)
	waItalicToCW = regexp.MustCompile(`_([^_\n]+?)_`)
	waStrikeToCW = regexp.MustCompile(`~([^~\n]+?)~`)

	cwBoldToWA   = regexp.MustCompile(`\*\*([^*\n]+?)\*\*`)
	cwItalicToWA = regexp.MustCompile(`\*(\S(?:[^*\n]*\S)?)\*`)
	cwStrikeToWA = regexp.MustCompile(`~~([^~\n]+?)~~`)
)

func shouldIgnoreJID(chatJID string, ignoreGroups bool, ignoreJIDs []string) bool {
	if ignoreGroups && strings.HasSuffix(chatJID, "@g.us") {
		return true
	}

	for _, jid := range ignoreJIDs {
		if jid == "@g.us" && strings.HasSuffix(chatJID, "@g.us") {
			return true
		}
		if jid == "@s.whatsapp.net" && strings.HasSuffix(chatJID, "@s.whatsapp.net") {
			return true
		}
		if jid == chatJID {
			return true
		}
	}

	return false
}

func jidsContainGroup(ignoreJIDs []string) bool {
	for _, jid := range ignoreJIDs {
		if jid == "@g.us" {
			return true
		}
	}
	return false
}

func (s *Service) findOrCreateConversation(ctx context.Context, cfg *ChatwootConfig, chatJID, pushName string) (int, error) {
	cacheKey := cfg.SessionID + "+" + chatJID

	val, _ := s.convCache.LoadOrStore(cacheKey, &sync.Mutex{})
	mu := val.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

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
		logger.Debug().Int("contactID", contact.ID).Str("phone", phone).Msg("[CW] contact created")
		contactID = contact.ID
	} else {
		contactID = contacts[0].ID
		logger.Debug().Int("contactID", contactID).Str("phone", phone).Msg("[CW] contact found")
		if pushName != "" && contacts[0].Name != pushName {
			_ = client.UpdateContact(ctx, contactID, UpdateContactReq{Name: pushName})
		}
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
		InboxID:   cfg.InboxID,
		SourceID:  chatJID,
		ContactID: contactID,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create conversation: %w", err)
	}

	return conv.ID, nil
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

func (s *Service) handleMediaMessage(ctx context.Context, cfg *ChatwootConfig, convID int, msgID, mediaURL string, msg map[string]interface{}) {
	if mediaURL == "" {
		return
	}

	resp, err := s.httpClient.Get(mediaURL)
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

	caption := extractTextFromMessage(msg)
	client := s.clientFn(cfg)
	cwMsg, err := client.CreateMessageWithAttachment(ctx, convID, caption, filename, data, mimeType)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to upload media to Chatwoot")
		return
	}

	if msgID != "" {
		_ = s.msgRepo.UpdateChatwootRef(ctx, cfg.SessionID, msgID, cwMsg.ID, convID, cwMsg.SourceID)
	}
}

func (s *Service) sendAttachmentToWhatsApp(ctx context.Context, cfg *ChatwootConfig, chatJID, attachmentURL, caption, mimeType string, replyTo *dto.ReplyContext) error {
	resp, err := s.httpClient.Get(attachmentURL)
	if err != nil {
		return fmt.Errorf("download attachment: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read attachment: %w", err)
	}

	msgType := strings.Split(mimeType, "/")[0]
	mediaReq := dto.SendMediaReq{
		Phone:    chatJID,
		Caption:  caption,
		MimeType: mimeType,
		Base64:   base64.StdEncoding.EncodeToString(data),
		ReplyTo:  replyTo,
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

	if body.Message != nil && body.Message.MessageType == 1 {
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

	var replyTo *dto.ReplyContext
	if body.Message.ContentAttributes != nil {
		if inReplyTo, ok := body.Message.ContentAttributes["in_reply_to"].(float64); ok && inReplyTo > 0 {
			origMsg, err := s.msgRepo.FindByCWMessageID(ctx, cfg.SessionID, int(inReplyTo))
			if err == nil {
				replyTo = &dto.ReplyContext{MessageID: origMsg.ID}
			}
		}
	}

	if len(body.Message.Attachments) > 0 {
		for _, att := range body.Message.Attachments {
			if err := s.sendAttachmentToWhatsApp(ctx, cfg, chatJID, att.URL, body.Message.Content, att.FileType, replyTo); err != nil {
				logger.Warn().Err(err).Msg("Failed to send attachment from Chatwoot to WhatsApp")
			}
		}
		return nil
	}

	content := body.Message.Content
	content = convertCWToWAMarkdown(content)

	_, err := s.messageSvc.SendText(ctx, cfg.SessionID, dto.SendTextReq{
		Phone:   chatJID,
		Body:    content,
		ReplyTo: replyTo,
	})
	return err
}

func (s *Service) handleMessageUpdated(ctx context.Context, cfg *ChatwootConfig, body dto.ChatwootWebhookPayload) error {
	if body.Message == nil {
		return nil
	}

	cwMsgID := body.Message.ID
	msg, err := s.msgRepo.FindByCWMessageID(ctx, cfg.SessionID, cwMsgID)
	if err != nil {
		return nil
	}

	_, _ = s.messageSvc.DeleteMessage(ctx, cfg.SessionID, dto.DeleteMessageReq{
		Phone:     msg.ChatJID,
		MessageID: msg.ID,
	})

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

func (s *Service) findOrCreateBotConversation(ctx context.Context, cfg *ChatwootConfig) (int, error) {
	client := s.clientFn(cfg)

	contacts, err := client.FilterContacts(ctx, cfg.SessionID)
	if err != nil {
		return 0, err
	}

	var contactID int
	if len(contacts) == 0 {
		contact, err := client.CreateContact(ctx, CreateContactReq{
			InboxID:     cfg.InboxID,
			Name:        cfg.SessionID,
			PhoneNumber: cfg.SessionID,
		})
		if err != nil {
			return 0, fmt.Errorf("failed to create bot contact: %w", err)
		}
		contactID = contact.ID
	} else {
		contactID = contacts[0].ID
	}

	conversations, err := client.ListContactConversations(ctx, contactID)
	if err != nil {
		return 0, err
	}

	for _, conv := range conversations {
		if conv.InboxID == cfg.InboxID && conv.Status != "resolved" {
			return conv.ID, nil
		}
	}

	conv, err := client.CreateConversation(ctx, CreateConversationReq{
		InboxID:   cfg.InboxID,
		ContactID: contactID,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create bot conversation: %w", err)
	}

	return conv.ID, nil
}

func (s *Service) webhookURL(sessionID string) string {
	base := s.serverURL
	if base == "" {
		base = "http://localhost:8080"
	}
	return fmt.Sprintf("%s/chatwoot/webhook/%s", base, sessionID)
}

func (s *Service) Configure(ctx context.Context, cfg *ChatwootConfig) error {
	if cfg.InboxName == "" {
		cfg.InboxName = "wzap"
	}

	client := s.clientFn(cfg)
	whURL := s.webhookURL(cfg.SessionID)

	inboxes, err := client.ListInboxes(ctx)
	if err != nil {
		inboxes = nil
	}

	found := false
	for _, inbox := range inboxes {
		if inbox.ID == cfg.InboxID {
			found = true
			_ = client.UpdateInboxWebhook(ctx, cfg.InboxID, whURL)
			break
		}
	}

	if !found && cfg.InboxID == 0 {
		inbox, err := client.CreateInbox(ctx, cfg.InboxName, whURL)
		if err != nil {
			return fmt.Errorf("failed to auto-create inbox: %w", err)
		}
		cfg.InboxID = inbox.ID
	}

	cfg.Enabled = true
	return s.repo.Upsert(ctx, cfg)
}
