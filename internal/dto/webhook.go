package dto

type CreateWebhookReq struct {
	URL         string   `json:"url" validate:"required,url"`
	Secret      string   `json:"secret,omitempty"`
	Events      []string `json:"events" validate:"required"`
	NATSEnabled bool     `json:"natsEnabled"`
}

type UpdateWebhookReq struct {
	URL         *string  `json:"url,omitempty"`
	Secret      *string  `json:"secret,omitempty"`
	Events      []string `json:"events,omitempty"`
	Enabled     *bool    `json:"enabled,omitempty"`
	NATSEnabled *bool    `json:"natsEnabled,omitempty"`
}
