package wa

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"

	"wzap/internal/logger"
	"wzap/internal/model"
)

func (m *Manager) handleEvent(sessionID string, evt interface{}) {
	var natsData map[string]interface{}
	var eventType model.EventType

	switch v := evt.(type) {

	// ── Messages ──────────────────────────────────────────────
	case *events.Message:
		eventType = model.EventMessage
		natsData = map[string]interface{}{
			"id":        v.Info.ID,
			"pushName":  v.Info.PushName,
			"message":   v.Message,
			"timestamp": v.Info.Timestamp.Unix(),
			"fromMe":    v.Info.IsFromMe,
			"sender":    v.Info.Sender.String(),
			"chat":      v.Info.Chat.String(),
		}
		logger.Info().
			Str("session", sessionID).
			Str("from", v.Info.Sender.String()).
			Str("chat", v.Info.Chat.String()).
			Str("mid", v.Info.ID).
			Bool("fromMe", v.Info.IsFromMe).
			Msg("Message received")

		if m.OnMediaReceived != nil && v.Message != nil {
			switch {
			case v.Message.GetImageMessage() != nil:
				m.OnMediaReceived(sessionID, v.Info.ID, v.Message.GetImageMessage().GetMimetype(), v.Message.GetImageMessage())
			case v.Message.GetVideoMessage() != nil:
				m.OnMediaReceived(sessionID, v.Info.ID, v.Message.GetVideoMessage().GetMimetype(), v.Message.GetVideoMessage())
			case v.Message.GetAudioMessage() != nil:
				m.OnMediaReceived(sessionID, v.Info.ID, v.Message.GetAudioMessage().GetMimetype(), v.Message.GetAudioMessage())
			case v.Message.GetDocumentMessage() != nil:
				m.OnMediaReceived(sessionID, v.Info.ID, v.Message.GetDocumentMessage().GetMimetype(), v.Message.GetDocumentMessage())
			case v.Message.GetStickerMessage() != nil:
				m.OnMediaReceived(sessionID, v.Info.ID, v.Message.GetStickerMessage().GetMimetype(), v.Message.GetStickerMessage())
			}
		}

		if m.OnMessageReceived != nil {
			msgType, body, mediaType := extractMessageContent(v.Message)
			m.OnMessageReceived(sessionID, v.Info.ID, v.Info.Chat.String(), v.Info.Sender.String(), v.Info.IsFromMe, msgType, body, mediaType, v.Info.Timestamp.Unix(), v.Message)
		}

	case *events.UndecryptableMessage:
		eventType = model.EventUndecryptableMessage
		natsData = map[string]interface{}{
			"id":              v.Info.ID,
			"sender":          v.Info.Sender.String(),
			"chat":            v.Info.Chat.String(),
			"isUnavailable":   v.IsUnavailable,
			"unavailableType": string(v.UnavailableType),
			"timestamp":       v.Info.Timestamp.Unix(),
		}
		logger.Warn().Str("session", sessionID).Str("mid", v.Info.ID).Msg("Undecryptable message")

	case *events.MediaRetry:
		eventType = model.EventMediaRetry
		natsData = map[string]interface{}{
			"messageId": string(v.MessageID),
			"chatId":    v.ChatID.String(),
			"timestamp": v.Timestamp.Unix(),
		}
		if v.Error != nil {
			natsData["errorCode"] = v.Error.Code
		}
		logger.Debug().Str("session", sessionID).Msg("Media retry")

	case *events.Receipt:
		eventType = model.EventReceipt
		natsData = map[string]interface{}{
			"type":       v.Type,
			"messageIds": v.MessageIDs,
			"from":       v.SourceString(),
			"timestamp":  v.Timestamp.Unix(),
		}
		logger.Debug().Str("session", sessionID).Str("type", string(v.Type)).Msg("Receipt received")

	case *events.DeleteForMe:
		eventType = model.EventDeleteForMe
		natsData = map[string]interface{}{
			"chatJid":   v.ChatJID.String(),
			"senderJid": v.SenderJID.String(),
			"isFromMe":  v.IsFromMe,
			"messageId": v.MessageID,
			"timestamp": v.Timestamp.Unix(),
		}
		logger.Debug().Str("session", sessionID).Str("mid", v.MessageID).Msg("Message deleted for me")

	// ── Connection / Session lifecycle ────────────────────────
	case *events.Connected:
		eventType = model.EventConnected
		natsData = map[string]interface{}{}
		logger.Info().Str("session", sessionID).Msg("Session connected")

	case *events.Disconnected:
		eventType = model.EventDisconnected
		natsData = map[string]interface{}{}
		logger.Warn().Str("session", sessionID).Msg("Session disconnected")

	case *events.ConnectFailure:
		eventType = model.EventConnectFailure
		natsData = map[string]interface{}{
			"reason":  int(v.Reason),
			"message": v.Message,
		}
		logger.Error().Str("session", sessionID).Int("reason", int(v.Reason)).Msg("Connect failure")

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

	case *events.PairError:
		eventType = model.EventPairError
		natsData = map[string]interface{}{
			"id":       v.ID.String(),
			"platform": v.Platform,
		}
		if v.Error != nil {
			natsData["error"] = v.Error.Error()
		}
		logger.Error().Str("session", sessionID).Msg("Pair error")

	case *events.StreamError:
		eventType = model.EventStreamError
		natsData = map[string]interface{}{
			"code": v.Code,
		}
		logger.Error().Str("session", sessionID).Str("code", v.Code).Msg("Stream error")

	case *events.StreamReplaced:
		eventType = model.EventStreamReplaced
		natsData = map[string]interface{}{}
		logger.Warn().Str("session", sessionID).Msg("Stream replaced")

	case *events.KeepAliveTimeout:
		eventType = model.EventKeepAliveTimeout
		natsData = map[string]interface{}{
			"errorCount":  v.ErrorCount,
			"lastSuccess": v.LastSuccess.Format(time.RFC3339),
		}
		logger.Warn().Str("session", sessionID).Int("errorCount", v.ErrorCount).Msg("Keep-alive timeout")

	case *events.KeepAliveRestored:
		eventType = model.EventKeepAliveRestored
		natsData = map[string]interface{}{}
		logger.Info().Str("session", sessionID).Msg("Keep-alive restored")

	case *events.ClientOutdated:
		eventType = model.EventClientOutdated
		natsData = map[string]interface{}{}
		logger.Error().Str("session", sessionID).Msg("Client outdated")

	case *events.TemporaryBan:
		eventType = model.EventTemporaryBan
		natsData = map[string]interface{}{
			"code":   int(v.Code),
			"expire": v.Expire.String(),
		}
		logger.Error().Str("session", sessionID).Str("expire", v.Expire.String()).Msg("Temporary ban")

	case *events.CATRefreshError:
		eventType = model.EventCATRefreshError
		natsData = map[string]interface{}{}
		if v.Error != nil {
			natsData["error"] = v.Error.Error()
		}
		logger.Error().Str("session", sessionID).Msg("CAT refresh error")

	case *events.ManualLoginReconnect:
		eventType = model.EventManualLoginReconnect
		natsData = map[string]interface{}{}
		logger.Info().Str("session", sessionID).Msg("Manual login reconnect")

	// ── Contacts & Identity ──────────────────────────────────
	case *events.Contact:
		eventType = model.EventContact
		natsData = map[string]interface{}{
			"jid":       v.JID.String(),
			"timestamp": v.Timestamp.Unix(),
		}
		if v.Action != nil {
			natsData["fullName"] = v.Action.GetFullName()
			natsData["firstName"] = v.Action.GetFirstName()
		}
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Contact changed")

	case *events.PushName:
		eventType = model.EventPushName
		natsData = map[string]interface{}{
			"jid":         v.JID.String(),
			"oldPushName": v.OldPushName,
			"newPushName": v.NewPushName,
		}
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Push name changed")

	case *events.BusinessName:
		eventType = model.EventBusinessName
		natsData = map[string]interface{}{
			"jid":             v.JID.String(),
			"oldBusinessName": v.OldBusinessName,
			"newBusinessName": v.NewBusinessName,
		}
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Business name changed")

	case *events.Picture:
		eventType = model.EventPicture
		natsData = map[string]interface{}{
			"jid":       v.JID.String(),
			"author":    v.Author.String(),
			"timestamp": v.Timestamp.Unix(),
			"remove":    v.Remove,
			"pictureId": v.PictureID,
		}
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Picture changed")

	case *events.IdentityChange:
		eventType = model.EventIdentityChange
		natsData = map[string]interface{}{
			"jid":       v.JID.String(),
			"timestamp": v.Timestamp.Unix(),
			"implicit":  v.Implicit,
		}
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Identity changed")

	case *events.UserAbout:
		eventType = model.EventUserAbout
		natsData = map[string]interface{}{
			"jid":       v.JID.String(),
			"status":    v.Status,
			"timestamp": v.Timestamp.Unix(),
		}
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("User about changed")

	// ── Groups ───────────────────────────────────────────────
	case *events.GroupInfo:
		eventType = model.EventGroupInfo
		natsData = map[string]interface{}{
			"jid":    v.JID.String(),
			"notify": v.Notify,
		}
		logger.Debug().Str("session", sessionID).Str("group", v.JID.String()).Msg("Group info update")

	case *events.JoinedGroup:
		eventType = model.EventJoinedGroup
		natsData = map[string]interface{}{
			"jid":    v.JID.String(),
			"name":   v.Name,
			"reason": v.Reason,
			"type":   v.Type,
		}
		if v.Sender != nil {
			natsData["sender"] = v.Sender.String()
		}
		logger.Info().Str("session", sessionID).Str("group", v.JID.String()).Msg("Joined group")

	// ── Presence ─────────────────────────────────────────────
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
		logger.Debug().Str("session", sessionID).Str("chat", v.Chat.String()).Msg("Chat presence")

	// ── Chat state ───────────────────────────────────────────
	case *events.Archive:
		eventType = model.EventArchive
		natsData = map[string]interface{}{
			"jid":       v.JID.String(),
			"timestamp": v.Timestamp.Unix(),
		}
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Chat archived")

	case *events.Mute:
		eventType = model.EventMute
		natsData = map[string]interface{}{
			"jid":       v.JID.String(),
			"timestamp": v.Timestamp.Unix(),
		}
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Chat muted")

	case *events.Pin:
		eventType = model.EventPin
		natsData = map[string]interface{}{
			"jid":       v.JID.String(),
			"timestamp": v.Timestamp.Unix(),
		}
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Chat pinned")

	case *events.Star:
		eventType = model.EventStar
		natsData = map[string]interface{}{
			"chatJid":   v.ChatJID.String(),
			"senderJid": v.SenderJID.String(),
			"isFromMe":  v.IsFromMe,
			"messageId": v.MessageID,
			"timestamp": v.Timestamp.Unix(),
		}
		logger.Debug().Str("session", sessionID).Str("mid", v.MessageID).Msg("Message starred")

	case *events.ClearChat:
		eventType = model.EventClearChat
		natsData = map[string]interface{}{
			"jid":         v.JID.String(),
			"timestamp":   v.Timestamp.Unix(),
			"deleteMedia": v.DeleteMedia,
		}
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Chat cleared")

	case *events.DeleteChat:
		eventType = model.EventDeleteChat
		natsData = map[string]interface{}{
			"jid":         v.JID.String(),
			"timestamp":   v.Timestamp.Unix(),
			"deleteMedia": v.DeleteMedia,
		}
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Chat deleted")

	case *events.MarkChatAsRead:
		eventType = model.EventMarkChatAsRead
		natsData = map[string]interface{}{
			"jid":       v.JID.String(),
			"timestamp": v.Timestamp.Unix(),
		}
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Chat marked as read")

	case *events.UnarchiveChatsSetting:
		eventType = model.EventUnarchiveChatsSetting
		natsData = map[string]interface{}{
			"timestamp": v.Timestamp.Unix(),
		}
		logger.Debug().Str("session", sessionID).Msg("Unarchive chats setting changed")

	// ── Labels ───────────────────────────────────────────────
	case *events.LabelEdit:
		eventType = model.EventLabelEdit
		natsData = map[string]interface{}{
			"labelId":   v.LabelID,
			"timestamp": v.Timestamp.Unix(),
		}
		if v.Action != nil {
			natsData["name"] = v.Action.GetName()
			natsData["color"] = v.Action.GetColor()
			natsData["deleted"] = v.Action.GetDeleted()
		}
		logger.Debug().Str("session", sessionID).Str("labelId", v.LabelID).Msg("Label edited")

	case *events.LabelAssociationChat:
		eventType = model.EventLabelAssociationChat
		natsData = map[string]interface{}{
			"jid":       v.JID.String(),
			"labelId":   v.LabelID,
			"timestamp": v.Timestamp.Unix(),
		}
		logger.Debug().Str("session", sessionID).Str("labelId", v.LabelID).Msg("Label association chat")

	case *events.LabelAssociationMessage:
		eventType = model.EventLabelAssociationMessage
		natsData = map[string]interface{}{
			"jid":       v.JID.String(),
			"labelId":   v.LabelID,
			"messageId": v.MessageID,
			"timestamp": v.Timestamp.Unix(),
		}
		logger.Debug().Str("session", sessionID).Str("labelId", v.LabelID).Msg("Label association message")

	// ── Calls ────────────────────────────────────────────────
	case *events.CallOffer:
		eventType = model.EventCallOffer
		natsData = map[string]interface{}{
			"callId": v.CallID,
			"from":   v.From.String(),
		}
		logger.Info().Str("session", sessionID).Str("from", v.From.String()).Msg("Incoming call")

	case *events.CallAccept:
		eventType = model.EventCallAccept
		natsData = map[string]interface{}{
			"callId": v.CallID,
			"from":   v.From.String(),
		}
		logger.Debug().Str("session", sessionID).Msg("Call accepted")

	case *events.CallTerminate:
		eventType = model.EventCallTerminate
		natsData = map[string]interface{}{
			"callId": v.CallID,
			"from":   v.From.String(),
			"reason": v.Reason,
		}
		logger.Debug().Str("session", sessionID).Msg("Call terminated")

	case *events.CallOfferNotice:
		eventType = model.EventCallOfferNotice
		natsData = map[string]interface{}{
			"callId": v.CallID,
			"from":   v.From.String(),
			"media":  v.Media,
			"type":   v.Type,
		}
		logger.Debug().Str("session", sessionID).Msg("Call offer notice")

	case *events.CallRelayLatency:
		eventType = model.EventCallRelayLatency
		natsData = map[string]interface{}{
			"callId": v.CallID,
			"from":   v.From.String(),
		}
		logger.Debug().Str("session", sessionID).Msg("Call relay latency")

	case *events.CallPreAccept:
		eventType = model.EventCallPreAccept
		natsData = map[string]interface{}{
			"callId": v.CallID,
			"from":   v.From.String(),
		}
		logger.Debug().Str("session", sessionID).Msg("Call pre-accept")

	case *events.CallReject:
		eventType = model.EventCallReject
		natsData = map[string]interface{}{
			"callId": v.CallID,
			"from":   v.From.String(),
		}
		logger.Debug().Str("session", sessionID).Msg("Call rejected")

	case *events.CallTransport:
		eventType = model.EventCallTransport
		natsData = map[string]interface{}{
			"callId": v.CallID,
			"from":   v.From.String(),
		}
		logger.Debug().Str("session", sessionID).Msg("Call transport")

	case *events.UnknownCallEvent:
		eventType = model.EventUnknownCallEvent
		natsData = map[string]interface{}{}
		logger.Debug().Str("session", sessionID).Msg("Unknown call event")

	// ── Newsletter ───────────────────────────────────────────
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

	case *events.NewsletterMuteChange:
		eventType = model.EventNewsletterMuteChange
		natsData = map[string]interface{}{
			"jid":  v.ID.String(),
			"mute": string(v.Mute),
		}
		logger.Debug().Str("session", sessionID).Msg("Newsletter mute changed")

	case *events.NewsletterLiveUpdate:
		eventType = model.EventNewsletterLiveUpdate
		natsData = map[string]interface{}{
			"jid":          v.JID.String(),
			"time":         v.Time.Format(time.RFC3339),
			"messageCount": len(v.Messages),
		}
		logger.Debug().Str("session", sessionID).Msg("Newsletter live update")

	// ── Sync ─────────────────────────────────────────────────
	case *events.HistorySync:
		eventType = model.EventHistorySync
		natsData = map[string]interface{}{}
		if v.Data != nil {
			natsData["syncType"] = v.Data.GetSyncType().String()
			natsData["chunkOrder"] = v.Data.GetChunkOrder()
			natsData["progress"] = v.Data.GetProgress()
		}
		logger.Info().Str("session", sessionID).Msg("History sync received")

	case *events.AppStateSyncComplete:
		eventType = model.EventAppStateSyncComplete
		natsData = map[string]interface{}{
			"name":    string(v.Name),
			"version": v.Version,
		}
		logger.Debug().Str("session", sessionID).Msg("App state sync complete")

	case *events.OfflineSyncCompleted:
		eventType = model.EventOfflineSyncCompleted
		natsData = map[string]interface{}{
			"count": v.Count,
		}
		logger.Info().Str("session", sessionID).Int("count", v.Count).Msg("Offline sync completed")

	case *events.OfflineSyncPreview:
		eventType = model.EventOfflineSyncPreview
		natsData = map[string]interface{}{
			"total":          v.Total,
			"appDataChanges": v.AppDataChanges,
			"messages":       v.Messages,
			"notifications":  v.Notifications,
			"receipts":       v.Receipts,
		}
		logger.Debug().Str("session", sessionID).Int("total", v.Total).Msg("Offline sync preview")

	// ── Privacy & Settings ───────────────────────────────────
	case *events.PrivacySettings:
		eventType = model.EventPrivacySettings
		natsData = map[string]interface{}{
			"groupAddChanged":     v.GroupAddChanged,
			"lastSeenChanged":     v.LastSeenChanged,
			"statusChanged":       v.StatusChanged,
			"profileChanged":      v.ProfileChanged,
			"readReceiptsChanged": v.ReadReceiptsChanged,
			"onlineChanged":       v.OnlineChanged,
			"callAddChanged":      v.CallAddChanged,
		}
		logger.Debug().Str("session", sessionID).Msg("Privacy settings changed")

	case *events.PushNameSetting:
		eventType = model.EventPushNameSetting
		natsData = map[string]interface{}{
			"timestamp": v.Timestamp.Unix(),
		}
		if v.Action != nil {
			natsData["name"] = v.Action.GetName()
		}
		logger.Debug().Str("session", sessionID).Msg("Push name setting changed")

	case *events.UserStatusMute:
		eventType = model.EventUserStatusMute
		natsData = map[string]interface{}{
			"jid":       v.JID.String(),
			"timestamp": v.Timestamp.Unix(),
		}
		logger.Debug().Str("session", sessionID).Msg("User status mute changed")

	case *events.BlocklistChange:
		eventType = model.EventBlocklistChange
		natsData = map[string]interface{}{
			"jid":    v.JID.String(),
			"action": string(v.Action),
		}
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Blocklist changed")

	case *events.Blocklist:
		eventType = model.EventBlocklist
		natsData = map[string]interface{}{
			"action": string(v.Action),
		}
		if len(v.Changes) > 0 {
			changes := make([]map[string]string, len(v.Changes))
			for i, c := range v.Changes {
				changes[i] = map[string]string{
					"jid":    c.JID.String(),
					"action": string(c.Action),
				}
			}
			natsData["changes"] = changes
		}
		logger.Debug().Str("session", sessionID).Msg("Blocklist received")

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

func extractMessageContent(msg *waE2E.Message) (msgType, body, mediaType string) {
	if msg == nil {
		return "unknown", "", ""
	}
	switch {
	case msg.GetConversation() != "":
		return "text", msg.GetConversation(), ""
	case msg.GetExtendedTextMessage() != nil:
		return "text", msg.GetExtendedTextMessage().GetText(), ""
	case msg.GetImageMessage() != nil:
		return "image", msg.GetImageMessage().GetCaption(), msg.GetImageMessage().GetMimetype()
	case msg.GetVideoMessage() != nil:
		return "video", msg.GetVideoMessage().GetCaption(), msg.GetVideoMessage().GetMimetype()
	case msg.GetAudioMessage() != nil:
		return "audio", "", msg.GetAudioMessage().GetMimetype()
	case msg.GetDocumentMessage() != nil:
		return "document", msg.GetDocumentMessage().GetFileName(), msg.GetDocumentMessage().GetMimetype()
	case msg.GetStickerMessage() != nil:
		return "sticker", "", msg.GetStickerMessage().GetMimetype()
	case msg.GetContactMessage() != nil:
		return "contact", msg.GetContactMessage().GetDisplayName(), ""
	case msg.GetLocationMessage() != nil:
		return "location", msg.GetLocationMessage().GetName(), ""
	case msg.GetListMessage() != nil:
		return "list", msg.GetListMessage().GetTitle(), ""
	case msg.GetButtonsMessage() != nil:
		return "buttons", msg.GetButtonsMessage().GetContentText(), ""
	case msg.GetPollCreationMessage() != nil:
		return "poll", msg.GetPollCreationMessage().GetName(), ""
	default:
		return "unknown", "", ""
	}
}
