package chatwoot

import (
	"fmt"
	"strings"
)

func shouldIgnoreJID(chatJID string, ignoreGroups bool, ignoreJIDs []string) bool {
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

func extractPhone(jid string) string {
	jid = strings.Split(jid, "@")[0]
	jid = strings.TrimPrefix(jid, "+")
	return jid
}

func addOrRemoveBR9thDigit(phone string) string {
	if !strings.HasPrefix(phone, "55") {
		return phone
	}
	parts := strings.SplitN(phone, "", 13)
	if len(parts) < 12 {
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
