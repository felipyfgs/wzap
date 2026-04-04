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
	Name               string               `json:"name" validate:"required"`
	Token              string               `json:"token,omitempty"`
	Engine             string               `json:"engine,omitempty" validate:"omitempty,oneof=whatsmeow cloud_api"`
	PhoneNumberID      string               `json:"phoneNumberId,omitempty"`
	AccessToken        string               `json:"accessToken,omitempty"`
	BusinessAccountID  string               `json:"businessAccountId,omitempty"`
	AppSecret          string               `json:"appSecret,omitempty"`
	WebhookVerifyToken string               `json:"webhookVerifyToken,omitempty"`
	Proxy              SessionProxy         `json:"proxy,omitempty"`
	Webhook            *WebhookCreateInline `json:"webhook,omitempty"`
	Settings           SessionSettings      `json:"settings,omitempty"`
}

type SessionResp struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	JID               string          `json:"jid,omitempty"`
	Connected         int             `json:"connected"`
	Status            string          `json:"status"`
	Engine            string          `json:"engine,omitempty"`
	PhoneNumberID     string          `json:"phoneNumberId,omitempty"`
	BusinessAccountID string          `json:"businessAccountId,omitempty"`
	Proxy             SessionProxy    `json:"proxy"`
	Settings          SessionSettings `json:"settings"`
	PushName          string          `json:"pushName,omitempty"`
	BusinessName      string          `json:"businessName,omitempty"`
	Platform          string          `json:"platform,omitempty"`
	ChatwootEnabled   bool            `json:"chatwootEnabled"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
}

type SessionCreatedResp struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	Token             string          `json:"token"`
	JID               string          `json:"jid,omitempty"`
	Connected         int             `json:"connected"`
	Status            string          `json:"status"`
	Engine            string          `json:"engine,omitempty"`
	PhoneNumberID     string          `json:"phoneNumberId,omitempty"`
	BusinessAccountID string          `json:"businessAccountId,omitempty"`
	Proxy             SessionProxy    `json:"proxy"`
	Settings          SessionSettings `json:"settings"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
	Webhook           *WebhookResp    `json:"webhook,omitempty"`
}

type SessionUpdateReq struct {
	Name               *string          `json:"name,omitempty"`
	Engine             *string          `json:"engine,omitempty"`
	PhoneNumberID      *string          `json:"phoneNumberId,omitempty"`
	AccessToken        *string          `json:"accessToken,omitempty"`
	BusinessAccountID  *string          `json:"businessAccountId,omitempty"`
	AppSecret          *string          `json:"appSecret,omitempty"`
	WebhookVerifyToken *string          `json:"webhookVerifyToken,omitempty"`
	Proxy              *SessionProxy    `json:"proxy,omitempty"`
	Settings           *SessionSettings `json:"settings,omitempty"`
}

type SessionStatusResp struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	JID       string `json:"jid,omitempty"`
	Connected bool   `json:"connected"`
	LoggedIn  bool   `json:"loggedIn"`
	Status    string `json:"status"`
}

type SessionProfileResp struct {
	PushName     string `json:"pushName,omitempty"`
	BusinessName string `json:"businessName,omitempty"`
	Platform     string `json:"platform,omitempty"`
	PictureURL   string `json:"pictureUrl,omitempty"`
	Status       string `json:"status,omitempty"`
}

type PairPhoneReq struct {
	Phone string `json:"phone" validate:"required"`
}

type PairPhoneResp struct {
	PairingCode string `json:"pairingCode"`
}

type UpdateProfileNameReq struct {
	Name string `json:"name" validate:"required"`
}

func SessionToResp(s model.Session, pushName, businessName, platform string) SessionResp {
	return SessionResp{
		ID:                s.ID,
		Name:              s.Name,
		JID:               s.JID,
		Connected:         s.Connected,
		Status:            s.Status,
		Engine:            s.Engine,
		PhoneNumberID:     s.PhoneNumberID,
		BusinessAccountID: s.BusinessAccountID,
		Proxy:             SessionProxy(s.Proxy),
		Settings:          SessionSettings(s.Settings),
		PushName:          pushName,
		BusinessName:      businessName,
		Platform:          platform,
		CreatedAt:         s.CreatedAt,
		UpdatedAt:         s.UpdatedAt,
	}
}
