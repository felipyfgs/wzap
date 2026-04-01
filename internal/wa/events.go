package wa

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/types/events"
	"wzap/internal/logger"

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
		logger.Info().
			Str("session", sessionID).
			Str("from", v.Info.Sender.String()).
			Str("chat", v.Info.Chat.String()).
			Str("mid", v.Info.ID).
			Bool("fromMe", v.Info.IsFromMe).
			Msg("Message received")
	case *events.Receipt:
		eventType = model.EventReceipt
		natsData = map[string]interface{}{
			"type":       v.Type,
			"messageIds": v.MessageIDs,
			"from":       v.SourceString(),
			"timestamp":  v.Timestamp.Unix(),
		}
		logger.Debug().
			Str("session", sessionID).
			Str("type", string(v.Type)).
			Str("from", v.SourceString()).
			Msg("Receipt received")
	case *events.Connected:
		eventType = model.EventConnected
		natsData = map[string]interface{}{}
		logger.Info().Str("session", sessionID).Msg("Session connected")
	case *events.Disconnected:
		eventType = model.EventDisconnected
		natsData = map[string]interface{}{}
		logger.Warn().Str("session", sessionID).Msg("Session disconnected")
	case *events.LoggedOut:
		eventType = model.EventLoggedOut
		natsData = map[string]interface{}{
			"reason": v.Reason,
		}
		logger.Warn().Str("session", sessionID).Str("reason", v.Reason.String()).Msg("Session logged out")
	case *events.PairSuccess:
		eventType = model.EventPairSuccess
		natsData = map[string]interface{}{
			"jid": v.ID.String(),
		}
		logger.Info().Str("session", sessionID).Str("jid", v.ID.String()).Msg("Pair success")
	case *events.GroupInfo:
		eventType = model.EventGroupInfo
		natsData = map[string]interface{}{
			"jid":    v.JID.String(),
			"notify": v.Notify,
		}
		logger.Debug().Str("session", sessionID).Str("group", v.JID.String()).Msg("Group info update")
	case *events.Presence:
		eventType = model.EventPresence
		natsData = map[string]interface{}{
			"from":        v.From.String(),
			"unavailable": v.Unavailable,
			"lastSeen":    v.LastSeen,
		}
		logger.Debug().Str("session", sessionID).Str("from", v.From.String()).Msg("Presence update")
	case *events.ChatPresence:
		eventType = model.EventChatPresence
		natsData = map[string]interface{}{
			"chat":  v.Chat.String(),
			"state": v.State,
		}
		logger.Debug().Str("session", sessionID).Str("chat", v.Chat.String()).Str("state", string(v.State)).Msg("Chat presence")
	case *events.CallOffer:
		eventType = model.EventCallOffer
		natsData = map[string]interface{}{
			"callId": v.CallID,
			"from":   v.From.String(),
		}
		logger.Info().Str("session", sessionID).Str("from", v.From.String()).Msg("Incoming call")
	case *events.CallTerminate:
		eventType = model.EventCallTerminate
		natsData = map[string]interface{}{
			"callId": v.CallID,
			"from":   v.From.String(),
		}
		logger.Debug().Str("session", sessionID).Str("from", v.From.String()).Msg("Call terminated")
	case *events.NewsletterJoin:
		eventType = model.EventNewsletterJoin
		natsData = map[string]interface{}{
			"jid": v.ID,
		}
		logger.Debug().Str("session", sessionID).Msg("Newsletter joined")
	case *events.NewsletterLeave:
		eventType = model.EventNewsletterLeave
		natsData = map[string]interface{}{
			"jid": v.ID.String(),
		}
		logger.Debug().Str("session", sessionID).Msg("Newsletter left")
	default:
		return
	}

	payload := map[string]interface{}{
		"eventId":   uuid.NewString(),
		"sessionId": sessionID,
		"event":     eventType,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	for k, v := range natsData {
		payload[k] = v
	}

	bytes, err := json.Marshal(payload)
	if err != nil {
		logger.Error().Err(err).Str("session", sessionID).Msg("Failed to marshal event payload")
		return
	}

	if m.nats != nil {
		if err := m.nats.Publish(m.ctx, "wzap.events."+sessionID, bytes); err != nil {
			logger.Error().Err(err).Str("session", sessionID).Msg("Failed to publish NATS event")
		}
	}

	if m.dispatcher != nil {
		go m.dispatcher.Dispatch(sessionID, eventType, bytes)
	}
}
