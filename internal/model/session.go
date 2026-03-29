package model

import "time"

type Session struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Token     string                 `json:"token,omitempty"`
	JID       string                 `json:"jid,omitempty"`
	QRCode    string                 `json:"qrCode,omitempty"`
	Connected int                    `json:"connected"`
	Status    string                 `json:"status"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}
