package chatwoot

import "time"

type ChatwootConfig struct {
	SessionID           string    `json:"sessionId"`
	URL                 string    `json:"url"`
	AccountID           int       `json:"accountId"`
	Token               string    `json:"token"`
	InboxID             int       `json:"inboxId"`
	InboxName           string    `json:"inboxName"`
	SignMsg             bool      `json:"signMsg"`
	SignDelimiter       string    `json:"signDelimiter"`
	ReopenConversation  bool      `json:"reopenConversation"`
	MergeBRContacts     bool      `json:"mergeBrContacts"`
	IgnoreGroups        bool      `json:"ignoreGroups"`
	IgnoreJIDs          []string  `json:"ignoreJids"`
	ConversationPending bool      `json:"conversationPending"`
	Enabled             bool      `json:"enabled"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}
