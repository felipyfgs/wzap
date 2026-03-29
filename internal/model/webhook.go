package model

import "time"

type Webhook struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionId"`
	URL       string    `json:"url"`
	Secret    string    `json:"secret,omitempty"`
	Events    []string  `json:"events"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
