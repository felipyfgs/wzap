package chatwoot

import (
	"net/url"
	"strings"
	"time"
)

type Config struct {
	SessionID           string    `json:"sessionId"`
	URL                 string    `json:"url"`
	AccountID           int       `json:"accountId"`
	Token               string    `json:"token"`
	WebhookToken        string    `json:"webhookToken"`
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
	ImportOnConnect     bool      `json:"importOnConnect"`
	ImportPeriod        string    `json:"importPeriod"`
	TimeoutTextSeconds  int       `json:"timeoutTextSeconds"`
	TimeoutMediaSeconds int       `json:"timeoutMediaSeconds"`
	TimeoutLargeSeconds int       `json:"timeoutLargeSeconds"`
	MessageRead         bool      `json:"messageRead"`
	DatabaseURI         string    `json:"databaseUri"`
	RedisURL            string    `json:"redisUrl"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

func maskRedisURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if parsed.User == nil {
		return rawURL
	}
	if _, hasPassword := parsed.User.Password(); !hasPassword {
		return rawURL
	}

	parsed.User = nil
	masked := parsed.String()
	return strings.Replace(masked, "://", "://***@", 1)
}

func maskDatabaseURI(rawURI string) string {
	if rawURI == "" {
		return ""
	}

	parsed, err := url.Parse(rawURI)
	if err != nil {
		return rawURI
	}
	if parsed.User == nil {
		return rawURI
	}
	if _, hasPassword := parsed.User.Password(); !hasPassword {
		return rawURI
	}

	parsed.User = nil
	masked := parsed.String()
	return strings.Replace(masked, "://", "://***@", 1)
}
