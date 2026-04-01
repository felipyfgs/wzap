package model

import "time"

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

type Session struct {
	ID        string          `json:"Id"`
	Name      string          `json:"Name"`
	APIKey    string          `json:"ApiKey,omitempty"`
	JID       string          `json:"Jid,omitempty"`
	QRCode    string          `json:"QRCode,omitempty"`
	Connected int             `json:"Connected"`
	Status    string          `json:"Status"`
	Proxy     SessionProxy    `json:"Proxy"`
	Settings  SessionSettings `json:"Settings"`
	CreatedAt time.Time       `json:"CreatedAt"`
	UpdatedAt time.Time       `json:"UpdatedAt"`
}
