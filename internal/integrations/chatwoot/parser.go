package chatwoot

import (
	"encoding/json"
	"fmt"
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
