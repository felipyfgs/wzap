package chatwoot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"wzap/internal/logger"
)

func (s *Service) handleReceipt(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
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

		if msg.CWConversationID == nil || *msg.CWConversationID == 0 {
			continue
		}

		if msg.CWSourceID != nil {
			_ = client.UpdateLastSeen(ctx, fmt.Sprintf("%d", cfg.InboxID), *msg.CWSourceID, *msg.CWConversationID)
		}
	}
}

func (s *Service) handleDelete(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
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

	if msg.CWMessageID == nil || msg.CWConversationID == nil {
		return
	}

	client := s.clientFn(cfg)
	if err := client.DeleteMessage(ctx, *msg.CWConversationID, *msg.CWMessageID); err != nil {
		logger.Warn().Err(err).Msg("Failed to delete Chatwoot message")
	}
}

func (s *Service) handleRevoke(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
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

	logger.Debug().Str("session", cfg.SessionID).Str("revokedMsgID", revokedMsgID).Msg("[CW] handling message revoke")

	msg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, revokedMsgID)
	if err != nil {
		return
	}

	if msg.CWMessageID == nil || msg.CWConversationID == nil {
		return
	}

	client := s.clientFn(cfg)
	if err := client.DeleteMessage(ctx, *msg.CWConversationID, *msg.CWMessageID); err != nil {
		logger.Warn().Err(err).Str("revokedMsgID", revokedMsgID).Msg("[CW] failed to delete Chatwoot message on revoke")
	}
}

func (s *Service) handleEdit(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
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

	newText := extractTextFromMessage(editedMessage)
	if newText == "" {
		return
	}

	logger.Debug().Str("session", cfg.SessionID).Str("editedMsgID", editedMsgID).Str("newText", newText).Msg("[CW] handling message edit")

	msg, err := s.msgRepo.FindByID(ctx, cfg.SessionID, editedMsgID)
	if err != nil {
		return
	}

	if msg.CWMessageID == nil || msg.CWConversationID == nil {
		return
	}

	client := s.clientFn(cfg)
	if err := client.UpdateMessage(ctx, *msg.CWConversationID, *msg.CWMessageID, newText); err != nil {
		logger.Warn().Err(err).Str("editedMsgID", editedMsgID).Msg("[CW] failed to update Chatwoot message on edit")
	}
}

func (s *Service) handleConnected(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
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
			importCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			s.importHistory(importCtx, cfg.SessionID, period, 0)
		}()
	}
}

func (s *Service) handleDisconnected(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
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

func (s *Service) handleQR(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
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

	convID, err := s.findOrCreateBotConversation(ctx, cfg)
	if err != nil {
		logger.Warn().Err(err).Str("session", cfg.SessionID).Msg("Failed to find or create bot conversation for QR event")
		return
	}

	client := s.clientFn(cfg)
	qrPNG, err := generateQRCodePNG(qrContent)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to generate QR code")
		return
	}

	caption := "⚡️ QR Code gerado com sucesso!\n\nEscaneie o QR Code abaixo no WhatsApp para conectar."
	if data.PairingCode != "" {
		caption += fmt.Sprintf("\n\n*Código de pareamento:* %s-%s", data.PairingCode[:4], data.PairingCode[4:])
	}

	_, _ = client.CreateMessageWithAttachment(ctx, convID, caption, "qrcode.png", qrPNG, "image/png", "incoming", "")
}

func (s *Service) handleContact(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
	var data struct {
		JID    string `json:"JID"`
		Action struct {
			FullName  *string `json:"fullName"`
			FirstName *string `json:"firstName"`
			PnJID     *string `json:"pnJID"`
			LidJID    *string `json:"lidJID"`
		} `json:"Action"`
	}
	if err := parseEnvelopeData(payload, &data); err != nil {
		return
	}

	if data.JID == "" {
		return
	}

	jid := data.JID
	if strings.HasSuffix(jid, "@lid") {
		if data.Action.PnJID != nil && *data.Action.PnJID != "" {
			jid = *data.Action.PnJID
			if !strings.Contains(jid, "@") {
				jid = jid + "@s.whatsapp.net"
			}
		} else {
			jid = s.resolveJID(ctx, cfg.SessionID, jid)
		}
	}

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

func (s *Service) handlePushName(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
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

	jid := data.JID
	if strings.HasSuffix(jid, "@lid") {
		if data.JIDAlt != "" {
			jid = data.JIDAlt
		} else {
			jid = s.resolveJID(ctx, cfg.SessionID, jid)
		}
	}

	phone := extractPhone(jid)

	client := s.clientFn(cfg)
	contacts, _ := client.FilterContacts(ctx, phone)
	if len(contacts) == 0 {
		return
	}

	_ = client.UpdateContact(ctx, contacts[0].ID, UpdateContactReq{Name: data.NewPushName})
}

func (s *Service) handlePicture(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
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

	jid := data.JID
	if strings.HasSuffix(jid, "@lid") {
		jid = s.resolveJID(ctx, cfg.SessionID, jid)
	}

	phone := extractPhone(jid)

	var avatarURL string
	if s.profilePicGetter != nil {
		url, err := s.profilePicGetter.GetProfilePicture(ctx, cfg.SessionID, data.JID)
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

func (s *Service) handleGroupInfo(ctx context.Context, cfg *ChatwootConfig, payload []byte) error {
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
		return nil
	}
	if data.JID == "" {
		return nil
	}

	groupJID := data.JID
	convID, err := s.findOrCreateConversation(ctx, cfg, groupJID, "")
	if err != nil {
		logger.Warn().Err(err).Str("group", groupJID).Msg("[CW] failed to get group conversation for event")
		return err
	}

	client := s.clientFn(cfg)

	senderPhone := ""
	if data.SenderPN != nil && *data.SenderPN != "" {
		senderPhone = extractPhone(*data.SenderPN)
	} else if data.Sender != nil && *data.Sender != "" {
		senderJID := *data.Sender
		if strings.HasSuffix(senderJID, "@lid") {
			senderJID = s.resolveJID(ctx, cfg.SessionID, senderJID)
		}
		senderPhone = extractPhone(senderJID)
	}

	var notifications []string

	for _, jid := range data.Join {
		pJID := jid
		if strings.HasSuffix(pJID, "@lid") {
			pJID = s.resolveJID(ctx, cfg.SessionID, pJID)
		}
		notifications = append(notifications, fmt.Sprintf("➕ %s entrou no grupo", extractPhone(pJID)))
	}
	for _, jid := range data.Leave {
		pJID := jid
		if strings.HasSuffix(pJID, "@lid") {
			pJID = s.resolveJID(ctx, cfg.SessionID, pJID)
		}
		phone := extractPhone(pJID)
		if senderPhone != "" && senderPhone != phone {
			notifications = append(notifications, fmt.Sprintf("➖ %s foi removido do grupo por %s", phone, senderPhone))
		} else {
			notifications = append(notifications, fmt.Sprintf("🚪 %s saiu do grupo", phone))
		}
	}
	for _, jid := range data.Promote {
		pJID := jid
		if strings.HasSuffix(pJID, "@lid") {
			pJID = s.resolveJID(ctx, cfg.SessionID, pJID)
		}
		notifications = append(notifications, fmt.Sprintf("⬆️ %s foi promovido a admin por %s", extractPhone(pJID), senderPhone))
	}
	for _, jid := range data.Demote {
		pJID := jid
		if strings.HasSuffix(pJID, "@lid") {
			pJID = s.resolveJID(ctx, cfg.SessionID, pJID)
		}
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

func (s *Service) handleHistorySync(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
	logger.Debug().Str("session", cfg.SessionID).Msg("[CW] HistorySync received (no-op until import triggered)")
}
