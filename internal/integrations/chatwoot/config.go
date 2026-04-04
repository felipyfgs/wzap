package chatwoot

import "time"

type ChatwootConfig struct {
	SessionID        string    `json:"sessionId"`
	URL              string    `json:"url"`
	AccountID        int       `json:"accountId"`
	Token            string    `json:"token"`
	InboxID          int       `json:"inboxId"`
	InboxName        string    `json:"inboxName"`
	SignMsg          bool      `json:"signMsg"`
	SignDelimiter    string    `json:"signDelimiter"`
	ReopenConversation bool    `json:"reopenConversation"`
	MergeBRContacts  bool      `json:"mergeBrContacts"`
	IgnoreGroups     bool      `json:"ignoreGroups"`
	Enabled          bool      `json:"enabled"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}
