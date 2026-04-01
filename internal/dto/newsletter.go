package dto

type CreateNewsletterReq struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	Picture     string `json:"picture"`
}

type NewsletterMessageReq struct {
	NewsletterJID string `json:"newsletterJid" validate:"required"`
	Count         int    `json:"count"`
	BeforeID      int    `json:"beforeId"`
}

type NewsletterSubscribeReq struct {
	NewsletterJID string `json:"newsletterJid" validate:"required"`
}
