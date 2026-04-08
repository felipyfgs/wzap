package wa

import (
	"encoding/json"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"

	"wzap/internal/logger"
	"wzap/internal/model"
)

func (m *Manager) handleEvent(sessionID string, evt interface{}) {
	var eventType model.EventType

	switch v := evt.(type) {

	// ── Messages ──────────────────────────────────────────────
	case *events.Message:
		if proto := v.Message.GetProtocolMessage(); proto != nil &&
			proto.GetType() == waE2E.ProtocolMessage_REVOKE &&
			proto.GetKey() != nil && proto.GetKey().GetID() != "" {
			eventType = model.EventMessageRevoke
			logger.Debug().
				Str("session", sessionID).
				Str("revokedMsgID", proto.GetKey().GetID()).
				Str("chat", v.Info.Chat.String()).
				Bool("fromMe", v.Info.IsFromMe).
				Msg("Message revoked (delete for everyone)")
		} else if proto := v.Message.GetProtocolMessage(); proto != nil &&
			proto.GetType() == waE2E.ProtocolMessage_MESSAGE_EDIT &&
			proto.GetKey() != nil && proto.GetKey().GetID() != "" &&
			proto.GetEditedMessage() != nil {
			eventType = model.EventMessageEdit
			logger.Debug().
				Str("session", sessionID).
				Str("editedMsgID", proto.GetKey().GetID()).
				Str("chat", v.Info.Chat.String()).
				Bool("fromMe", v.Info.IsFromMe).
				Msg("Message edited")
		} else {
			eventType = model.EventMessage
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
		}

	case *events.UndecryptableMessage:
		eventType = model.EventUndecryptableMessage
		logger.Warn().Str("session", sessionID).Str("mid", v.Info.ID).Msg("Undecryptable message")

	case *events.MediaRetry:
		eventType = model.EventMediaRetry
		logger.Debug().Str("session", sessionID).Msg("Media retry")

	case *events.Receipt:
		eventType = model.EventReceipt
		if v.Type != "" && v.Type != "read" && v.Type != "read-self" && v.Type != "played" && v.Type != "played-self" {
			logger.Debug().Str("session", sessionID).Str("type", string(v.Type)).Msg("Receipt ignored")
			return
		}
		logger.Debug().Str("session", sessionID).Str("type", string(v.Type)).Msg("Receipt received")

	case *events.DeleteForMe:
		eventType = model.EventDeleteForMe
		logger.Debug().Str("session", sessionID).Str("mid", v.MessageID).Msg("Message deleted for me")

	// ── Connection ────────────────────────────────────────────
	case *events.Connected:
		eventType = model.EventConnected
		logger.Info().Str("session", sessionID).Msg("Session connected")

	case *events.Disconnected:
		eventType = model.EventDisconnected
		logger.Warn().Str("session", sessionID).Msg("Session disconnected")

	case *events.ManualLoginReconnect:
		eventType = model.EventManualLoginReconnect
		logger.Info().Str("session", sessionID).Msg("Manual login reconnect")

	// ── Pairing ───────────────────────────────────────────────
	case *events.QR:
		eventType = model.EventQR
		logger.Debug().Str("session", sessionID).Int("codes", len(v.Codes)).Msg("QR codes received")

	case *events.QRScannedWithoutMultidevice:
		eventType = model.EventQRScannedWithoutMultidevice
		logger.Warn().Str("session", sessionID).Msg("QR scanned without multidevice")

	case *events.PairSuccess:
		eventType = model.EventPairSuccess
		logger.Info().Str("session", sessionID).Str("jid", v.ID.String()).Msg("Pair success")

	case *events.PairError:
		eventType = model.EventPairError
		logger.Error().Str("session", sessionID).Msg("Pair error")

	// ── Connection Errors ─────────────────────────────────────
	case *events.ConnectFailure:
		eventType = model.EventConnectFailure
		logger.Error().Str("session", sessionID).Int("reason", int(v.Reason)).Msg("Connect failure")

	case *events.LoggedOut:
		eventType = model.EventLoggedOut
		logger.Warn().Str("session", sessionID).Str("reason", v.Reason.String()).Msg("Session logged out")

	case *events.StreamError:
		eventType = model.EventStreamError
		logger.Error().Str("session", sessionID).Str("code", v.Code).Msg("Stream error")

	case *events.StreamReplaced:
		eventType = model.EventStreamReplaced
		logger.Warn().Str("session", sessionID).Msg("Stream replaced")

	case *events.KeepAliveTimeout:
		eventType = model.EventKeepAliveTimeout
		logger.Warn().Str("session", sessionID).Int("errorCount", v.ErrorCount).Msg("Keep-alive timeout")

	case *events.KeepAliveRestored:
		eventType = model.EventKeepAliveRestored
		logger.Info().Str("session", sessionID).Msg("Keep-alive restored")

	case *events.ClientOutdated:
		eventType = model.EventClientOutdated
		logger.Error().Str("session", sessionID).Msg("Client outdated")

	case *events.TemporaryBan:
		eventType = model.EventTemporaryBan
		logger.Error().Str("session", sessionID).Str("expire", v.Expire.String()).Msg("Temporary ban")

	case *events.CATRefreshError:
		eventType = model.EventCATRefreshError
		logger.Error().Str("session", sessionID).Msg("CAT refresh error")

	// ── Contacts ──────────────────────────────────────────────
	case *events.Contact:
		eventType = model.EventContact
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Contact changed")

	case *events.PushName:
		eventType = model.EventPushName
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Push name changed")

	case *events.BusinessName:
		eventType = model.EventBusinessName
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Business name changed")

	// ── Profile & Identity ────────────────────────────────────
	case *events.Picture:
		eventType = model.EventPicture
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Picture changed")

	case *events.IdentityChange:
		eventType = model.EventIdentityChange
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Identity changed")

	case *events.UserAbout:
		eventType = model.EventUserAbout
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("User about changed")

	// ── Groups ────────────────────────────────────────────────
	case *events.GroupInfo:
		eventType = model.EventGroupInfo
		logger.Debug().Str("session", sessionID).Str("group", v.JID.String()).Msg("Group info update")

	case *events.JoinedGroup:
		eventType = model.EventJoinedGroup
		logger.Info().Str("session", sessionID).Str("group", v.JID.String()).Msg("Joined group")

	// ── Presence ──────────────────────────────────────────────
	case *events.Presence:
		eventType = model.EventPresence
		logger.Debug().Str("session", sessionID).Str("jid", v.From.String()).Bool("unavailable", v.Unavailable).Msg("Presence update")

	case *events.ChatPresence:
		eventType = model.EventChatPresence
		logger.Debug().Str("session", sessionID).Str("chat", v.Chat.String()).Msg("Chat presence")

	// ── Chat State ────────────────────────────────────────────
	case *events.Archive:
		eventType = model.EventArchive
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Chat archived")

	case *events.Mute:
		eventType = model.EventMute
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Chat muted")

	case *events.Pin:
		eventType = model.EventPin
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Chat pinned")

	case *events.Star:
		eventType = model.EventStar
		logger.Debug().Str("session", sessionID).Str("mid", v.MessageID).Msg("Message starred")

	case *events.ClearChat:
		eventType = model.EventClearChat
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Chat cleared")

	case *events.DeleteChat:
		eventType = model.EventDeleteChat
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Chat deleted")

	case *events.MarkChatAsRead:
		eventType = model.EventMarkChatAsRead
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Chat marked as read")

	case *events.UnarchiveChatsSetting:
		eventType = model.EventUnarchiveChatsSetting
		logger.Debug().Str("session", sessionID).Msg("Unarchive chats setting changed")

	// ── Labels ────────────────────────────────────────────────
	case *events.LabelEdit:
		eventType = model.EventLabelEdit
		logger.Debug().Str("session", sessionID).Str("labelId", v.LabelID).Msg("Label edited")

	case *events.LabelAssociationChat:
		eventType = model.EventLabelAssociationChat
		logger.Debug().Str("session", sessionID).Str("labelId", v.LabelID).Msg("Label association chat")

	case *events.LabelAssociationMessage:
		eventType = model.EventLabelAssociationMessage
		logger.Debug().Str("session", sessionID).Str("labelId", v.LabelID).Msg("Label association message")

	// ── Calls ─────────────────────────────────────────────────
	case *events.CallOffer:
		eventType = model.EventCallOffer
		logger.Info().Str("session", sessionID).Str("from", v.From.String()).Msg("Incoming call")

	case *events.CallAccept:
		eventType = model.EventCallAccept
		logger.Debug().Str("session", sessionID).Msg("Call accepted")

	case *events.CallTerminate:
		eventType = model.EventCallTerminate
		logger.Debug().Str("session", sessionID).Msg("Call terminated")

	case *events.CallOfferNotice:
		eventType = model.EventCallOfferNotice
		logger.Debug().Str("session", sessionID).Msg("Call offer notice")

	case *events.CallRelayLatency:
		eventType = model.EventCallRelayLatency
		logger.Debug().Str("session", sessionID).Msg("Call relay latency")

	case *events.CallPreAccept:
		eventType = model.EventCallPreAccept
		logger.Debug().Str("session", sessionID).Msg("Call pre-accept")

	case *events.CallReject:
		eventType = model.EventCallReject
		logger.Debug().Str("session", sessionID).Msg("Call rejected")

	case *events.CallTransport:
		eventType = model.EventCallTransport
		logger.Debug().Str("session", sessionID).Msg("Call transport")

	case *events.UnknownCallEvent:
		eventType = model.EventUnknownCallEvent
		logger.Debug().Str("session", sessionID).Msg("Unknown call event")

	// ── Newsletter ────────────────────────────────────────────
	case *events.NewsletterJoin:
		eventType = model.EventNewsletterJoin
		logger.Debug().Str("session", sessionID).Msg("Newsletter joined")

	case *events.NewsletterLeave:
		eventType = model.EventNewsletterLeave
		logger.Debug().Str("session", sessionID).Msg("Newsletter left")

	case *events.NewsletterMuteChange:
		eventType = model.EventNewsletterMuteChange
		logger.Debug().Str("session", sessionID).Msg("Newsletter mute changed")

	case *events.NewsletterLiveUpdate:
		eventType = model.EventNewsletterLiveUpdate
		logger.Debug().Str("session", sessionID).Msg("Newsletter live update")

	// ── Sync ──────────────────────────────────────────────────
	case *events.HistorySync:
		eventType = model.EventHistorySync
		logger.Info().Str("session", sessionID).Msg("History sync received")
		if m.OnHistorySyncReceived != nil {
			m.OnHistorySyncReceived(sessionID, v)
		}

	case *events.AppState:
		eventType = model.EventAppState
		logger.Debug().Str("session", sessionID).Msg("App state received")

	case *events.AppStateSyncComplete:
		eventType = model.EventAppStateSyncComplete
		logger.Debug().Str("session", sessionID).Msg("App state sync complete")

	case *events.AppStateSyncError:
		eventType = model.EventAppStateSyncError
		logger.Debug().Str("session", sessionID).Msg("App state sync error")

	case *events.OfflineSyncCompleted:
		eventType = model.EventOfflineSyncCompleted
		logger.Info().Str("session", sessionID).Int("count", v.Count).Msg("Offline sync completed")

	case *events.OfflineSyncPreview:
		eventType = model.EventOfflineSyncPreview
		logger.Debug().Str("session", sessionID).Int("total", v.Total).Msg("Offline sync preview")

	// ── Privacy & Settings ────────────────────────────────────
	case *events.PrivacySettings:
		eventType = model.EventPrivacySettings
		logger.Debug().Str("session", sessionID).Msg("Privacy settings changed")

	case *events.PushNameSetting:
		eventType = model.EventPushNameSetting
		logger.Debug().Str("session", sessionID).Msg("Push name setting changed")

	case *events.UserStatusMute:
		eventType = model.EventUserStatusMute
		logger.Debug().Str("session", sessionID).Msg("User status mute changed")

	case *events.BlocklistChange:
		eventType = model.EventBlocklistChange
		logger.Debug().Str("session", sessionID).Str("jid", v.JID.String()).Msg("Blocklist changed")

	case *events.Blocklist:
		eventType = model.EventBlocklist
		logger.Debug().Str("session", sessionID).Msg("Blocklist received")

	// ── FB/Meta Bridge ────────────────────────────────────────
	case *events.FBMessage:
		eventType = model.EventFBMessage
		logger.Info().Str("session", sessionID).Msg("FB message received")

	default:
		return
	}

	var data map[string]interface{}

	switch v := evt.(type) {
	case *events.HistorySync:
		data = map[string]interface{}{}
		if v.Data != nil {
			data["syncType"] = v.Data.GetSyncType().String()
			data["chunkOrder"] = v.Data.GetChunkOrder()
			data["progress"] = v.Data.GetProgress()
			data["conversationCount"] = len(v.Data.GetConversations())
		}
	case *events.AppState:
		data = map[string]interface{}{}
		data["index"] = v.Index
		if v.SyncActionValue != nil {
			data["timestamp"] = v.GetTimestamp()
		}
	default:
		evtBytes, err := json.Marshal(evt)
		if err != nil {
			logger.Error().Err(err).Str("session", sessionID).Msg("Failed to serialize event")
			return
		}
		if err := json.Unmarshal(evtBytes, &data); err != nil {
			logger.Error().Err(err).Str("session", sessionID).Msg("Failed to unmarshal event")
			return
		}

		switch v2 := evt.(type) {
		case *events.PairError:
			if v2.Error != nil {
				data["Error"] = v2.Error.Error()
			}
		case *events.CATRefreshError:
			if v2.Error != nil {
				data["Error"] = v2.Error.Error()
			}
		case *events.AppStateSyncError:
			if v2.Error != nil {
				data["Error"] = v2.Error.Error()
			}
		case *events.Message:
			delete(data, "RawMessage")
			delete(data, "SourceWebMsg")
		}
	}

	bytes, err := model.BuildEventEnvelope(sessionID, m.getSessionName(sessionID), eventType, data)
	if err != nil {
		logger.Error().Err(err).Str("session", sessionID).Msg("Failed to marshal event payload")
		return
	}

	const maxNATSPayloadSize = 512 * 1024
	if m.nats != nil {
		if len(bytes) > maxNATSPayloadSize {
			logger.Debug().Str("session", sessionID).Str("event", string(eventType)).Int("size", len(bytes)).Msg("Event payload too large for NATS, skipping")
		} else if err := m.nats.Publish(m.ctx, "wzap.events."+sessionID, bytes); err != nil {
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
	case msg.GetReactionMessage() != nil:
		return "reaction", msg.GetReactionMessage().GetText(), ""
	default:
		return "unknown", "", ""
	}
}
