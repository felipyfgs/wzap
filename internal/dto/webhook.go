package dto

import "wzap/internal/model"

type CreateWebhookReq struct {
	URL         string            `json:"url" validate:"required,url"`
	Secret      string            `json:"secret,omitempty"`
	Events      []model.EventType `json:"events" validate:"required"`
	NatsEnabled bool              `json:"natsEnabled"`
}
