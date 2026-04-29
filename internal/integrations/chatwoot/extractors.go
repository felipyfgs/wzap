package chatwoot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"wzap/internal/model"
	"wzap/internal/wautil"
)

var (
	googleMapsRegex = regexp.MustCompile(`[?&]q=(-?\d+\.\d+),(-?\d+\.\d+)`)
	coordRegex      = regexp.MustCompile(`(-?\d+\.\d+),\s*(-?\d+\.\d+)`)
)

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

type mediaInfo struct {
	DirectPath    string
	MediaKey      []byte
	FileSHA256    []byte
	FileEncSHA256 []byte
	FileLength    int
	MimeType      string
	MediaType     string
	FileName      string
}

func getStringField(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloatField(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

func getMapField(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if m2, ok := v.(map[string]any); ok {
			return m2
		}
	}
	return nil
}

func convertWAToCWMarkdown(s string) string { return wautil.WAToMarkdown(s) }

func convertCWToWAMarkdown(s string) string { return wautil.MarkdownToWA(s) }

func formatVCard(vcard string) string {
	return formatVCardWithName(vcard, "")
}

func formatVCardWithName(vcard, displayName string) string {
	name := ""
	var phones []string
	for _, raw := range strings.Split(vcard, "\n") {
		line := strings.TrimRight(raw, "\r")
		switch {
		case len(line) >= 3 && strings.EqualFold(line[:3], "FN:"):
			name = line[3:]
		case len(line) >= 3 && strings.EqualFold(line[:3], "TEL"):
			if idx := strings.LastIndex(line, ":"); idx >= 0 {
				phones = append(phones, line[idx+1:])
			}
		}
	}
	if displayName != "" {
		name = displayName
	}
	if name == "" {
		return vcard
	}
	var sb strings.Builder
	sb.WriteString("*Contato:*\n\n")
	sb.WriteString("_Nome:_ ")
	sb.WriteString(name)
	for i, phone := range phones {
		fmt.Fprintf(&sb, "\n_Número (%d):_ %s", i+1, phone)
	}
	return sb.String()
}

func isVCardContent(content string) bool {
	return strings.HasPrefix(strings.TrimSpace(content), "BEGIN:VCARD")
}

func splitVCards(content string) []string {
	var vcards []string
	lines := strings.Split(content, "\n")
	var current strings.Builder
	for _, line := range lines {
		current.WriteString(line)
		current.WriteString("\n")
		if strings.TrimSpace(line) == "END:VCARD" {
			vcards = append(vcards, current.String())
			current.Reset()
		}
	}
	return vcards
}

func extractVCardName(vcard string) string {
	for line := range strings.SplitSeq(vcard, "\n") {
		if after, ok := strings.CutPrefix(strings.TrimSpace(line), "FN:"); ok {
			return after
		}
	}
	return ""
}

var mediaTypeMap = map[string]string{
	"imageMessage":    "image",
	"videoMessage":    "video",
	"audioMessage":    "audio",
	"documentMessage": "document",
	"stickerMessage":  "sticker",
}

func extractMediaInfo(msg map[string]any) *mediaInfo {
	if msg == nil {
		return nil
	}

	for key, mt := range mediaTypeMap {
		sub := getMapField(msg, key)
		if sub == nil {
			continue
		}

		directPath := getStringField(sub, "directPath")
		if directPath == "" {
			continue
		}

		mediaKeyB64 := getStringField(sub, "mediaKey")
		fileSHA256B64 := getStringField(sub, "fileSHA256")
		fileEncSHA256B64 := getStringField(sub, "fileEncSHA256")

		mediaKey, _ := base64.StdEncoding.DecodeString(mediaKeyB64)
		fileSHA256, _ := base64.StdEncoding.DecodeString(fileSHA256B64)
		fileEncSHA256, _ := base64.StdEncoding.DecodeString(fileEncSHA256B64)

		mimeType := getStringField(sub, "mimetype")
		fileLength := int(getFloatField(sub, "fileLength"))
		fileName := getStringField(sub, "fileName")

		return &mediaInfo{
			DirectPath:    directPath,
			MediaKey:      mediaKey,
			FileSHA256:    fileSHA256,
			FileEncSHA256: fileEncSHA256,
			FileLength:    fileLength,
			MimeType:      mimeType,
			MediaType:     mt,
			FileName:      fileName,
		}
	}

	if dwc := getMapField(msg, "documentWithCaptionMessage"); dwc != nil {
		if innerMsg := getMapField(dwc, "message"); innerMsg != nil {
			return extractMediaInfo(innerMsg)
		}
	}

	return nil
}

var msgTypeKeys = []struct {
	key     string
	msgType string
}{
	{"imageMessage", "image"},
	{"videoMessage", "video"},
	{"audioMessage", "audio"},
	{"documentMessage", "document"},
	{"stickerMessage", "sticker"},
	{"contactMessage", "contact"},
	{"locationMessage", "location"},
	{"listMessage", "list"},
	{"buttonsMessage", "buttons"},
	{"pollCreationMessage", "poll"},
	{"pollCreationMessageV3", "poll"},
	{"documentWithCaptionMessage", "document"},
	{"reactionMessage", "reaction"},
}

func detectMessageType(msg map[string]any) string {
	if msg == nil {
		return "text"
	}
	for _, entry := range msgTypeKeys {
		if _, ok := msg[entry.key]; ok {
			return entry.msgType
		}
	}
	return "text"
}

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
		// Retorna apenas a caption real; o `fileName` NÃO é texto/caption — é
		// o nome do arquivo e deve ser tratado em separado pelo caller (ex.:
		// `document.filename` no webhook Cloud API, ou parâmetro `filename`
		// de `CreateAttachment`). Retorná-lo aqui fazia o Chatwoot exibir o
		// nome do arquivo como se fosse uma mensagem de texto extra.
		return getStringField(docMsg, "caption")
	}

	if dwc := getMapField(msg, "documentWithCaptionMessage"); dwc != nil {
		if innerMsg := getMapField(dwc, "message"); innerMsg != nil {
			if docMsg := getMapField(innerMsg, "documentMessage"); docMsg != nil {
				return getStringField(docMsg, "caption")
			}
		}
	}

	if locMsg := getMapField(msg, "locationMessage"); locMsg != nil {
		return formatLocation(locMsg)
	}

	if contactMsg := getMapField(msg, "contactMessage"); contactMsg != nil {
		if vcard := getStringField(contactMsg, "vcard"); vcard != "" {
			return formatVCard(vcard)
		}
		return getStringField(contactMsg, "displayName")
	}

	if contactsMsg := getMapField(msg, "contactsArrayMessage"); contactsMsg != nil {
		contacts, _ := contactsMsg["contacts"].([]any)
		var parts []string
		for _, c := range contacts {
			if cm, ok := c.(map[string]any); ok {
				displayName := getStringField(cm, "displayName")
				if vcard := getStringField(cm, "vcard"); vcard != "" {
					formatted := formatVCardWithName(vcard, displayName)
					parts = append(parts, formatted)
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n\n")
		}
	}

	if text := extractButtonText(msg); text != "" {
		return text
	}

	if text := extractListText(msg); text != "" {
		return text
	}

	if text := extractTemplateText(msg); text != "" {
		return text
	}

	for _, key := range []string{"ephemeralMessage", "viewOnceMessage", "viewOnceMessageV2", "viewOnceMessageV2Extension", "editedMessage"} {
		if wrapper := getMapField(msg, key); wrapper != nil {
			if inner := getMapField(wrapper, "message"); inner != nil {
				if text := extractText(inner); text != "" {
					return text
				}
			}
		}
	}

	if protocolMsg := getMapField(msg, "protocolMessage"); protocolMsg != nil {
		if editedMsg := getMapField(protocolMsg, "editedMessage"); editedMsg != nil {
			if text := extractText(editedMsg); text != "" {
				return text
			}
		}
	}

	return ""
}

func formatLocation(locMsg map[string]any) string {
	lat := getFloatField(locMsg, "degreesLatitude")
	lng := getFloatField(locMsg, "degreesLongitude")
	name := getStringField(locMsg, "name")
	address := getStringField(locMsg, "address")

	var sb strings.Builder
	sb.WriteString("*Localização:*\n\n")
	fmt.Fprintf(&sb, "_Latitude:_ %f\n", lat)
	fmt.Fprintf(&sb, "_Longitude:_ %f\n", lng)
	if name != "" {
		fmt.Fprintf(&sb, "_Nome:_ %s\n", name)
	}
	if address != "" {
		fmt.Fprintf(&sb, "_Endereço:_ %s\n", address)
	}
	fmt.Fprintf(&sb, "_URL:_ https://www.google.com/maps/search/?api=1&query=%f,%f", lat, lng)
	return sb.String()
}

func findNestedContextInfo(msg map[string]any, depth int) map[string]any {
	if msg == nil || depth > 8 {
		return nil
	}

	if ci := getMapField(msg, "contextInfo"); ci != nil {
		return ci
	}

	for _, key := range []string{
		"extendedTextMessage",
		"imageMessage",
		"videoMessage",
		"audioMessage",
		"documentMessage",
		"stickerMessage",
		"contactMessage",
		"contactsArrayMessage",
		"locationMessage",
		"liveLocationMessage",
		"buttonsResponseMessage",
		"listResponseMessage",
		"templateButtonReplyMessage",
		"pollCreationMessage",
		"pollCreationMessageV3",
		"reactionMessage",
	} {
		if sub := getMapField(msg, key); sub != nil {
			if ci := findNestedContextInfo(sub, depth+1); ci != nil {
				return ci
			}
		}
	}

	for _, key := range []string{
		"documentWithCaptionMessage",
		"ephemeralMessage",
		"viewOnceMessage",
		"viewOnceMessageV2",
		"viewOnceMessageV2Extension",
		"editedMessage",
	} {
		if sub := getMapField(msg, key); sub != nil {
			if ci := findNestedContextInfo(sub, depth+1); ci != nil {
				return ci
			}
			if inner := getMapField(sub, "message"); inner != nil {
				if ci := findNestedContextInfo(inner, depth+1); ci != nil {
					return ci
				}
			}
		}
	}

	if inner := getMapField(msg, "message"); inner != nil {
		if ci := findNestedContextInfo(inner, depth+1); ci != nil {
			return ci
		}
	}

	if protocolMsg := getMapField(msg, "protocolMessage"); protocolMsg != nil {
		if editedMsg := getMapField(protocolMsg, "editedMessage"); editedMsg != nil {
			if ci := findNestedContextInfo(editedMsg, depth+1); ci != nil {
				return ci
			}
		}
	}

	return nil
}

func extractStanzaID(msg map[string]any) string {
	if msg == nil {
		return ""
	}

	if ci := findNestedContextInfo(msg, 0); ci != nil {
		if id := getStringField(ci, "stanzaId"); id != "" {
			return id
		}
		if id := getStringField(ci, "stanzaID"); id != "" {
			return id
		}
	}

	return ""
}

func extractQuoteText(msg map[string]any) string {
	if msg == nil {
		return ""
	}

	ci := findNestedContextInfo(msg, 0)
	if ci == nil {
		return ""
	}

	quoted := getMapField(ci, "quotedMessage")
	if quoted == nil {
		return ""
	}

	if text := strings.TrimSpace(extractText(quoted)); text != "" {
		return text
	}

	return ""
}

func extractLocationFromText(text string) (lat, lng float64, ok bool) {
	if m := googleMapsRegex.FindStringSubmatch(text); m != nil {
		la, err1 := strconv.ParseFloat(m[1], 64)
		ln, err2 := strconv.ParseFloat(m[2], 64)
		if err1 == nil && err2 == nil {
			return la, ln, true
		}
	}
	if m := coordRegex.FindStringSubmatch(text); m != nil {
		la, err1 := strconv.ParseFloat(m[1], 64)
		ln, err2 := strconv.ParseFloat(m[2], 64)
		if err1 == nil && err2 == nil {
			return la, ln, true
		}
	}
	return 0, 0, false
}
