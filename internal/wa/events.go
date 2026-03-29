package wa

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.mau.fi/whatsmeow/types/events"
)

// handleEvent publishes whatsmeow events to NATS.
func (m *Manager) handleEvent(sessionID string, evt interface{}) {
	var natsData map[string]interface{}
	eventType := ""

	switch v := evt.(type) {
	case *events.Message:
		eventType = "messages.upsert"
		natsData = map[string]interface{}{
			"id":        v.Info.ID,
			"pushName":  v.Info.PushName,
			"message":   v.Message,
			"timestamp": v.Info.Timestamp.Unix(),
			"fromMe":    v.Info.IsFromMe,
		}
	default:
		return
	}

	payload := map[string]interface{}{
		"event_id":   uuid.NewString(),
		"session_id": sessionID,
		"event":      eventType,
		"data":       natsData,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	bytes, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Str("session", sessionID).Msg("Failed to marshal NATS event payload")
		return
	}
	if m.nats != nil {
		if err := m.nats.Publish(context.Background(), "wzap.events."+sessionID, bytes); err != nil {
			log.Error().Err(err).Str("session", sessionID).Msg("Failed to publish NATS event")
		}
	}
}
