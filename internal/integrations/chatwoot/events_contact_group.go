package chatwoot

import (
	"context"
	"fmt"

	"wzap/internal/logger"
)

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
	// Chatwoot Cloud inbox rejects source_ids > 15 digits (group JIDs have 18+),
	// so we skip group notifications entirely in cloud mode.
	if cfg.InboxType == "cloud" {
		return nil
	}
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
		return fmt.Errorf("find or create group conversation for %s: %w", groupJID, err)
	}

	client := s.clientFn(cfg)

	senderPhone := ""
	if data.SenderPN != nil && *data.SenderPN != "" {
		senderPhone = extractPhone(*data.SenderPN)
	} else if data.Sender != nil && *data.Sender != "" {
		senderPhone = extractPhone(s.resolveLID(ctx, cfg.SessionID, *data.Sender))
	}

	var notifications []string

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
