package model

import "time"

type Status struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"sessionId"`
	SenderJID  string    `json:"senderJid"`
	SenderName string    `json:"senderName,omitempty"`
	FromMe     bool      `json:"fromMe"`
	StatusType string    `json:"statusType"`
	Body       string    `json:"body"`
	MediaType  string    `json:"mediaType,omitempty"`
	MediaURL   string    `json:"mediaUrl,omitempty"`
	Raw        any       `json:"raw,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	ExpiresAt  time.Time `json:"expiresAt"`
	CreatedAt  time.Time `json:"createdAt"`
}
