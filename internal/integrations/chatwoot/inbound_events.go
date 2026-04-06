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
		JID       string `json:"JID"`
		FirstName string `json:"FirstName"`
		FullName  string `json:"FullName"`
	}
	if err := parseEnvelopeData(payload, &data); err != nil {
		return
	}

	if data.JID == "" {
		return
	}

	phone := extractPhone(data.JID)
	name := data.FullName
	if name == "" {
		name = data.FirstName
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
		JID      string `json:"JID"`
		PushName string `json:"PushName"`
	}
	if err := parseEnvelopeData(payload, &data); err != nil {
		return
	}

	if data.JID == "" || data.PushName == "" {
		return
	}

	phone := extractPhone(data.JID)

	client := s.clientFn(cfg)
	contacts, _ := client.FilterContacts(ctx, phone)
	if len(contacts) == 0 {
		return
	}

	_ = client.UpdateContact(ctx, contacts[0].ID, UpdateContactReq{Name: data.PushName})
}

func (s *Service) handlePicture(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
	var data struct {
		JID       string `json:"JID"`
		PictureID string `json:"ID"`
		URL       string `json:"URL"`
		IsGroup   bool   `json:"IsGroup"`
	}
	if err := parseEnvelopeData(payload, &data); err != nil {
		return
	}

	if data.JID == "" || data.URL == "" {
		return
	}

	phone := extractPhone(data.JID)

	client := s.clientFn(cfg)
	contacts, _ := client.FilterContacts(ctx, phone)
	if len(contacts) == 0 {
		return
	}

	_ = client.UpdateContact(ctx, contacts[0].ID, UpdateContactReq{
		AdditionalAttributes: map[string]any{"avatar_url": data.URL},
	})
}

func (s *Service) handleGroupInfo(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
	var data struct {
		JID       string `json:"JID"`
		Timestamp int64  `json:"Timestamp"`
		Updates   []struct {
			Type       string `json:"Type"`
			JID        string `json:"JID"`
			Participant string `json:"Participant"`
			Name       string `json:"Name"`
			Description string `json:"Description"`
			Announce    bool   `json:"Announce"`
			Ephemeral   int    `json:"Ephemeral"`
			NewInviteLink string `json:"NewInviteLink"`
		} `json:"Updates"`
		Sender struct {
			User   string `json:"User"`
			Server string `json:"Server"`
		} `json:"Sender"`
	}
	if err := parseEnvelopeData(payload, &data); err != nil {
		return
	}
	if data.JID == "" {
		return
	}

	groupJID := data.JID
	convID, err := s.findOrCreateConversation(ctx, cfg, groupJID, "")
	if err != nil {
		logger.Warn().Err(err).Str("group", groupJID).Msg("[CW] failed to get group conversation for event")
		return
	}

	client := s.clientFn(cfg)
	sender := data.Sender.User

	for _, upd := range data.Updates {
		var notif string
		participant := upd.Participant
		if participant == "" {
			participant = upd.JID
		}
		participantPhone := extractPhone(participant)

		switch upd.Type {
		case "add":
			notif = fmt.Sprintf("➕ %s entrou no grupo", participantPhone)
		case "remove":
			notif = fmt.Sprintf("➖ %s foi removido do grupo por %s", participantPhone, sender)
		case "leave":
			notif = fmt.Sprintf("🚪 %s saiu do grupo", participantPhone)
		case "promote":
			notif = fmt.Sprintf("⬆️ %s foi promovido a admin por %s", participantPhone, sender)
		case "demote":
			notif = fmt.Sprintf("⬇️ %s foi rebaixado de admin por %s", participantPhone, sender)
		case "subject":
			notif = fmt.Sprintf("📝 Nome do grupo alterado para: %s", upd.Name)
		case "description":
			notif = fmt.Sprintf("📋 Descrição do grupo atualizada: %s", upd.Description)
		case "invite":
			if upd.NewInviteLink != "" {
				notif = fmt.Sprintf("🔗 Link de convite do grupo: %s", upd.NewInviteLink)
			}
		case "announce":
			if upd.Announce {
				notif = "🔒 Grupo restrito a admins"
			} else {
				notif = "🔓 Grupo aberto para todos enviarem mensagens"
			}
		case "ephemeral":
			if upd.Ephemeral > 0 {
				notif = fmt.Sprintf("⏳ Mensagens temporárias ativadas (%d segundos)", upd.Ephemeral)
			} else {
				notif = "⏳ Mensagens temporárias desativadas"
			}
		}

		if notif != "" && !strings.HasSuffix(groupJID, "@s.whatsapp.net") {
			_, _ = client.CreateMessage(ctx, convID, MessageReq{
				Content:     notif,
				MessageType: "activity",
			})
		}
	}
}

func (s *Service) handleHistorySync(ctx context.Context, cfg *ChatwootConfig, payload []byte) {
	logger.Debug().Str("session", cfg.SessionID).Msg("[CW] HistorySync received (no-op until import triggered)")
}
