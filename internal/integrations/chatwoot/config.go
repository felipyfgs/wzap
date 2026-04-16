package chatwoot

import (
	"net/url"
	"strings"
	"time"
)

type Config struct {
	SessionID       string    `json:"sessionId"`
	URL             string    `json:"url"`
	AccountID       int       `json:"accountId"`
	Token           string    `json:"token"`
	WebhookToken    string    `json:"webhookToken"`
	InboxID         int       `json:"inboxId"`
	InboxName       string    `json:"inboxName"`
	InboxType       string    `json:"inboxType"`
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
	DatabaseURI     string    `json:"databaseUri"`
	RedisURL        string    `json:"redisUrl"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func maskURL(rawURL string) string {
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
