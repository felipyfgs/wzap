package model

import "time"

type SessionStatus string

const (
	StatusInit    SessionStatus = "INIT"
	StatusPending SessionStatus = "PENDING"
	StatusReady   SessionStatus = "READY"
	StatusError   SessionStatus = "ERROR"
	StatusClosed  SessionStatus = "CLOSED"
)

type Session struct {
	ID          string                 `json:"id" db:"id"`
	APIKey      string                 `json:"api_key" db:"api_key"`
	DeviceJID   string                 `json:"device_jid,omitempty" db:"device_jid"`
	Status      SessionStatus          `json:"status" db:"status"`
	IsConnected bool                   `json:"is_connected" db:"is_connected"`
	QRCode      string                 `json:"qr_code,omitempty" db:"qr_code"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

type SessionCreateReq struct {
	ID       string                 `json:"id" validate:"required"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
