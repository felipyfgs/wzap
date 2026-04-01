package dto

import (
	"time"

	"wzap/internal/model"
)

type SessionProxy struct {
	Host     string `json:"Host,omitempty"`
	Port     int    `json:"Port,omitempty"`
	Protocol string `json:"Protocol,omitempty"`
	Username string `json:"Username,omitempty"`
	Password string `json:"Password,omitempty"`
}

type SessionSettings struct {
	AlwaysOnline  bool   `json:"AlwaysOnline"`
	RejectCall    bool   `json:"RejectCall"`
	MsgRejectCall string `json:"MsgRejectCall,omitempty"`
	ReadMessages  bool   `json:"ReadMessages"`
	IgnoreGroups  bool   `json:"IgnoreGroups"`
	IgnoreStatus  bool   `json:"IgnoreStatus"`
}

type WebhookResp struct {
	ID          string    `json:"Id"`
	SessionID   string    `json:"SessionId"`
	URL         string    `json:"URL"`
	Secret      string    `json:"Secret,omitempty"`
	Events      []string  `json:"Events"`
	Enabled     bool      `json:"Enabled"`
	NatsEnabled bool      `json:"NatsEnabled"`
	CreatedAt   time.Time `json:"CreatedAt"`
	UpdatedAt   time.Time `json:"UpdatedAt"`
}

type WebhookCreateInline struct {
	URL    string   `json:"URL"`
	Events []string `json:"Events,omitempty"`
}

type SessionCreateReq struct {
	Name     string               `json:"Name" validate:"required"`
	APIKey   string               `json:"ApiKey,omitempty"`
	Proxy    SessionProxy         `json:"Proxy,omitempty"`
	Webhook  *WebhookCreateInline `json:"Webhook,omitempty"`
	Settings SessionSettings      `json:"Settings,omitempty"`
}

type SessionResp struct {
	ID        string          `json:"Id"`
	Name      string          `json:"Name"`
	JID       string          `json:"Jid,omitempty"`
	Connected int             `json:"Connected"`
	Status    string          `json:"Status"`
	Proxy     SessionProxy    `json:"Proxy"`
	Settings  SessionSettings `json:"Settings"`
	CreatedAt time.Time       `json:"CreatedAt"`
	UpdatedAt time.Time       `json:"UpdatedAt"`
}

type SessionCreatedResp struct {
	ID        string          `json:"Id"`
	Name      string          `json:"Name"`
	APIKey    string          `json:"ApiKey"`
	JID       string          `json:"Jid,omitempty"`
	Connected int             `json:"Connected"`
	Status    string          `json:"Status"`
	Proxy     SessionProxy    `json:"Proxy"`
	Settings  SessionSettings `json:"Settings"`
	CreatedAt time.Time       `json:"CreatedAt"`
	UpdatedAt time.Time       `json:"UpdatedAt"`
	Webhook   *WebhookResp    `json:"Webhook,omitempty"`
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
