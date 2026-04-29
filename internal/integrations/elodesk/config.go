package elodesk

import "time"

// Config espelha internal/integrations/chatwoot/Config.
// Diferenças:
//   - Sem AccountID (elodesk é single-tenant por instalação).
//   - InboxID → InboxIdentifier (string). Elodesk usa inbox identifier
//     no path /public/api/v1/inboxes/{identifier}/...
//   - APIToken é o `api-access-token` do elodesk.
//   - HMACToken assina o webhook outbound.
//
// Tokens usam json:"-" para não vazar em logs ou responses automáticas.
type Config struct {
	SessionID       string `json:"sessionId"`
	URL             string `json:"url"`
	InboxIdentifier string `json:"inboxIdentifier"`
	APIToken        string `json:"-"`
	HMACToken       string `json:"-"`
	WebhookSecret   string `json:"-"`
	UserAccessToken string `json:"-"`
	AccountID       int    `json:"accountId"`
	ChannelID       int64  `json:"-"`
	// InboxName é transient — usado apenas durante o auto-provision para
	// nomear a inbox criada no Elodesk. Não persistido no wz_elodesk.
	InboxName       string    `json:"-"`
	SignMsg         bool      `json:"signMsg"`
	SignDelimiter   string    `json:"signDelimiter"`
	ReopenConv      bool      `json:"reopenConv"`
	MergeBRContacts bool      `json:"mergeBrContacts"`
	IgnoreGroups    bool      `json:"ignoreGroups"`
	IgnoreJIDs      []string  `json:"ignoreJids"`
	PendingConv     bool      `json:"pendingConv"`
	Enabled         bool      `json:"enabled"`
	ImportOnConnect bool      `json:"importOnConnect"`
	ImportPeriod    string    `json:"importPeriod"`
	TextTimeout     int       `json:"textTimeout"`
	MediaTimeout    int       `json:"mediaTimeout"`
	LargeTimeout    int       `json:"largeTimeout"`
	MessageRead     bool      `json:"messageRead"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
