package model

import "time"

type Session struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"userId"`
	Jid       string                 `json:"jid,omitempty"`
	QrCode    string                 `json:"qrCode,omitempty"`
	Connected int                    `json:"connected"`
	Status    string                 `json:"status"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

type SessionCreateReq struct {
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
