package dto

type CreateNewsletterReq struct {
	Name        string `json:"Name" validate:"required"`
	Description string `json:"Description"`
	Picture     string `json:"Picture"`
}

type NewsletterMessageReq struct {
	NewsletterJID string `json:"NewsletterJid" validate:"required"`
	Count         int    `json:"Count"`
	BeforeID      int    `json:"BeforeId"`
}

type NewsletterSubscribeReq struct {
	NewsletterJID string `json:"NewsletterJid" validate:"required"`
}
