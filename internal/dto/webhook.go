package dto

type CreateWebhookReq struct {
	URL         string   `json:"URL" validate:"required,url"`
	Secret      string   `json:"Secret,omitempty"`
	Events      []string `json:"Events" validate:"required"`
	NatsEnabled bool     `json:"NatsEnabled"`
}
