package chatwoot

import (
	"context"
	"fmt"
	"strings"

	"wzap/internal/logger"
)

func (s *Service) findOrCreateConversation(ctx context.Context, cfg *Config, chatJID, pushName string) (int, error) {
	if convID, _, ok := s.cache.GetConv(ctx, cfg.SessionID, chatJID); ok {
		return convID, nil
	}

	key := cfg.SessionID + "+" + chatJID
	result, err, _ := s.convFlight.Do(key, func() (any, error) {
		if convID, _, ok := s.cache.GetConv(ctx, cfg.SessionID, chatJID); ok {
			return convID, nil
		}
		return s.upsertConversation(ctx, cfg, chatJID, pushName)
	})
	if err != nil {
		return 0, err
	}

	convID, ok := result.(int)
	if !ok {
		return 0, fmt.Errorf("invalid conversation id type %T", result)
	}
	return convID, nil
}

func (s *Service) upsertConversation(ctx context.Context, cfg *Config, chatJID, pushName string) (int, error) {
	client := s.clientFn(cfg)
	isGroup := strings.HasSuffix(chatJID, "@g.us")

	contactName := pushName
	if s.contactNameGetter != nil {
		if name := s.contactNameGetter.GetContactName(ctx, cfg.SessionID, chatJID); name != "" {
			contactName = name
		}
	}

	var contacts []Contact
	if isGroup {
		all, _ := client.SearchContacts(ctx, chatJID)
		for _, c := range all {
			if c.Identifier == chatJID {
				contacts = append(contacts, c)
				break
			}
		}
	} else {
		phone := extractPhone(chatJID)
		contacts, _ = client.FilterContacts(ctx, phone)
		if strings.HasPrefix(phone, "55") {
			phoneVariant := normalizeBRPhone(phone)
			if phoneVariant != phone {
				contacts2, _ := client.FilterContacts(ctx, phoneVariant)
				contacts = deduplicateContacts(append(contacts, contacts2...))
			}
		}
		if cfg.MergeBRContacts && len(contacts) == 2 {
			phone0 := strings.TrimPrefix(contacts[0].PhoneNumber, "+")
			phone1 := strings.TrimPrefix(contacts[1].PhoneNumber, "+")
			var baseID, mergeeID int
			if len(phone0) == 14 && len(phone1) == 13 {
				baseID = contacts[0].ID
				mergeeID = contacts[1].ID
			} else if len(phone1) == 14 && len(phone0) == 13 {
				baseID = contacts[1].ID
				mergeeID = contacts[0].ID
			}
			if baseID > 0 && mergeeID > 0 {
				if err := client.MergeContacts(ctx, baseID, mergeeID); err != nil {
					logger.Warn().Str("component", "chatwoot").Err(err).Int("baseID", baseID).Int("mergeeID", mergeeID).Msg("failed to merge BR contacts")
				}
				contacts = []Contact{{ID: baseID}}
			}
		}
	}

	var contactID int
	if len(contacts) == 0 {
		name := contactName
		if name == "" {
			if isGroup {
				name = chatJID
			} else {
				name = extractPhone(chatJID)
			}
		}
		var avatarURL string
		if s.picGetter != nil {
			if picURL, err := s.picGetter.GetProfilePicture(ctx, cfg.SessionID, chatJID); err == nil {
				avatarURL = picURL
			}
		}
		req := CreateContactReq{
			InboxID:    cfg.InboxID,
			Name:       name,
			Identifier: chatJID,
			AvatarURL:  avatarURL,
		}
		if !isGroup {
			req.PhoneNumber = "+" + extractPhone(chatJID)
		}
		contact, err := client.CreateContact(ctx, req)
		if err != nil {
			return 0, fmt.Errorf("failed to create contact: %w", err)
		}
		logger.Debug().Str("component", "chatwoot").Int("contactID", contact.ID).Str("jid", chatJID).Msg("contact created")
		contactID = contact.ID
		if cfg.DatabaseURI != "" {
			if err := addLabelToContact(ctx, cfg.DatabaseURI, cfg.InboxName, contact.ID); err != nil {
				logger.Warn().Str("component", "chatwoot").Err(err).Int("contactID", contact.ID).Msg("failed to add label to contact")
			}
		}
	} else {
		contactID = contacts[0].ID
		logger.Debug().Str("component", "chatwoot").Int("contactID", contactID).Str("jid", chatJID).Msg("contact found")

		update := UpdateContactReq{}
		existingPhone := extractPhone(chatJID)
		if contactName != "" && (contacts[0].Name == "" || contacts[0].Name == existingPhone) {
			update.Name = contactName
		}
		if contacts[0].Identifier == "" || strings.HasSuffix(contacts[0].Identifier, "@lid") {
			update.Identifier = chatJID
			logger.Debug().Str("component", "chatwoot").Int("contactID", contactID).Str("identifier", chatJID).Msg("updating contact identifier")
		}
		if s.picGetter != nil {
			if picURL, err := s.picGetter.GetProfilePicture(ctx, cfg.SessionID, chatJID); err == nil && picURL != "" {
				if urlFilename(picURL) != urlFilename(contacts[0].Thumbnail) {
					update.AvatarURL = picURL
				}
			}
		}
		if update.Name != "" || update.Identifier != "" || update.AvatarURL != "" {
			if err := client.UpdateContact(ctx, contactID, update); err != nil {
				logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Int("contactID", contactID).Msg("Failed to update contact info")
			}
		}
	}

	conversations, err := client.ListContactConversations(ctx, contactID)
	if err != nil {
		return 0, fmt.Errorf("failed to list conversations: %w", err)
	}

	for _, conv := range conversations {
		if conv.InboxID == cfg.InboxID {
			if conv.Status == "resolved" && cfg.ReopenConversation {
				reopenStatus := "open"
				if cfg.ConversationPending {
					reopenStatus = "pending"
				}
				if err := client.UpdateConversationStatus(ctx, conv.ID, reopenStatus); err != nil {
					logger.Warn().Str("component", "chatwoot").Err(err).Int("convID", conv.ID).Msg("Failed to reopen conversation")
				}
				s.cache.SetConv(ctx, cfg.SessionID, chatJID, conv.ID, contactID)
				return conv.ID, nil
			}
			if conv.Status != "resolved" {
				s.cache.SetConv(ctx, cfg.SessionID, chatJID, conv.ID, contactID)
				return conv.ID, nil
			}
		}
	}

	req := CreateConversationReq{
		InboxID:   cfg.InboxID,
		SourceID:  chatJID,
		ContactID: contactID,
	}
	if cfg.ConversationPending {
		req.Status = "pending"
	}

	conv, err := client.CreateConversation(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("failed to create conversation: %w", err)
	}

	s.cache.SetConv(ctx, cfg.SessionID, chatJID, conv.ID, contactID)
	return conv.ID, nil
}

func (s *Service) ensureBotConv(ctx context.Context, cfg *Config) (int, error) {
	contactID, err := s.ensureBotContact(ctx, cfg)
	if err != nil {
		return 0, err
	}

	client := s.clientFn(cfg)
	conversations, err := client.ListContactConversations(ctx, contactID)
	if err != nil {
		return 0, err
	}

	for _, conv := range conversations {
		if conv.InboxID == cfg.InboxID && conv.Status != "resolved" {
			return conv.ID, nil
		}
	}

	conv, err := client.CreateConversation(ctx, CreateConversationReq{
		InboxID:   cfg.InboxID,
		ContactID: contactID,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create bot conversation: %w", err)
	}

	return conv.ID, nil
}

func (s *Service) findOpenBotConversation(ctx context.Context, cfg *Config) (int, bool) {
	contactID, err := s.ensureBotContact(ctx, cfg)
	if err != nil {
		return 0, false
	}

	client := s.clientFn(cfg)
	conversations, err := client.ListContactConversations(ctx, contactID)
	if err != nil {
		return 0, false
	}

	for _, conv := range conversations {
		if conv.InboxID == cfg.InboxID && conv.Status != "resolved" {
			return conv.ID, true
		}
	}

	return 0, false
}

func (s *Service) ensureBotContact(ctx context.Context, cfg *Config) (int, error) {
	client := s.clientFn(cfg)

	botName := cfg.InboxName
	if botName == "" {
		botName = "wzap"
	}
	botIdentifier := "bot@" + cfg.SessionID

	contacts, err := client.FilterContacts(ctx, botIdentifier)
	if err != nil {
		return 0, err
	}

	if len(contacts) == 0 {
		contact, err := client.CreateContact(ctx, CreateContactReq{
			InboxID:    cfg.InboxID,
			Name:       botName,
			Identifier: botIdentifier,
		})
		if err != nil {
			return 0, fmt.Errorf("failed to create bot contact: %w", err)
		}
		return contact.ID, nil
	}

	contactID := contacts[0].ID
	if contacts[0].Name != botName {
		if err := client.UpdateContact(ctx, contactID, UpdateContactReq{Name: botName}); err != nil {
			logger.Warn().Str("component", "chatwoot").Err(err).Int("contactID", contactID).Msg("Failed to update bot contact name")
		}
	}
	return contactID, nil
}

func (s *Service) webhookURL(sessionID string) string {
	base := s.serverURL
	if base == "" {
		base = "http://localhost:8080"
	}
	return fmt.Sprintf("%s/chatwoot/webhook/%s", base, sessionID)
}

func (s *Service) Configure(ctx context.Context, cfg *Config) error {
	if cfg.InboxName == "" {
		cfg.InboxName = "wzap"
	}

	client := s.clientFn(cfg)
	whURL := s.webhookURL(cfg.SessionID)

	inboxes, err := client.ListInboxes(ctx)
	if err != nil {
		inboxes = nil
	}

	found := false
	for _, inbox := range inboxes {
		if inbox.ID == cfg.InboxID {
			found = true
			if err := client.UpdateInboxWebhook(ctx, cfg.InboxID, whURL); err != nil {
				logger.Warn().Str("component", "chatwoot").Err(err).Str("session", cfg.SessionID).Int("inboxID", cfg.InboxID).Msg("Failed to update inbox webhook URL")
			}
			break
		}
	}

	if !found && cfg.InboxID == 0 {
		inbox, err := client.CreateInbox(ctx, cfg.InboxName, whURL)
		if err != nil {
			return fmt.Errorf("failed to auto-create inbox: %w", err)
		}
		cfg.InboxID = inbox.ID
	}

	cfg.Enabled = true
	if err := s.repo.Upsert(ctx, cfg); err != nil {
		return err
	}
	s.ClearConfigCache(cfg.SessionID)
	return nil
}

func deduplicateContacts(contacts []Contact) []Contact {
	seen := make(map[int]struct{}, len(contacts))
	result := make([]Contact, 0, len(contacts))
	for _, c := range contacts {
		if _, ok := seen[c.ID]; ok {
			continue
		}
		seen[c.ID] = struct{}{}
		result = append(result, c)
	}
	return result
}
