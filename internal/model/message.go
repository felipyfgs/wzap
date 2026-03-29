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
	MimeType string `json:"mimeType" validate:"required"`
	Caption  string `json:"caption"`
	Filename string `json:"filename"` // For documents
	Base64   string `json:"base64"`   // For base64 upload
}

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

type SendContactReq struct {
	To    string `json:"to" validate:"required"`
	Name  string `json:"name" validate:"required"`
	Vcard string `json:"vcard" validate:"required"`
}

type SendLocationReq struct {
	To      string  `json:"to" validate:"required"`
	Lat     float64 `json:"lat" validate:"required"`
	Lng     float64 `json:"lng" validate:"required"`
	Name    string  `json:"name"`
	Address string  `json:"address"`
}

type SendPollReq struct {
	To              string   `json:"to" validate:"required"`
	Name            string   `json:"name" validate:"required"`
	Options         []string `json:"options" validate:"required,min=2"`
	SelectableCount int      `json:"selectableCount"`
}

type SendStickerReq struct {
	To       string `json:"to" validate:"required"`
	MimeType string `json:"mime_type" validate:"required"`
	Base64   string `json:"base64" validate:"required"`
}

type SendLinkReq struct {
	To          string `json:"to" validate:"required"`
	URL         string `json:"url" validate:"required"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type EditMessageReq struct {
	To        string `json:"to" validate:"required"`
	MessageID string `json:"messageId" validate:"required"`
	Text      string `json:"text" validate:"required"`
}

type DeleteMessageReq struct {
	To        string `json:"to" validate:"required"`
	MessageID string `json:"messageId" validate:"required"`
}

type ReactMessageReq struct {
	To        string `json:"to" validate:"required"`
	MessageID string `json:"messageId" validate:"required"`
	Reaction  string `json:"reaction" validate:"required"` // emoji or empty to remove
}

type MarkReadReq struct {
	To        string `json:"to" validate:"required"`
	MessageID string `json:"messageId" validate:"required"`
}

type SetPresenceReq struct {
	To       string `json:"to" validate:"required"`
	Presence string `json:"presence" validate:"required"` // typing, recording, paused
}
