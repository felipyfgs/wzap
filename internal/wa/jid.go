package wa

import "strings"

// JID server suffixes used by WhatsApp multidevice.
// Reference: https://www.mintlify.com/whiskeysockets/Baileys/concepts/whatsapp-ids
//
// Server enum mapping:
//
//	DEFAULT (0)  → @s.whatsapp.net  (regular user)
//	GROUP (1)    → @g.us            (group chat)
//	LID (1)      → @lid             (local/hidden ID for privacy)
//	HOSTED (128) → @s.whatsapp.net  (hosted business)
//	HOSTED_LID (129) → @lid        (hosted + local ID)
const (
	ServerUser      = "@s.whatsapp.net"  // individual user JIDs
	ServerGroup     = "@g.us"            // group chat JIDs
	ServerBroadcast = "@broadcast"       // broadcast list JIDs
	ServerNewsletter = "@newsletter"     // channel/status JIDs
	ServerLID       = "@lid"             // local/hidden ID (privacy)
)

// IsAnyJIDServer reports whether jid contains any known WhatsApp server suffix.
func IsAnyJIDServer(jid string) bool {
	if jid == "" {
		return false
	}
	for _, s := range []string{
		ServerUser,
		ServerGroup,
		ServerBroadcast,
		ServerNewsletter,
		ServerLID,
	} {
		if strings.HasSuffix(jid, s) {
			return true
		}
	}
	return false
}

// EnsureJIDSuffix adds the user server suffix if the JID doesn't already have one.
// Used for converting raw phone numbers (e.g. "5511999999999") to proper JIDs
// (e.g. "5511999999999@s.whatsapp.net").
func EnsureJIDSuffix(jid string) string {
	if jid == "" || IsAnyJIDServer(jid) {
		return jid
	}
	return jid + ServerUser
}
