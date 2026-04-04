package model

import "time"

type Message struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionId"`
	ChatJID   string    `json:"chatJid"`
	SenderJID string    `json:"senderJid"`
	FromMe    bool      `json:"fromMe"`
	MsgType   string    `json:"msgType"`
	Body      string    `json:"body"`
	MediaType string    `json:"mediaType,omitempty"`
	MediaURL  string    `json:"mediaUrl,omitempty"`
	Raw       any       `json:"raw,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	CreatedAt time.Time `json:"createdAt"`

	CWMessageID      *int    `json:"cwMessageId,omitempty"`
	CWConversationID *int    `json:"cwConversationId,omitempty"`
	CWSourceID       *string `json:"cwSourceId,omitempty"`
}
