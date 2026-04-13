package chatwoot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"wzap/internal/model"
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
	Info    waMessageInfo          `json:"Info"`
	Message map[string]interface{} `json:"Message"`
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

var mediaTypeMap = map[string]string{
	"imageMessage":    "image",
	"videoMessage":    "video",
	"audioMessage":    "audio",
	"documentMessage": "document",
	"stickerMessage":  "sticker",
}

func extractMediaInfo(msg map[string]interface{}) *mediaInfo {
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

func detectMessageType(msg map[string]interface{}) string {
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

func extractText(msg map[string]interface{}) string {
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

	if dwc := getMapField(msg, "documentWithCaptionMessage"); dwc != nil {
		if innerMsg := getMapField(dwc, "message"); innerMsg != nil {
			if docMsg := getMapField(innerMsg, "documentMessage"); docMsg != nil {
				caption := getStringField(docMsg, "caption")
				if caption != "" {
					return caption
				}
				return getStringField(docMsg, "fileName")
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
		contacts, _ := contactsMsg["contacts"].([]interface{})
		var parts []string
		for _, c := range contacts {
			if cm, ok := c.(map[string]interface{}); ok {
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

	return ""
}

func formatLocation(locMsg map[string]interface{}) string {
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

func formatVCard(vcard string) string {
	return formatVCardWithName(vcard, "")
}

func formatVCardWithName(vcard, displayName string) string {
	name := ""
	var phones []string
	for _, line := range splitLines(vcard) {
		if startsWithCI(line, "FN:") {
			name = line[3:]
		} else if startsWithCI(line, "TEL") {
			if idx := lastIndex(line, ":"); idx >= 0 {
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

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func startsWithCI(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		sc := s[i]
		pc := prefix[i]
		if sc >= 'a' && sc <= 'z' {
			sc -= 32
		}
		if pc >= 'a' && pc <= 'z' {
			pc -= 32
		}
		if sc != pc {
			return false
		}
	}
	return true
}

func lastIndex(s, sep string) int {
	idx := -1
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			idx = i
		}
	}
	return idx
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

	if ci := getMapField(msg, "contextInfo"); ci != nil {
		if id := getStringField(ci, "stanzaId"); id != "" {
			return id
		}
		if id := getStringField(ci, "stanzaID"); id != "" {
			return id
		}
	}

	msgTypes := []string{
		"extendedTextMessage",
		"imageMessage",
		"videoMessage",
		"audioMessage",
		"documentMessage",
		"stickerMessage",
	}

	for _, key := range msgTypes {
		sub := getMapField(msg, key)
		if sub == nil {
			continue
		}
		ci := getMapField(sub, "contextInfo")
		if ci == nil {
			continue
		}
		if id := getStringField(ci, "stanzaId"); id != "" {
			return id
		}
		if id := getStringField(ci, "stanzaID"); id != "" {
			return id
		}
	}

	return ""
}

func extractQuoteText(msg map[string]interface{}) string {
	if msg == nil {
		return ""
	}

	msgTypes := []string{
		"extendedTextMessage",
		"imageMessage",
		"videoMessage",
		"audioMessage",
		"documentMessage",
		"stickerMessage",
	}

	for _, key := range msgTypes {
		sub, ok := msg[key].(map[string]interface{})
		if !ok {
			continue
		}
		ci, ok := sub["contextInfo"].(map[string]interface{})
		if !ok {
			continue
		}
		quoted, ok := ci["quotedMessage"].(map[string]interface{})
		if !ok {
			continue
		}
		if conv, ok := quoted["conversation"].(string); ok && conv != "" {
			text := strings.Trim(conv, `"`)
			return strings.TrimSpace(text)
		}
		if extText, ok := quoted["extendedTextMessage"].(map[string]interface{}); ok {
			if text, ok := extText["text"].(string); ok && text != "" {
				return strings.TrimSpace(text)
			}
		}
	}

	return ""
}
