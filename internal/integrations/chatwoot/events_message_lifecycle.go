package chatwoot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"wzap/internal/logger"
	"wzap/internal/model"
)

func (s *Service) processReceipt(ctx context.Context, cfg *Config, payload []byte) {
	data, err := parseReceiptPayload(payload)
	if err != nil {
		return
	}

	if len(data.MessageIDs) == 0 {
		return
	}

	client := s.clientFn(cfg)
	for _, msgID := range data.MessageIDs {
		if msgID == "" {
			continue
		}

		msg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, msgID)
		if err != nil {
			continue
		}

		if msg.CWConvID == nil || *msg.CWConvID == 0 {
			continue
		}

		if msg.CWSrcID != nil {
			_ = client.UpdateLastSeen(ctx, fmt.Sprintf("%d", cfg.InboxID), *msg.CWSrcID, *msg.CWConvID)
		}
	}
}

func (s *Service) processDelete(ctx context.Context, cfg *Config, payload []byte) {
	data, err := parseDeletePayload(payload)
	if err != nil {
		return
	}

	msgID := data.MessageID
	if msgID == "" {
		return
	}

	msg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, msgID)
	if err != nil {
		return
	}

	if msg.CWMessageID == nil || msg.CWConvID == nil {
		return
	}

	client := s.clientFn(cfg)
	if err := client.DeleteMessage(ctx, *msg.CWConvID, *msg.CWMessageID); err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("Failed to delete Chatwoot message")
	}
}

func (s *Service) processRevoke(ctx context.Context, cfg *Config, payload []byte) {
	data, err := parseMessagePayload(payload)
	if err != nil {
		return
	}

	protocolMsg := getMapField(data.Message, "protocolMessage")
	if protocolMsg == nil {
		return
	}
	key := getMapField(protocolMsg, "key")
	if key == nil {
		return
	}
	revokedMsgID := getStringField(key, "ID")
	if revokedMsgID == "" {
		return
	}

	logger.Debug().Str("component", "chatwoot").Str("session", cfg.SessionID).Str("revokedMsgID", revokedMsgID).Msg("handling message revoke")

	msg, err := s.waitForCWRef(ctx, cfg.SessionID, revokedMsgID)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("revokedMsgID", revokedMsgID).Msg("message not found for revoke")
		return
	}

	if msg.CWMessageID == nil || msg.CWConvID == nil {
		logger.Warn().Str("component", "chatwoot").Str("revokedMsgID", revokedMsgID).Msg("revoke: CW refs not available after retry")
		return
	}

	client := s.clientFn(cfg)
	if err := client.DeleteMessage(ctx, *msg.CWConvID, *msg.CWMessageID); err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("revokedMsgID", revokedMsgID).Msg("failed to delete Chatwoot message on revoke")
	} else {
		logger.Debug().Str("component", "chatwoot").Str("revokedMsgID", revokedMsgID).Int("cwMsgID", *msg.CWMessageID).Msg("successfully deleted Chatwoot message on revoke")
	}
}

func (s *Service) processEdit(ctx context.Context, cfg *Config, payload []byte) {
	data, err := parseMessagePayload(payload)
	if err != nil {
		return
	}

	protocolMsg := getMapField(data.Message, "protocolMessage")
	if protocolMsg == nil {
		return
	}
	key := getMapField(protocolMsg, "key")
	if key == nil {
		return
	}
	editedMsgID := getStringField(key, "ID")
	if editedMsgID == "" {
		return
	}

	editedMessage := getMapField(protocolMsg, "editedMessage")
	if editedMessage == nil {
		return
	}

	newText := extractText(editedMessage)
	if newText == "" {
		return
	}

	logger.Debug().Str("component", "chatwoot").Str("session", cfg.SessionID).Str("editedMsgID", editedMsgID).Str("newText", newText).Msg("handling message edit")

	// Cloud mode: re-post via cloud webhook with the original message ID.
	// Chatwoot matches by source_id and updates the message in-place.
	// This is the same approach used by unoapi-cloud (no CW ref lookup needed).
	if cfg.InboxType == "cloud" {
		s.processEditCloud(ctx, cfg, data, editedMsgID, newText)
		return
	}

	// API mode: create a private reply note with the edited content.
	msg, err := s.waitForCWRef(ctx, cfg.SessionID, editedMsgID)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("editedMsgID", editedMsgID).Msg("message not found for edit")
		return
	}

	if msg.CWMessageID == nil || msg.CWConvID == nil {
		logger.Warn().Str("component", "chatwoot").Str("editedMsgID", editedMsgID).Msg("edit: CW refs not available after retry")
		return
	}

	logger.Debug().Str("component", "chatwoot").Str("editedMsgID", editedMsgID).Int("cwMsgID", *msg.CWMessageID).Int("cwConvID", *msg.CWConvID).Msg("creating edit notification in Chatwoot")

	client := s.clientFn(cfg)

	messageType := "incoming"
	if msg.FromMe {
		messageType = "outgoing"
	}

	editedContent := "✏️ *Mensagem editada:*\n" + newText
	_, err = client.CreateMessage(ctx, *msg.CWConvID, MessageReq{
		Content:           editedContent,
		MessageType:       messageType,
		Private:           true,
		SourceReplyID:     *msg.CWMessageID,
		ContentAttributes: map[string]any{"in_reply_to": *msg.CWMessageID},
	})
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("editedMsgID", editedMsgID).Msg("failed to create edit notification in Chatwoot")
		return
	}

	logger.Debug().Str("component", "chatwoot").Str("editedMsgID", editedMsgID).Msg("successfully created edit notification in Chatwoot")
}

// processEditCloud handles message edits for cloud inbox mode.
// It uses resolveCloudRefViaAPI to find the CW message/conversation IDs
// (since cloud mode doesn't populate refs via webhook reliably), then
// creates a private note with the edited content — same as API mode.
func (s *Service) processEditCloud(ctx context.Context, cfg *Config, data *waMessagePayload, editedMsgID, newText string) {
	chatJID := data.Info.Chat
	chatJID = s.resolveLID(ctx, cfg.SessionID, chatJID, data.Info.SenderAlt, data.Info.RecipientAlt)
	if strings.HasSuffix(chatJID, "@lid") {
		logger.Warn().Str("component", "chatwoot").Str("session", cfg.SessionID).Str("editedMsgID", editedMsgID).Str("lid", chatJID).Msg("edit cloud: unresolvable LID, skipping")
		return
	}

	// Try to resolve CW refs — first from wz_messages (may have been filled
	// by resolveCloudRefAsync), then via database_uri, then via REST API.
	msg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, editedMsgID)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("editedMsgID", editedMsgID).Msg("edit cloud: message not found in wz_messages")
		return
	}

	if msg.CWMessageID == nil || msg.CWConvID == nil {
		// Not yet populated — resolve proactively via API
		if ref, ok := s.resolveAndPersistMessageRef(ctx, cfg, editedMsgID); ok && ref != nil {
			msg.CWMessageID = &ref.MessageID
			msg.CWConvID = &ref.ConversationID
		} else if ref, ok := s.resolveCloudRefViaAPI(ctx, cfg, editedMsgID, chatJID); ok && ref != nil {
			msg.CWMessageID = &ref.MessageID
			msg.CWConvID = &ref.ConversationID
		}
	}

	if msg.CWMessageID == nil || msg.CWConvID == nil {
		logger.Warn().Str("component", "chatwoot").Str("session", cfg.SessionID).Str("editedMsgID", editedMsgID).Msg("edit cloud: CW refs not available")
		return
	}

	client := s.clientFn(cfg)

	messageType := "incoming"
	if msg.FromMe {
		messageType = "outgoing"
	}

	editedContent := "✏️ *Mensagem editada:*\n" + newText
	logger.Debug().Str("component", "chatwoot").Str("session", cfg.SessionID).Str("editedMsgID", editedMsgID).Str("inbox_type", cfg.InboxType).Str("message_type", messageType).Msg("Attempting to create edit notification in Cloud mode")
	_, err = client.CreateMessage(ctx, *msg.CWConvID, MessageReq{
		Content:           editedContent,
		MessageType:       messageType,
		Private:           true,
		SourceReplyID:     *msg.CWMessageID,
		ContentAttributes: map[string]any{"in_reply_to": *msg.CWMessageID},
	})
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Str("editedMsgID", editedMsgID).Msg("edit cloud: failed to create edit notification")
		return
	}

	logger.Debug().Str("component", "chatwoot").Str("session", cfg.SessionID).Str("editedMsgID", editedMsgID).Int("cwMsgID", *msg.CWMessageID).Int("cwConvID", *msg.CWConvID).Msg("edit cloud: created edit notification")
}

func (s *Service) waitForCWRef(ctx context.Context, sessionID, msgID string) (*model.Message, error) {
	delays := []time.Duration{
		200 * time.Millisecond,
		300 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		3 * time.Second,
	}

	msg, err := s.msgRepo.FindByID(ctx, sessionID, msgID)
	if err != nil {
		return nil, err
	}

	if msg.CWMessageID != nil && msg.CWConvID != nil {
		return msg, nil
	}

	cfg, cfgErr := s.repo.FindBySessionID(ctx, sessionID)
	if cfgErr == nil {
		if ref, ok := s.resolveAndPersistMessageRef(ctx, cfg, msgID); ok && ref != nil {
			refreshed, findErr := s.msgRepo.FindByID(ctx, sessionID, msgID)
			if findErr == nil && refreshed != nil {
				return refreshed, nil
			}
			msg.CWMessageID = &ref.MessageID
			msg.CWConvID = &ref.ConversationID
			storedSourceID := ref.SourceID
			msg.CWSrcID = &storedSourceID
			return msg, nil
		}

		// Fallback: try Chatwoot REST API when database_uri is unavailable
		if msg.ChatJID != "" {
			if ref, ok := s.resolveCloudRefViaAPI(ctx, cfg, msgID, msg.ChatJID); ok && ref != nil {
				refreshed, findErr := s.msgRepo.FindByID(ctx, sessionID, msgID)
				if findErr == nil && refreshed != nil {
					return refreshed, nil
				}
				msg.CWMessageID = &ref.MessageID
				msg.CWConvID = &ref.ConversationID
				storedSourceID := ref.SourceID
				msg.CWSrcID = &storedSourceID
				return msg, nil
			}
		}
	}

	logger.Debug().Str("component", "chatwoot").Str("msgID", msgID).Msg("CW refs not yet available, starting retry loop")

	for i, delay := range delays {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}

		msg, err = s.msgRepo.FindByID(ctx, sessionID, msgID)
		if err != nil {
			return nil, err
		}

		if msg.CWMessageID != nil && msg.CWConvID != nil {
			logger.Debug().Str("component", "chatwoot").Str("msgID", msgID).Int("attempt", i+2).Msg("CW refs available after retry")
			return msg, nil
		}

		if cfgErr == nil {
			if ref, ok := s.resolveAndPersistMessageRef(ctx, cfg, msgID); ok && ref != nil {
				refreshed, findErr := s.msgRepo.FindByID(ctx, sessionID, msgID)
				if findErr == nil && refreshed != nil {
					return refreshed, nil
				}
				msg.CWMessageID = &ref.MessageID
				msg.CWConvID = &ref.ConversationID
				storedSourceID := ref.SourceID
				msg.CWSrcID = &storedSourceID
				return msg, nil
			}

			// Fallback: try Chatwoot REST API when database_uri is unavailable
			if msg.ChatJID != "" {
				if ref, ok := s.resolveCloudRefViaAPI(ctx, cfg, msgID, msg.ChatJID); ok && ref != nil {
					refreshed, findErr := s.msgRepo.FindByID(ctx, sessionID, msgID)
					if findErr == nil && refreshed != nil {
						return refreshed, nil
					}
					msg.CWMessageID = &ref.MessageID
					msg.CWConvID = &ref.ConversationID
					storedSourceID := ref.SourceID
					msg.CWSrcID = &storedSourceID
					return msg, nil
				}
			}
		}
	}

	return msg, nil
}
