package chatwoot

import (
	"context"
	"fmt"
	"strings"

	"wzap/internal/logger"
)

func shouldIgnoreJID(chatJID string, ignoreGroups bool, ignoreJIDs []string) bool {
	if chatJID == "status@broadcast" {
		return true
	}
	if strings.HasSuffix(chatJID, "@newsletter") {
		return true
	}

	if ignoreGroups && strings.HasSuffix(chatJID, "@g.us") {
		return true
	}

	for _, jid := range ignoreJIDs {
		if jid == "@g.us" && strings.HasSuffix(chatJID, "@g.us") {
			return true
		}
		if jid == "@s.whatsapp.net" && strings.HasSuffix(chatJID, "@s.whatsapp.net") {
			return true
		}
		if jid == chatJID {
			return true
		}
	}

	return false
}

func jidsContainGroup(ignoreJIDs []string) bool {
	for _, jid := range ignoreJIDs {
		if jid == "@g.us" {
			return true
		}
	}
	return false
}

func isValidWhatsAppJID(jid string) bool {
	return strings.HasSuffix(jid, "@s.whatsapp.net") ||
		strings.HasSuffix(jid, "@g.us") ||
		strings.HasSuffix(jid, "@lid") ||
		strings.HasSuffix(jid, "@broadcast")
}

func extractPhone(jid string) string {
	jid = strings.Split(jid, "@")[0]
	jid = strings.TrimPrefix(jid, "+")
	return jid
}

func normalizeBRPhone(phone string) string {
	if !strings.HasPrefix(phone, "55") {
		return phone
	}
	if len(phone) < 12 {
		return phone
	}
	ddd := phone[2:4]
	number := phone[4:]
	if len(number) == 8 {
		return "55" + ddd + "9" + number
	}
	if len(number) == 9 && number[0] == '9' {
		return "55" + ddd + number[1:]
	}
	return phone
}

func formatGroupContent(phone, pushName, body string, fromMe bool) string {
	if fromMe {
		return body
	}
	return fmt.Sprintf("**+%s - %s:**\n\n%s", phone, pushName, body)
}

func (s *Service) resolveJID(ctx context.Context, sessionID, jid string) string {
	if !strings.HasSuffix(jid, "@lid") || s.jidResolver == nil {
		return jid
	}
	if pn := s.jidResolver.GetPNForLID(ctx, sessionID, jid); pn != "" {
		return pn + "@s.whatsapp.net"
	}
	return jid
}

func urlFilename(rawURL string) string {
	u := strings.Split(strings.Split(rawURL, "?")[0], "#")[0]
	parts := strings.Split(u, "/")
	return parts[len(parts)-1]
}

func (s *Service) resolvePhoneToJID(ctx context.Context, sessionID, phone string) string {
	if s.numberChecker == nil {
		return phone + "@s.whatsapp.net"
	}

	variant := normalizeBRPhone(phone)
	phones := []string{"+" + phone}
	if variant != phone {
		phones = append(phones, "+"+variant)
	}

	resolved, err := s.numberChecker.IsOnWhatsApp(ctx, sessionID, phones)
	if err != nil {
		logger.Warn().Str("component", "chatwoot").Err(err).Str("phone", phone).Msg("IsOnWhatsApp check failed, using phone as-is")
		return phone + "@s.whatsapp.net"
	}

	for _, p := range phones {
		if jid, ok := resolved[p]; ok {
			return jid
		}
	}

	return phone + "@s.whatsapp.net"
}
