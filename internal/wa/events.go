package wa

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.mau.fi/whatsmeow/types/events"

	"wzap/internal/model"
)

// handleEvent publishes whatsmeow events to NATS.
func (m *Manager) handleEvent(sessionID string, evt interface{}) {
	var natsData map[string]interface{}
	var eventType model.EventType

	switch v := evt.(type) {
	case *events.Message:
		eventType = model.EventMessage
		natsData = map[string]interface{}{
			"id":        v.Info.ID,
			"pushName":  v.Info.PushName,
			"message":   v.Message,
			"timestamp": v.Info.Timestamp.Unix(),
			"fromMe":    v.Info.IsFromMe,
		}
	case *events.Receipt:
		eventType = model.EventReceipt
		natsData = map[string]interface{}{
			"type":      v.Type,
			"messageId": v.MessageIDs,
			"from":      v.SourceString(),
			"timestamp": v.Timestamp.Unix(),
		}
	case *events.Connected:
		eventType = model.EventConnected
		natsData = map[string]interface{}{}
	case *events.Disconnected:
		eventType = model.EventDisconnected
		natsData = map[string]interface{}{}
	case *events.LoggedOut:
		eventType = model.EventLoggedOut
		natsData = map[string]interface{}{
			"reason": v.Reason,
		}
	case *events.PairSuccess:
		eventType = model.EventPairSuccess
		natsData = map[string]interface{}{
			"jid": v.ID.String(),
		}
	case *events.GroupInfo:
		eventType = model.EventGroupInfo
		natsData = map[string]interface{}{
			"jid":    v.JID.String(),
			"notify": v.Notify,
		}
	case *events.Presence:
		eventType = model.EventPresence
		natsData = map[string]interface{}{
			"from":        v.From.String(),
			"unavailable": v.Unavailable,
			"lastSeen":    v.LastSeen,
		}
	case *events.ChatPresence:
		eventType = model.EventChatPresence
		natsData = map[string]interface{}{
			"chat":  v.Chat.String(),
			"state": v.State,
		}
	case *events.CallOffer:
		eventType = model.EventCallOffer
		natsData = map[string]interface{}{
			"callId": v.CallID,
			"from":   v.From.String(),
		}
	case *events.CallTerminate:
		eventType = model.EventCallTerminate
		natsData = map[string]interface{}{
			"callId": v.CallID,
			"from":   v.From.String(),
		}
	case *events.NewsletterJoin:
		eventType = model.EventNewsletterJoin
		natsData = map[string]interface{}{
			"jid": v.NewsletterMetadata.ID,
		}
	case *events.NewsletterLeave:
		eventType = model.EventNewsletterLeave
		natsData = map[string]interface{}{
			"jid": v.ID.String(),
		}
	default:
		return
	}

	payload := map[string]interface{}{
		"eventId":   uuid.NewString(),
		"sessionId": sessionID,
		"event":      eventType,
		"timestamp":  time.Now().Format(time.RFC3339),
	}
	for k, v := range natsData {
		payload[k] = v
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
