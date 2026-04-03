package wa

import (
	"strings"

	"go.mau.fi/whatsmeow/types"
)

// EnsureJIDSuffix adds the default user server suffix if the JID doesn't
// already contain a server part. Used for converting raw phone numbers
// (e.g. "5511999999999") to proper JIDs ("5511999999999@s.whatsapp.net").
func EnsureJIDSuffix(jid string) string {
	if jid == "" || strings.Contains(jid, "@") {
		return jid
	}
	return jid + "@" + types.DefaultUserServer
}
