package model

import "time"

type MessageType string

const (
	MsgTypeText     MessageType = "text"
	MsgTypeImage    MessageType = "image"
	MsgTypeVideo    MessageType = "video"
	MsgTypeAudio    MessageType = "audio"
	MsgTypeDocument MessageType = "document"
)

type MessageEvent struct {
	SessionID string      `json:"sessionId"`
	ID        string      `json:"id"`
	From      string      `json:"from"`
	To        string      `json:"to"`
	Timestamp time.Time   `json:"timestamp"`
	Type      MessageType `json:"type"`
	Text      string      `json:"text,omitempty"`
	MediaURL  string      `json:"mediaUrl,omitempty"`
}
