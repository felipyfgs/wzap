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

type SendTextReq struct {
	To   string `json:"to" validate:"required"` // Phone number or group JID
	Text string `json:"text" validate:"required"`
}

type SendMediaReq struct {
	To       string `json:"to" validate:"required"`
	MimeType string `json:"mime_type" validate:"required"`
	Caption  string `json:"caption"`
	Filename string `json:"filename"` // For documents
	Base64   string `json:"base64"`   // For base64 upload
}

// Emitted when a message is sent or received
type MessageEvent struct {
	SessionID string      `json:"session_id"`
	ID        string      `json:"id"`
	From      string      `json:"from"`
	To        string      `json:"to"`
	Timestamp time.Time   `json:"timestamp"`
	Type      MessageType `json:"type"`
	Text      string      `json:"text,omitempty"`
	MediaURL  string      `json:"media_url,omitempty"`
}
