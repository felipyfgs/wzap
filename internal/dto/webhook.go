package dto

type CreateWebhookReq struct {
	URL    string   `json:"url" validate:"required,url"`
	Secret string   `json:"secret,omitempty"`
	Events []string `json:"events" validate:"required"`
}
