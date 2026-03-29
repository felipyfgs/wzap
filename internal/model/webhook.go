package model

import "time"

type Webhook struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	URL       string    `json:"url"`
	Secret    string    `json:"secret,omitempty"`
	Events    []string  `json:"events"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateWebhookReq struct {
	URL    string   `json:"url" validate:"required,url"`
	Secret string   `json:"secret,omitempty"`
	Events []string `json:"events" validate:"required"`
}
