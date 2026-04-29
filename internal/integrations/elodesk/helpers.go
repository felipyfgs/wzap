package elodesk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"wzap/internal/model"
)

// Este arquivo duplica helpers básicos de internal/integrations/chatwoot/
// (extractors.go, jid.go, etc.). A duplicação é aceita até surgir a 3ª
// integração — ver design.md D1.

type flexTimestamp int64

func (ft *flexTimestamp) UnmarshalJSON(b []byte) error {
	var n int64
	if err := json.Unmarshal(b, &n); err == nil {
		*ft = flexTimestamp(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("cannot unmarshal timestamp from %s", string(b))
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			*ft = flexTimestamp(t.Unix())
			return nil
		}
	}
	return fmt.Errorf("cannot parse timestamp string %q", s)
}

type waMessageInfo struct {
	Chat           string        `json:"Chat"`
	Sender         string        `json:"Sender"`
	SenderAlt      string        `json:"SenderAlt"`
	RecipientAlt   string        `json:"RecipientAlt"`
	AddressingMode string        `json:"AddressingMode"`
	IsFromMe       bool          `json:"IsFromMe"`
	IsGroup        bool          `json:"IsGroup"`
	ID             string        `json:"ID"`
	PushName       string        `json:"PushName"`
	Timestamp      flexTimestamp `json:"Timestamp"`
}

type waMessagePayload struct {
	Info    waMessageInfo  `json:"Info"`
	Message map[string]any `json:"Message"`
}

type waReceiptPayload struct {
	Type       string        `json:"Type"`
	MessageIDs []string      `json:"MessageIDs"`
	Chat       string        `json:"Chat"`
	Sender     string        `json:"Sender"`
	Timestamp  flexTimestamp `json:"Timestamp"`
}

type waDeletePayload struct {
	Chat      string        `json:"Chat"`
	Sender    string        `json:"Sender"`
	MessageID string        `json:"MessageID"`
	Timestamp flexTimestamp `json:"Timestamp"`
}

func parseEnvelopeData(payload []byte, target any) error {
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

func getStringField(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getMapField(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if m2, ok := v.(map[string]any); ok {
			return m2
		}
	}
	return nil
}

func detectMessageType(msg map[string]any) string {
	if msg == nil {
		return "text"
	}
	keys := []string{
		"imageMessage", "videoMessage", "audioMessage", "documentMessage",
		"stickerMessage", "contactMessage", "locationMessage",
	}
	for _, k := range keys {
		if _, ok := msg[k]; ok {
			return strings.TrimSuffix(k, "Message")
		}
	}
	return "text"
}

// extractText extrai o texto principal de uma message WhatsApp. Cobre os
// casos mais comuns (conversation, extendedTextMessage, captions). Os
// casos ricos (vCard, polls, buttons, list, template) não são traduzidos
// para o contrato elodesk no MVP — caem em no-op silencioso com body vazio.
func extractText(msg map[string]any) string {
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
		return getStringField(docMsg, "caption")
	}
	return ""
}

// extractForwardingFromMap procura o ContextInfo aninhado no payload da
// mensagem (map[string]any vindo do envelope JSON) e devolve o par
// (isForwarded, forwardingScore). Espelha a lógica de wautil.ExtractForwarding
// mas opera no formato map para evitar reparsing do proto. Cobre os tipos de
// mensagem que efetivamente carregam ContextInfo no protocolo do WhatsApp.
func extractForwardingFromMap(msg map[string]any) (bool, uint32) {
	ci := findContextInfoInMessage(msg)
	if ci == nil {
		return false, 0
	}
	isFwd, _ := ci["isForwarded"].(bool)
	var score uint32
	switch v := ci["forwardingScore"].(type) {
	case float64:
		if v > 0 {
			score = uint32(v)
		}
	case int:
		if v > 0 {
			score = uint32(v)
		}
	case int64:
		if v > 0 && v <= 0xFFFFFFFF {
			score = uint32(v)
		}
	}
	return isFwd, score
}

func findContextInfoInMessage(msg map[string]any) map[string]any {
	if msg == nil {
		return nil
	}
	if ci := getMapField(msg, "contextInfo"); ci != nil {
		return ci
	}
	for _, key := range []string{
		"extendedTextMessage", "imageMessage", "videoMessage", "audioMessage",
		"documentMessage", "stickerMessage", "contactMessage", "contactsArrayMessage",
		"liveLocationMessage", "buttonsMessage", "buttonsResponseMessage",
		"listMessage", "listResponseMessage", "templateMessage",
		"templateButtonReplyMessage", "pollCreationMessage", "groupInviteMessage",
		"productMessage", "orderMessage",
	} {
		if sub := getMapField(msg, key); sub != nil {
			if ci := getMapField(sub, "contextInfo"); ci != nil {
				return ci
			}
		}
	}
	for _, wrapper := range []string{
		"ephemeralMessage", "viewOnceMessage", "viewOnceMessageV2",
		"viewOnceMessageV2Extension", "documentWithCaptionMessage",
	} {
		if sub := getMapField(msg, wrapper); sub != nil {
			if inner := getMapField(sub, "message"); inner != nil {
				if ci := findContextInfoInMessage(inner); ci != nil {
					return ci
				}
			}
		}
	}
	return nil
}

func shouldIgnoreJID(chatJID string, ignoreGroups bool, ignoreJIDs []string) bool {
	if strings.HasPrefix(chatJID, "status@") {
		return true
	}
	if strings.HasSuffix(chatJID, "@newsletter") {
		return true
	}
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

func extractPhone(jid string) string {
	jid = strings.Split(jid, "@")[0]
	if idx := strings.Index(jid, ":"); idx >= 0 {
		jid = jid[:idx]
	}
	jid = strings.TrimPrefix(jid, "+")
	return jid
}

// stripDeviceFromJID removes the ":N" device suffix from user/lid JIDs.
// Whatsmeow's Info.Chat is normally a chat-level JID without device, but
// secondary devices can leak through (e.g. "559992032709:7@s.whatsapp.net").
// Group JIDs ("@g.us") and broadcasts ("@broadcast") never carry a device,
// so they pass through unchanged.
func stripDeviceFromJID(jid string) string {
	at := strings.LastIndex(jid, "@")
	if at < 0 {
		return jid
	}
	user := jid[:at]
	server := jid[at:]
	if server != "@s.whatsapp.net" && server != "@lid" {
		return jid
	}
	if idx := strings.Index(user, ":"); idx >= 0 {
		user = user[:idx]
	}
	return user + server
}

// resolveLID resolve um JID @lid para um JID de telefone. Tenta os altJIDs
// informados antes de cair no jidResolver (se injetado).
func (s *Service) resolveLID(ctx context.Context, sessionID, jid string, altJIDs ...string) string {
	if !strings.HasSuffix(jid, "@lid") {
		return jid
	}
	for _, alt := range altJIDs {
		if alt != "" && !strings.HasSuffix(alt, "@lid") {
			if !strings.Contains(alt, "@") {
				return alt + "@s.whatsapp.net"
			}
			return alt
		}
	}
	if s.jidResolver != nil {
		if pn := s.jidResolver.GetPNForLID(ctx, sessionID, jid); pn != "" {
			return pn + "@s.whatsapp.net"
		}
	}
	return jid
}
