package chatwoot

import (
	"context"
	"fmt"
	"time"

	"github.com/skip2/go-qrcode"

	"wzap/internal/logger"
	"wzap/internal/model"
)

// eventDispatcher encapsula o switch de eventos inbound, delegando cada
// caso ao método process* correspondente no Service. Mantém ponteiro de
// volta para o Service porque os handlers dependem das deps injetadas
// (cache, clientFn, repos, etc.).
type eventDispatcher struct {
	svc *Service
}

func (d *eventDispatcher) Handle(ctx context.Context, cfg *Config, event model.EventType, payload []byte) error {
	s := d.svc
	switch event {
	case model.EventMessage:
		if err := s.processMessage(ctx, cfg, payload); err != nil {
			if isRetryableError(err) {
				return err
			}
			logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Msg("permanent error in processMessage, dropping")
		}
	case model.EventGroupInfo:
		if err := s.processGroupInfo(ctx, cfg, payload); err != nil {
			if isRetryableError(err) {
				return err
			}
			logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Msg("permanent error in processGroupInfo, dropping")
		}
	case model.EventReceipt:
		s.processReceipt(ctx, cfg, payload)
	case model.EventDeleteForMe:
		s.processDelete(ctx, cfg, payload)
	case model.EventMessageRevoke:
		s.processRevoke(ctx, cfg, payload)
	case model.EventMessageEdit:
		s.processEdit(ctx, cfg, payload)
	case model.EventConnected:
		s.processConnected(ctx, cfg, payload)
	case model.EventDisconnected:
		s.processDisconnected(ctx, cfg, payload)
	case model.EventQR:
		s.processQR(ctx, cfg, payload)
	case model.EventContact:
		s.processContact(ctx, cfg, payload)
	case model.EventPushName:
		s.processPushName(ctx, cfg, payload)
	case model.EventPicture:
		s.processPicture(ctx, cfg, payload)
	case model.EventHistorySync:
		s.processHistorySync(ctx, cfg, payload)
	default:
		// Events not handled by Chatwoot are ignored
		return nil
	}
	return nil
}

func (s *Service) processConnected(ctx context.Context, cfg *Config, _ []byte) {
	now := time.Now()
	if v, ok := s.lastBotNotify.Load(cfg.SessionID); ok {
		if lastTime, valid := v.(time.Time); valid && now.Sub(lastTime) < 30*time.Second {
			return
		}
	}
	s.lastBotNotify.Store(cfg.SessionID, now)

	convID, ok := s.findOpenBotConversation(ctx, cfg)
	if !ok {
		return
	}

	client := s.clientFn(cfg)
	_, _ = client.CreateMessage(ctx, convID, MessageReq{
		Content:     "✅ WhatsApp conectado com sucesso!",
		MessageType: "incoming",
	})

	if cfg.ImportOnConnect {
		period := cfg.ImportPeriod
		if period == "" {
			period = "7d"
		}
		go func() {
			s.ImportHistoryAsync(context.Background(), cfg.SessionID, period, 0)
		}()
	}
}

func (s *Service) processDisconnected(ctx context.Context, cfg *Config, _ []byte) {
	convID, ok := s.findOpenBotConversation(ctx, cfg)
	if !ok {
		return
	}

	client := s.clientFn(cfg)
	_, _ = client.CreateMessage(ctx, convID, MessageReq{
		Content:     "⚠️ Sessão desconectada do WhatsApp.",
		MessageType: "incoming",
	})
}

func (s *Service) processQR(ctx context.Context, cfg *Config, payload []byte) {
	var data struct {
		Codes       []string `json:"Codes"`
		PairingCode string   `json:"PairingCode"`
	}
	if err := parseEnvelopeData(payload, &data); err != nil {
		return
	}

	if len(data.Codes) == 0 {
		return
	}

	qrContent := data.Codes[0]

	convID, err := s.ensureBotConv(ctx, cfg)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Msg("Failed to find or create bot conversation for QR event")
		return
	}

	client := s.clientFn(cfg)
	qrPNG, err := qrcode.Encode(qrContent, qrcode.Medium, 256)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Msg("Failed to generate QR code")
		return
	}

	caption := "⚡️ QR Code gerado com sucesso!\n\nEscaneie o QR Code abaixo no WhatsApp para conectar."
	if len(data.PairingCode) >= 4 {
		caption += fmt.Sprintf("\n\n*Código de pareamento:* %s-%s", data.PairingCode[:4], data.PairingCode[4:])
	} else if data.PairingCode != "" {
		caption += fmt.Sprintf("\n\n*Código de pareamento:* %s", data.PairingCode)
	}

	_, _ = client.CreateAttachment(ctx, convID, caption, "qrcode.png", qrPNG, "image/png", "incoming", "", 0, nil)
}

func (s *Service) processContact(ctx context.Context, cfg *Config, payload []byte) {
	var data struct {
		JID    string `json:"JID"`
		Action struct {
			FullName  *string `json:"fullName"`
			FirstName *string `json:"firstName"`
			PnJID     *string `json:"pnLID"`
			LidJID    *string `json:"lidJID"`
		} `json:"Action"`
	}
	if err := parseEnvelopeData(payload, &data); err != nil {
		return
	}

	if data.JID == "" {
		return
	}

	pnJID := ""
	if data.Action.PnJID != nil {
		pnJID = *data.Action.PnJID
	}
	jid := s.resolveLID(ctx, cfg.SessionID, data.JID, pnJID)

	phone := extractPhone(jid)
	name := ""
	if data.Action.FullName != nil {
		name = *data.Action.FullName
	}
	if name == "" && data.Action.FirstName != nil {
		name = *data.Action.FirstName
	}
	if name == "" {
		return
	}

	client := s.clientFn(cfg)
	contacts, _ := client.FilterContacts(ctx, phone)
	if len(contacts) == 0 {
		return
	}

	_ = client.UpdateContact(ctx, contacts[0].ID, UpdateContactReq{Name: name})
}

func (s *Service) processPushName(ctx context.Context, cfg *Config, payload []byte) {
	var data struct {
		JID         string `json:"JID"`
		JIDAlt      string `json:"JIDAlt"`
		NewPushName string `json:"NewPushName"`
	}
	if err := parseEnvelopeData(payload, &data); err != nil {
		return
	}

	if data.JID == "" || data.NewPushName == "" {
		return
	}

	jid := s.resolveLID(ctx, cfg.SessionID, data.JID, data.JIDAlt)

	phone := extractPhone(jid)

	client := s.clientFn(cfg)
	contacts, _ := client.FilterContacts(ctx, phone)
	if len(contacts) == 0 {
		return
	}

	if s.contactNameGetter != nil {
		existingName := s.contactNameGetter.GetContactName(ctx, cfg.SessionID, jid)
		if existingName != "" && existingName != phone {
			return
		}
	}

	_ = client.UpdateContact(ctx, contacts[0].ID, UpdateContactReq{Name: data.NewPushName})
}

func (s *Service) processPicture(ctx context.Context, cfg *Config, payload []byte) {
	var data struct {
		JID       string `json:"JID"`
		PictureID string `json:"PictureID"`
		Remove    bool   `json:"Remove"`
	}
	if err := parseEnvelopeData(payload, &data); err != nil {
		return
	}

	if data.JID == "" || data.Remove {
		return
	}

	jid := s.resolveLID(ctx, cfg.SessionID, data.JID)

	phone := extractPhone(jid)

	var avatarURL string
	if s.picGetter != nil {
		url, err := s.picGetter.GetProfilePicture(ctx, cfg.SessionID, data.JID)
		if err != nil || url == "" {
			return
		}
		avatarURL = url
	} else {
		return
	}

	client := s.clientFn(cfg)
	contacts, _ := client.FilterContacts(ctx, phone)
	if len(contacts) == 0 {
		return
	}

	_ = client.UpdateContact(ctx, contacts[0].ID, UpdateContactReq{
		AvatarURL: avatarURL,
	})
}

func (s *Service) processGroupInfo(ctx context.Context, cfg *Config, payload []byte) error {
	var data struct {
		JID      string  `json:"JID"`
		Sender   *string `json:"Sender"`
		SenderPN *string `json:"SenderPN"`
		Name     *struct {
			Name string `json:"Name"`
		} `json:"Name"`
		Topic *struct {
			Topic        string `json:"Topic"`
			TopicDeleted bool   `json:"TopicDeleted"`
		} `json:"Topic"`
		Announce *struct {
			IsAnnounce bool `json:"IsAnnounce"`
		} `json:"Announce"`
		Ephemeral *struct {
			IsEphemeral       bool   `json:"IsEphemeral"`
			DisappearingTimer uint32 `json:"DisappearingTimer"`
		} `json:"Ephemeral"`
		NewInviteLink *string  `json:"NewInviteLink"`
		Join          []string `json:"Join"`
		Leave         []string `json:"Leave"`
		Promote       []string `json:"Promote"`
		Demote        []string `json:"Demote"`
	}
	if err := parseEnvelopeData(payload, &data); err != nil {
		return fmt.Errorf("parse group info envelope: %w", err)
	}
	if data.JID == "" {
		return nil
	}

	groupJID := data.JID
	convID, err := s.findOrCreateConversation(ctx, cfg, groupJID, "")
	if err != nil {
		return fmt.Errorf("find or create group conversation for %s: %w", groupJID, err)
	}

	client := s.clientFn(cfg)

	senderPhone := ""
	if data.SenderPN != nil && *data.SenderPN != "" {
		senderPhone = extractPhone(*data.SenderPN)
	} else if data.Sender != nil && *data.Sender != "" {
		senderPhone = extractPhone(s.resolveLID(ctx, cfg.SessionID, *data.Sender))
	}

	notifications := make([]string, 0, len(data.Join)+len(data.Leave)+len(data.Promote)+len(data.Demote))

	for _, jid := range data.Join {
		pJID := s.resolveLID(ctx, cfg.SessionID, jid)
		notifications = append(notifications, fmt.Sprintf("➕ %s entrou no grupo", extractPhone(pJID)))
	}
	for _, jid := range data.Leave {
		pJID := s.resolveLID(ctx, cfg.SessionID, jid)
		phone := extractPhone(pJID)
		if senderPhone != "" && senderPhone != phone {
			notifications = append(notifications, fmt.Sprintf("➖ %s foi removido do grupo por %s", phone, senderPhone))
		} else {
			notifications = append(notifications, fmt.Sprintf("🚪 %s saiu do grupo", phone))
		}
	}
	for _, jid := range data.Promote {
		pJID := s.resolveLID(ctx, cfg.SessionID, jid)
		notifications = append(notifications, fmt.Sprintf("⬆️ %s foi promovido a admin por %s", extractPhone(pJID), senderPhone))
	}
	for _, jid := range data.Demote {
		pJID := s.resolveLID(ctx, cfg.SessionID, jid)
		notifications = append(notifications, fmt.Sprintf("⬇️ %s foi rebaixado de admin por %s", extractPhone(pJID), senderPhone))
	}

	if data.Name != nil && data.Name.Name != "" {
		notifications = append(notifications, fmt.Sprintf("📝 Nome do grupo alterado para: %s", data.Name.Name))
	}
	if data.Topic != nil {
		if data.Topic.TopicDeleted {
			notifications = append(notifications, "📋 Descrição do grupo removida")
		} else if data.Topic.Topic != "" {
			notifications = append(notifications, fmt.Sprintf("📋 Descrição do grupo atualizada: %s", data.Topic.Topic))
		}
	}
	if data.NewInviteLink != nil && *data.NewInviteLink != "" {
		notifications = append(notifications, fmt.Sprintf("🔗 Link de convite do grupo: %s", *data.NewInviteLink))
	}
	if data.Announce != nil {
		if data.Announce.IsAnnounce {
			notifications = append(notifications, "🔒 Grupo restrito a admins")
		} else {
			notifications = append(notifications, "🔓 Grupo aberto para todos enviarem mensagens")
		}
	}
	if data.Ephemeral != nil {
		if data.Ephemeral.IsEphemeral {
			notifications = append(notifications, fmt.Sprintf("⏳ Mensagens temporárias ativadas (%d segundos)", data.Ephemeral.DisappearingTimer))
		} else {
			notifications = append(notifications, "⏳ Mensagens temporárias desativadas")
		}
	}

	for _, notif := range notifications {
		_, _ = client.CreateMessage(ctx, convID, MessageReq{
			Content:     notif,
			MessageType: "activity",
		})
	}
	return nil
}

func (s *Service) processHistorySync(_ context.Context, cfg *Config, _ []byte) {
	logger.Debug().Str("component", "chatwoot").Str("session", cfg.SessionID).Msg("HistorySync received (no-op until import triggered)")
}

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
	}

	return msg, nil
}
