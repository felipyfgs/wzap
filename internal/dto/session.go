package dto

import (
	"time"

	"wzap/internal/model"
)

type SessionProxy struct {
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type SessionSettings struct {
	AlwaysOnline  bool   `json:"alwaysOnline"`
	RejectCall    bool   `json:"rejectCall"`
	MsgRejectCall string `json:"msgRejectCall,omitempty"`
	ReadMessages  bool   `json:"readMessages"`
	IgnoreGroups  bool   `json:"ignoreGroups"`
	IgnoreStatus  bool   `json:"ignoreStatus"`
}

type WebhookResp struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"sessionId"`
	URL         string    `json:"url"`
	Secret      string    `json:"secret,omitempty"`
	Events      []string  `json:"events"`
	Enabled     bool      `json:"enabled"`
	NATSEnabled bool      `json:"natsEnabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type WebhookCreateInline struct {
	URL    string   `json:"url"`
	Events []string `json:"events,omitempty"`
}

type SessionCreateReq struct {
	Name     string               `json:"name" validate:"required"`
	APIKey   string               `json:"apiKey,omitempty"`
	Proxy    SessionProxy         `json:"proxy,omitempty"`
	Webhook  *WebhookCreateInline `json:"webhook,omitempty"`
	Settings SessionSettings      `json:"settings,omitempty"`
}

type SessionResp struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	JID       string          `json:"jid,omitempty"`
	Connected int             `json:"connected"`
	Status    string          `json:"status"`
	Proxy     SessionProxy    `json:"proxy"`
	Settings  SessionSettings `json:"settings"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type SessionCreatedResp struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	APIKey    string          `json:"apiKey"`
	JID       string          `json:"jid,omitempty"`
	Connected int             `json:"connected"`
	Status    string          `json:"status"`
	Proxy     SessionProxy    `json:"proxy"`
	Settings  SessionSettings `json:"settings"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
	Webhook   *WebhookResp    `json:"webhook,omitempty"`
}

func SessionToResp(s model.Session) SessionResp {
	return SessionResp{
		ID:        s.ID,
		Name:      s.Name,
		JID:       s.JID,
		Connected: s.Connected,
		Status:    s.Status,
		Proxy:     SessionProxy(s.Proxy),
		Settings:  SessionSettings(s.Settings),
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
