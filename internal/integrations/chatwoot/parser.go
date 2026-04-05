package chatwoot

import (
	"encoding/json"
	"fmt"

	"wzap/internal/model"
)

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
