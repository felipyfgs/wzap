package model

import "time"

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
	RejectCallMsg string `json:"rejectCallMsg,omitempty"`
	ReadMessages  bool   `json:"readMessages"`
	IgnoreGroups  bool   `json:"ignoreGroups"`
	IgnoreStatus  bool   `json:"ignoreStatus"`
}

type Session struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	Token              string          `json:"token,omitempty"`
	JID                string          `json:"jid,omitempty"`
	QRCode             string          `json:"qrCode,omitempty"`
	Connected          int             `json:"connected"`
	Status             string          `json:"status"`
	Engine             string          `json:"engine,omitempty"`
	PhoneNumberID      string          `json:"phoneNumberId,omitempty"`
	AccessToken        string          `json:"accessToken,omitempty"`
	BusinessAccountID  string          `json:"businessAccountId,omitempty"`
	AppSecret          string          `json:"appSecret,omitempty"`
	WebhookVerifyToken string          `json:"webhookVerifyToken,omitempty"`
	Proxy              SessionProxy    `json:"proxy"`
	Settings           SessionSettings `json:"settings"`
	CreatedAt          time.Time       `json:"createdAt"`
	UpdatedAt          time.Time       `json:"updatedAt"`
}
