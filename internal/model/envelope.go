package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type SessionInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type EventEnvelope struct {
	Event     string          `json:"event"`
	EventID   string          `json:"eventId"`
	Session   SessionInfo     `json:"session"`
	Timestamp string          `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

func BuildEventEnvelope(sessionID, sessionName string, event EventType, data any) ([]byte, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	envelope := EventEnvelope{
		Event:     string(event),
		EventID:   uuid.NewString(),
		Session:   SessionInfo{ID: sessionID, Name: sessionName},
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      dataBytes,
	}

	return json.Marshal(envelope)
}

func BuildEventEnvelopeFromRaw(sessionID, sessionName string, event EventType, rawData json.RawMessage) ([]byte, error) {
	envelope := EventEnvelope{
		Event:     string(event),
		EventID:   uuid.NewString(),
		Session:   SessionInfo{ID: sessionID, Name: sessionName},
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      rawData,
	}

	return json.Marshal(envelope)
}

func ParseEventEnvelope(payload []byte) (*EventEnvelope, error) {
	var envelope EventEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, err
	}
	return &envelope, nil
}
