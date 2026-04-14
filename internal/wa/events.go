package wa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waMmsRetry"
	"go.mau.fi/whatsmeow/types/events"

	"wzap/internal/logger"
	"wzap/internal/model"
)

var eventBufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

func (m *Manager) handleEvent(sessionID string, evt any) {
	var eventType model.EventType

	switch v := evt.(type) {

	// ── Messages ──────────────────────────────────────────────
	case *events.Message:
		if proto := v.Message.GetProtocolMessage(); proto != nil &&
			proto.GetType() == waE2E.ProtocolMessage_REVOKE &&
			proto.GetKey() != nil && proto.GetKey().GetID() != "" {
			eventType = model.EventMessageRevoke
			logger.Debug().
				Str("component", "wa").
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
				Str("component", "wa").
				Str("session", sessionID).
				Str("editedMsgID", proto.GetKey().GetID()).
				Str("chat", v.Info.Chat.String()).
				Bool("fromMe", v.Info.IsFromMe).
				Msg("Message edited")
		} else if proto := v.Message.GetProtocolMessage(); proto != nil {
			return
		} else {
			if v.Message.GetSenderKeyDistributionMessage() != nil {
				msgType, _, _ := extractMessageContent(v.Message)
				if msgType == "unknown" {
					logger.Debug().
						Str("component", "wa").
						Str("session", sessionID).
						Str("mid", v.Info.ID).
						Msg("Sender key distribution message filtered")
					return
				}
			}
			eventType = model.EventMessage
			{
				msgType, _, mediaType := extractMessageContent(v.Message)
				logger.Info().
					Str("component", "wa").
					Str("session", sessionID).
					Str("from", v.Info.Sender.String()).
					Str("chat", v.Info.Chat.String()).
					Str("mid", v.Info.ID).
					Bool("fromMe", v.Info.IsFromMe).
					Str("msgType", msgType).
					Str("mediaType", mediaType).
					Msg("Message received")
				if msgType == "unknown" && v.Message != nil {
					logger.Debug().
						Str("component", "wa").
						Str("session", sessionID).
						Str("mid", v.Info.ID).
						Str("proto", v.Message.String()).
						Msg("Unknown message type — proto dump")
				}
			}

			if m.OnMediaReceived != nil && v.Message != nil {
				chatJID := v.Info.Chat.String()
				senderJID := v.Info.Sender.String()
				fromMe := v.Info.IsFromMe
				ts := v.Info.Timestamp
				switch {
				case v.Message.GetImageMessage() != nil:
					m.OnMediaReceived(sessionID, v.Info.ID, chatJID, senderJID, v.Message.GetImageMessage().GetMimetype(), fromMe, ts, v.Message.GetImageMessage())
				case v.Message.GetVideoMessage() != nil:
					m.OnMediaReceived(sessionID, v.Info.ID, chatJID, senderJID, v.Message.GetVideoMessage().GetMimetype(), fromMe, ts, v.Message.GetVideoMessage())
				case v.Message.GetAudioMessage() != nil:
					m.OnMediaReceived(sessionID, v.Info.ID, chatJID, senderJID, v.Message.GetAudioMessage().GetMimetype(), fromMe, ts, v.Message.GetAudioMessage())
				case v.Message.GetDocumentMessage() != nil:
					m.OnMediaReceived(sessionID, v.Info.ID, chatJID, senderJID, v.Message.GetDocumentMessage().GetMimetype(), fromMe, ts, v.Message.GetDocumentMessage())
				case v.Message.GetStickerMessage() != nil:
					m.OnMediaReceived(sessionID, v.Info.ID, chatJID, senderJID, v.Message.GetStickerMessage().GetMimetype(), fromMe, ts, v.Message.GetStickerMessage())
				}
			}

			if m.OnMessageReceived != nil {
				msgType, body, mediaType := extractMessageContent(v.Message)
				m.OnMessageReceived(sessionID, v.Info.ID, v.Info.Chat.String(), v.Info.Sender.String(), v.Info.IsFromMe, msgType, body, mediaType, v.Info.Timestamp.Unix(), v.Message)
			}
		}

	case *events.UndecryptableMessage:
		eventType = model.EventUndecryptableMessage
		logger.Warn().Str("component", "wa").Str("session", sessionID).Str("mid", v.Info.ID).Msg("Undecryptable message")

	case *events.MediaRetry:
		eventType = model.EventMediaRetry
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("mid", string(v.MessageID)).Msg("Media retry")
		m.handleMediaRetry(v)

	case *events.Receipt:
		eventType = model.EventReceipt
		if v.Type != "" && v.Type != "read" && v.Type != "read-self" && v.Type != "played" && v.Type != "played-self" {
			logger.Debug().Str("component", "wa").Str("session", sessionID).Str("type", string(v.Type)).Msg("Receipt ignored")
			return
		}
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("type", string(v.Type)).Str("chat", v.Chat.String()).Int("count", len(v.MessageIDs)).Msg("Receipt received")

	case *events.DeleteForMe:
		eventType = model.EventDeleteForMe
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("mid", v.MessageID).Msg("Message deleted for me")

	// ── Connection ────────────────────────────────────────────
	case *events.Connected:
		eventType = model.EventConnected
		logger.Info().Str("component", "wa").Str("session", sessionID).Msg("Session connected")

	case *events.Disconnected:
		eventType = model.EventDisconnected
		logger.Warn().Str("component", "wa").Str("session", sessionID).Msg("Session disconnected")

	case *events.ManualLoginReconnect:
		eventType = model.EventManualLoginReconnect
		logger.Info().Str("component", "wa").Str("session", sessionID).Msg("Manual login reconnect")

	// ── Pairing ───────────────────────────────────────────────
	case *events.QR:
		eventType = model.EventQR
		logger.Debug().Str("component", "wa").Str("session", sessionID).Int("codes", len(v.Codes)).Msg("QR codes received")

	case *events.QRScannedWithoutMultidevice:
		eventType = model.EventQRScannedWithoutMultidevice
		logger.Warn().Str("component", "wa").Str("session", sessionID).Msg("QR scanned without multidevice")

	case *events.PairSuccess:
		eventType = model.EventPairSuccess
		logger.Info().Str("component", "wa").Str("session", sessionID).Str("jid", v.ID.String()).Msg("Pair success")

	case *events.PairError:
		eventType = model.EventPairError
		logger.Error().Str("component", "wa").Str("session", sessionID).Msg("Pair error")

	// ── Connection Errors ─────────────────────────────────────
	case *events.ConnectFailure:
		eventType = model.EventConnectFailure
		logger.Error().Str("component", "wa").Str("session", sessionID).Int("reason", int(v.Reason)).Msg("Connect failure")

	case *events.LoggedOut:
		eventType = model.EventLoggedOut
		logger.Warn().Str("component", "wa").Str("session", sessionID).Str("reason", v.Reason.String()).Msg("Session logged out")

	case *events.StreamError:
		eventType = model.EventStreamError
		logger.Error().Str("component", "wa").Str("session", sessionID).Str("code", v.Code).Msg("Stream error")

	case *events.StreamReplaced:
		eventType = model.EventStreamReplaced
		logger.Warn().Str("component", "wa").Str("session", sessionID).Msg("Stream replaced")

	case *events.KeepAliveTimeout:
		eventType = model.EventKeepAliveTimeout
		logger.Warn().Str("component", "wa").Str("session", sessionID).Int("errorCount", v.ErrorCount).Msg("Keep-alive timeout")

	case *events.KeepAliveRestored:
		eventType = model.EventKeepAliveRestored
		logger.Info().Str("component", "wa").Str("session", sessionID).Msg("Keep-alive restored")

	case *events.ClientOutdated:
		eventType = model.EventClientOutdated
		logger.Error().Str("component", "wa").Str("session", sessionID).Msg("Client outdated")

	case *events.TemporaryBan:
		eventType = model.EventTemporaryBan
		logger.Error().Str("component", "wa").Str("session", sessionID).Str("expire", v.Expire.String()).Msg("Temporary ban")

	case *events.CATRefreshError:
		eventType = model.EventCATRefreshError
		logger.Error().Str("component", "wa").Str("session", sessionID).Msg("CAT refresh error")

	// ── Contacts ──────────────────────────────────────────────
	case *events.Contact:
		eventType = model.EventContact
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("jid", v.JID.String()).Msg("Contact changed")

	case *events.PushName:
		eventType = model.EventPushName
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("jid", v.JID.String()).Msg("Push name changed")

	case *events.BusinessName:
		eventType = model.EventBusinessName
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("jid", v.JID.String()).Msg("Business name changed")

	// ── Profile & Identity ────────────────────────────────────
	case *events.Picture:
		eventType = model.EventPicture
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("jid", v.JID.String()).Msg("Picture changed")

	case *events.IdentityChange:
		eventType = model.EventIdentityChange
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("jid", v.JID.String()).Msg("Identity changed")

	case *events.UserAbout:
		eventType = model.EventUserAbout
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("jid", v.JID.String()).Msg("User about changed")

	// ── Groups ────────────────────────────────────────────────
	case *events.GroupInfo:
		eventType = model.EventGroupInfo
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("group", v.JID.String()).Msg("Group info update")

	case *events.JoinedGroup:
		eventType = model.EventJoinedGroup
		logger.Info().Str("component", "wa").Str("session", sessionID).Str("group", v.JID.String()).Msg("Joined group")

	// ── Presence ──────────────────────────────────────────────
	case *events.Presence:
		eventType = model.EventPresence
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("jid", v.From.String()).Bool("unavailable", v.Unavailable).Msg("Presence update")

	case *events.ChatPresence:
		eventType = model.EventChatPresence
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("chat", v.Chat.String()).Msg("Chat presence")

	// ── Chat State ────────────────────────────────────────────
	case *events.Archive:
		eventType = model.EventArchive
		{
			action := "archived"
			if v.Action != nil && !v.Action.GetArchived() {
				action = "unarchived"
			}
			logger.Debug().Str("component", "wa").Str("session", sessionID).Str("jid", v.JID.String()).Str("action", action).Msg("Chat archived/unarchived")
		}

	case *events.Mute:
		eventType = model.EventMute
		{
			action := "muted"
			if v.Action == nil || !v.Action.GetMuted() {
				action = "unmuted"
			}
			logger.Debug().Str("component", "wa").Str("session", sessionID).Str("jid", v.JID.String()).Str("action", action).Msg("Chat muted/unmuted")
		}

	case *events.Pin:
		eventType = model.EventPin
		{
			action := "pinned"
			if v.Action != nil && !v.Action.GetPinned() {
				action = "unpinned"
			}
			logger.Debug().Str("component", "wa").Str("session", sessionID).Str("jid", v.JID.String()).Str("action", action).Msg("Chat pinned/unpinned")
		}

	case *events.Star:
		eventType = model.EventStar
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("mid", v.MessageID).Msg("Message starred")

	case *events.ClearChat:
		eventType = model.EventClearChat
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("jid", v.JID.String()).Msg("Chat cleared")

	case *events.DeleteChat:
		eventType = model.EventDeleteChat
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("jid", v.JID.String()).Msg("Chat deleted")

	case *events.MarkChatAsRead:
		eventType = model.EventMarkChatAsRead
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("jid", v.JID.String()).Msg("Chat marked as read")

	case *events.UnarchiveChatsSetting:
		eventType = model.EventUnarchiveChatsSetting
		logger.Debug().Str("component", "wa").Str("session", sessionID).Msg("Unarchive chats setting changed")

	// ── Labels ────────────────────────────────────────────────
	case *events.LabelEdit:
		eventType = model.EventLabelEdit
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("labelId", v.LabelID).Msg("Label edited")

	case *events.LabelAssociationChat:
		eventType = model.EventLabelAssociationChat
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("labelId", v.LabelID).Msg("Label association chat")

	case *events.LabelAssociationMessage:
		eventType = model.EventLabelAssociationMessage
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("labelId", v.LabelID).Msg("Label association message")

	// ── Calls ─────────────────────────────────────────────────
	case *events.CallOffer:
		eventType = model.EventCallOffer
		logger.Info().Str("component", "wa").Str("session", sessionID).Str("from", v.From.String()).Str("callID", v.CallID).Msg("Incoming call")

	case *events.CallAccept:
		eventType = model.EventCallAccept
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("from", v.From.String()).Str("callID", v.CallID).Msg("Call accepted")

	case *events.CallTerminate:
		eventType = model.EventCallTerminate
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("from", v.From.String()).Str("callID", v.CallID).Msg("Call terminated")

	case *events.CallOfferNotice:
		eventType = model.EventCallOfferNotice
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("from", v.From.String()).Str("callID", v.CallID).Msg("Call offer notice")

	case *events.CallRelayLatency:
		eventType = model.EventCallRelayLatency
		logger.Debug().Str("component", "wa").Str("session", sessionID).Msg("Call relay latency")

	case *events.CallPreAccept:
		eventType = model.EventCallPreAccept
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("from", v.From.String()).Str("callID", v.CallID).Msg("Call pre-accept")

	case *events.CallReject:
		eventType = model.EventCallReject
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("from", v.From.String()).Str("callID", v.CallID).Msg("Call rejected")

	case *events.CallTransport:
		eventType = model.EventCallTransport
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("from", v.From.String()).Str("callID", v.CallID).Msg("Call transport")

	case *events.UnknownCallEvent:
		eventType = model.EventUnknownCallEvent
		logger.Debug().Str("component", "wa").Str("session", sessionID).Msg("Unknown call event")

	// ── Newsletter ────────────────────────────────────────────
	case *events.NewsletterJoin:
		eventType = model.EventNewsletterJoin
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("jid", v.ID.String()).Msg("Newsletter joined")

	case *events.NewsletterLeave:
		eventType = model.EventNewsletterLeave
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("jid", v.ID.String()).Msg("Newsletter left")

	case *events.NewsletterMuteChange:
		eventType = model.EventNewsletterMuteChange
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("jid", v.ID.String()).Msg("Newsletter mute changed")

	case *events.NewsletterLiveUpdate:
		eventType = model.EventNewsletterLiveUpdate
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("jid", v.JID.String()).Msg("Newsletter live update")

	// ── Sync ──────────────────────────────────────────────────
	case *events.HistorySync:
		eventType = model.EventHistorySync
		logger.Info().Str("component", "wa").Str("session", sessionID).Msg("History sync received")
		if m.OnHistorySyncReceived != nil {
			m.OnHistorySyncReceived(sessionID, v)
		}

	case *events.AppState:
		eventType = model.EventAppState
		logger.Debug().Str("component", "wa").Str("session", sessionID).Msg("App state received")

	case *events.AppStateSyncComplete:
		eventType = model.EventAppStateSyncComplete
		logger.Debug().Str("component", "wa").Str("session", sessionID).Msg("App state sync complete")

	case *events.AppStateSyncError:
		eventType = model.EventAppStateSyncError
		logger.Warn().Str("component", "wa").Str("session", sessionID).Msg("App state sync error")

	case *events.OfflineSyncCompleted:
		eventType = model.EventOfflineSyncCompleted
		logger.Info().Str("component", "wa").Str("session", sessionID).Int("count", v.Count).Msg("Offline sync completed")

	case *events.OfflineSyncPreview:
		eventType = model.EventOfflineSyncPreview
		logger.Debug().Str("component", "wa").Str("session", sessionID).Int("total", v.Total).Msg("Offline sync preview")

	// ── Privacy & Settings ────────────────────────────────────
	case *events.PrivacySettings:
		eventType = model.EventPrivacySettings
		logger.Debug().Str("component", "wa").Str("session", sessionID).Msg("Privacy settings changed")

	case *events.PushNameSetting:
		eventType = model.EventPushNameSetting
		logger.Debug().Str("component", "wa").Str("session", sessionID).Msg("Push name setting changed")

	case *events.UserStatusMute:
		eventType = model.EventUserStatusMute
		logger.Debug().Str("component", "wa").Str("session", sessionID).Msg("User status mute changed")

	case *events.BlocklistChange:
		eventType = model.EventBlocklistChange
		logger.Debug().Str("component", "wa").Str("session", sessionID).Str("jid", v.JID.String()).Msg("Blocklist changed")

	case *events.Blocklist:
		eventType = model.EventBlocklist
		logger.Debug().Str("component", "wa").Str("session", sessionID).Msg("Blocklist received")

	// ── FB/Meta Bridge ────────────────────────────────────────
	case *events.FBMessage:
		eventType = model.EventFBMessage
		logger.Info().Str("component", "wa").Str("session", sessionID).Msg("FB message received")

	default:
		return
	}

	var data map[string]any

	switch v := evt.(type) {
	case *events.HistorySync:
		data = map[string]any{}
		if v.Data != nil {
			data["syncType"] = v.Data.GetSyncType().String()
			data["chunkOrder"] = v.Data.GetChunkOrder()
			data["progress"] = v.Data.GetProgress()
			data["conversationCount"] = len(v.Data.GetConversations())
		}
	case *events.AppState:
		data = map[string]any{}
		data["index"] = v.Index
		if v.SyncActionValue != nil {
			data["timestamp"] = v.GetTimestamp()
		}
	default:
		buf := eventBufPool.Get().(*bytes.Buffer)
		buf.Reset()
		if err := json.NewEncoder(buf).Encode(evt); err != nil {
			eventBufPool.Put(buf)
			logger.Error().Str("component", "wa").Err(err).Str("session", sessionID).Msg("Failed to serialize event")
			return
		}
		if err := json.NewDecoder(buf).Decode(&data); err != nil {
			eventBufPool.Put(buf)
			logger.Error().Str("component", "wa").Err(err).Str("session", sessionID).Msg("Failed to unmarshal event")
			return
		}
		eventBufPool.Put(buf)

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

	nameCtx, nameCancel := context.WithTimeout(context.Background(), 3*time.Second)
	sessionName := m.getSessionName(nameCtx, sessionID)
	nameCancel()

	bytes, err := model.BuildEventEnvelope(sessionID, sessionName, eventType, data)
	if err != nil {
		logger.Error().Str("component", "wa").Err(err).Str("session", sessionID).Msg("Failed to marshal event payload")
		return
	}

	const maxNATSPayloadSize = 512 * 1024
	if m.nats != nil {
		if len(bytes) > maxNATSPayloadSize {
			logger.Debug().Str("component", "wa").Str("session", sessionID).Str("event", string(eventType)).Int("size", len(bytes)).Msg("Event payload too large for NATS, skipping")
		} else {
			pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := m.nats.Publish(pubCtx, "wzap.events."+sessionID, bytes); err != nil {
				logger.Error().Str("component", "wa").Err(err).Str("session", sessionID).Msg("Failed to publish NATS event")
			}
			pubCancel()
		}
	}

	if m.dispatcher != nil {
		m.dispatcher.DispatchAsync(sessionID, eventType, bytes)
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
	case msg.GetTemplateMessage() != nil:
		return "template", msg.GetTemplateMessage().GetHydratedTemplate().GetHydratedContentText(), ""
	case msg.GetInteractiveMessage() != nil:
		return "interactive", msg.GetInteractiveMessage().GetHeader().GetSubtitle(), ""
	case msg.GetPollUpdateMessage() != nil:
		return "poll_update", "", ""
	default:
		return "unknown", "", ""
	}
}

func (m *Manager) handleMediaRetry(v *events.MediaRetry) {
	mid := string(v.MessageID)
	raw, ok := m.mediaRetryCache.Load(mid)
	if !ok {
		return
	}

	entry := raw.(mediaRetryCacheEntry)
	m.mediaRetryCache.Delete(mid)

	if time.Now().After(entry.expiresAt) {
		return
	}

	retryData, err := whatsmeow.DecryptMediaRetryNotification(v, entry.mediaKey)
	if err != nil {
		if errors.Is(err, whatsmeow.ErrMediaNotAvailableOnPhone) {
			logger.Warn().Str("component", "wa").Str("session", entry.sessionID).Str("mid", mid).Msg("Media retry: mídia não disponível no celular")
		} else {
			logger.Warn().Str("component", "wa").Err(err).Str("session", entry.sessionID).Str("mid", mid).Msg("Media retry: falha ao decriptar notificação")
		}
		return
	}

	if retryData.GetResult() != waMmsRetry.MediaRetryNotification_SUCCESS {
		logger.Warn().Str("component", "wa").Str("session", entry.sessionID).Str("mid", mid).Str("result", retryData.GetResult().String()).Msg("Media retry: servidor retornou falha")
		return
	}

	if retryData.GetDirectPath() == "" {
		logger.Warn().Str("component", "wa").Str("session", entry.sessionID).Str("mid", mid).Msg("Media retry: SUCCESS mas directPath vazio")
		return
	}

	if m.OnMediaRetry == nil {
		return
	}

	logger.Debug().Str("component", "wa").Str("session", entry.sessionID).Str("mid", mid).Msg("Media retry: sucesso, re-enviando para upload")
	m.OnMediaRetry(
		entry.sessionID, mid, entry.chatJID, entry.senderJID,
		entry.fromMe, entry.mimeType, entry.timestamp,
		retryData.GetDirectPath(), entry.encFileHash, entry.fileHash, entry.mediaKey, entry.fileLength,
	)
}
