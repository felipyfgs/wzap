package dto

import (
	"time"

	"wzap/internal/model"
)

type WebhookCreateInline struct {
	URL    string            `json:"url"`
	Events []model.EventType `json:"events,omitempty"`
}

type SessionCreateReq struct {
	Name     string                `json:"name" validate:"required"`
	APIKey   string                `json:"apiKey,omitempty"`
	Proxy    model.SessionProxy    `json:"proxy,omitempty"`
	Webhook  *WebhookCreateInline  `json:"webhook,omitempty"`
	Settings model.SessionSettings `json:"settings,omitempty"`
}

// SessionResp is the public representation of a session — APIKey is never included.
type SessionResp struct {
	ID        string                `json:"id"`
	Name      string                `json:"name"`
	JID       string                `json:"jid,omitempty"`
	Connected int                   `json:"connected"`
	Status    string                `json:"status"`
	Proxy     model.SessionProxy    `json:"proxy"`
	Settings  model.SessionSettings `json:"settings"`
	CreatedAt time.Time             `json:"createdAt"`
	UpdatedAt time.Time             `json:"updatedAt"`
}

func SessionToResp(s model.Session) SessionResp {
	return SessionResp{
		ID:        s.ID,
		Name:      s.Name,
		JID:       s.JID,
		Connected: s.Connected,
		Status:    s.Status,
		Proxy:     s.Proxy,
		Settings:  s.Settings,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

type SessionCreatedResp struct {
	model.Session
	Webhook *model.Webhook `json:"webhook,omitempty"`
}
