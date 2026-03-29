package dto

import "wzap/internal/model"

type WebhookCreateInline struct {
	URL    string            `json:"url"`
	Events []model.EventType `json:"events,omitempty"`
}

type SessionCreateReq struct {
	Name     string                `json:"name"`
	APIKey   string                `json:"apiKey,omitempty"`
	Proxy    model.SessionProxy    `json:"proxy,omitempty"`
	Webhook  *WebhookCreateInline  `json:"webhook,omitempty"`
	Settings model.SessionSettings `json:"settings,omitempty"`
}

type SessionCreatedResp struct {
	model.Session
	Webhook *model.Webhook `json:"webhook,omitempty"`
}
