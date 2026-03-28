package model

import "time"

type Webhook struct {
	ID        string    `json:"id" db:"id"`
	SessionID string    `json:"session_id" db:"session_id"`
	URL       string    `json:"url" db:"url"`
	Secret    string    `json:"secret,omitempty" db:"secret"`
	Events    []string  `json:"events" db:"events"`
	Enabled   bool      `json:"enabled" db:"enabled"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type CreateWebhookReq struct {
	URL    string   `json:"url" validate:"required,url"`
	Secret string   `json:"secret,omitempty"`
	Events []string `json:"events" validate:"required"`
}
