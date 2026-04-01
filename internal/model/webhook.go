package model

import "time"

type Webhook struct {
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
